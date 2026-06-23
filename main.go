package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maybemaby/sveltekit-fyi/internal"
	"github.com/maybemaby/sveltekit-fyi/migrations"
	"golang.org/x/sync/errgroup"
)

func createLogger(cfg *internal.Config) *slog.Logger {

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	return logger
}

func runMigrations(ctx context.Context, db *sql.DB, cfg *internal.Config) error {
	if cfg.RunMigrations {
		return migrations.RunMigrations(ctx, db)
	}

	return nil
}

func loadS3Config() internal.S3Config {
	return internal.S3Config{
		Region:          os.Getenv("S3_REGION"),
		Bucket:          os.Getenv("S3_BUCKET"),
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		AccessKeyID:     os.Getenv("S3_KEY"),
		SecretAccessKey: os.Getenv("S3_SECRET"),
		UsePathStyle:    os.Getenv("S3_USE_PATH_STYLE") == "true",
	}
}

func setupDb(ctx context.Context, config *internal.Config) (*sql.DB, error) {
	db, err := internal.ConnectDB(ctx)

	if err != nil {
		return nil, err
	}

	err = runMigrations(ctx, db, config)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func runBackup(db *sql.DB, client *internal.S3Client) error {
	backupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err := internal.BackupDB(backupCtx, db, client)

	if err != nil {
		return err
	}

	return nil
}

func runJetStream(ctx context.Context, store *internal.AppStore, client *internal.S3Client, logger *slog.Logger) error {
	processor := internal.NewJetStreamProcessor(store, client)
	processor.SetLogger(logger)

	err := processor.ProcessEvents(ctx)

	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("error processing jetstream events", "error", err)
		return err
	}

	return nil
}

func main() {
	cfg := internal.LoadConfig()
	logger := createLogger(&cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errGroup, ctx := errgroup.WithContext(ctx)

	db, err := setupDb(ctx, &cfg)

	if err != nil {
		logger.Error("failed to set up database", "error", err)
		panic(err)
	}

	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error("failed to close database connection", "error", err)
		}
	}()

	store := internal.NewAppStore(db)
	s3Client, err := internal.NewS3Client(ctx, loadS3Config())

	if err != nil {
		panic(err)
	}

	errGroup.Go(func() error {
		return runJetStream(ctx, store, s3Client, logger)
	})

	errGroup.Go(func() error {
		server := internal.NewServer(ctx, logger)

		finished := make(chan struct{})

		go func() {
			err := server.Start()

			if err != nil {
				logger.Error("error starting http server", "error", err)
			}

			close(finished)
		}()

		select {
		case <-finished:
			logger.Info("http server stopped")
		case <-ctx.Done():
			logger.Info("shutting down http server")
		}

		return nil
	})

	errGroup.Go(func() error {
		err := internal.RunSnapshots(ctx, db, logger)

		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("Snapshotting ran into an error", "error", err)
			return err
		}

		return nil
	})

	if cfg.CloudflareAPIKey != "" && cfg.CloudflareAccountID != "" {
		logger.Info("Cloudflare screenshotting enabled")
		renderer, err := internal.NewCloudflareRenderer(cfg.CloudflareAccountID, cfg.CloudflareAPIKey)

		if err != nil {
			logger.Error("failed to create cloudflare renderer", "error", err)
			panic(err)
		}

		screenshotService := internal.NewScreenshotService(renderer, store, s3Client)
		screenshotService.SetLogger(logger)

		errGroup.Go(func() error {
			err := screenshotService.Run(ctx)

			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}

			return nil
		})
	}

	<-ctx.Done()

	err = errGroup.Wait()
	if err != nil {
		logger.Error("exited with an error", "error", err)
	}

	err = runBackup(db, s3Client)

	if err != nil {
		logger.Error("failed to backup database", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}
