package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/maybemaby/sveltekit-fyi/internal"
	"github.com/maybemaby/sveltekit-fyi/migrations"
)

func createLogger() *slog.Logger {
	logLevel := slog.LevelDebug

	envLevel := os.Getenv("LOG_LEVEL")

	switch envLevel {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	return logger
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	runEmbedded := os.Getenv("RUN_MIGRATIONS")

	if runEmbedded == "true" {
		return migrations.RunMigrations(ctx, db)
	}

	return nil
}

func main() {
	logger := createLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	wg := &sync.WaitGroup{}

	db, err := internal.ConnectDB(ctx)

	if err != nil {
		panic(err)
	}

	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error("failed to close database connection", "error", err)
		}
	}()

	err = runMigrations(ctx, db)

	if err != nil {
		logger.Error("failed to run migrations", "error", err)
		panic(err)
	}

	store := internal.NewAppStore(db)

	wg.Go(func() {
		processor := internal.NewJetStreamProcessor(store)
		processor.SetLogger(logger)
		jetstreamErr := processor.ProcessEvents(ctx, store)

		if jetstreamErr != nil {
			logger.Error("error processing jetstream events", "error", jetstreamErr)
		}

		// We want the process to exit if the jetstream connection is lost, so we call stop() here to trigger a shutdown of the app
		stop()
	})

	wg.Go(func() {
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
			stop()
		case <-ctx.Done():
			logger.Info("shutting down http server")
		}
	})

	<-ctx.Done()

	wg.Wait()
}
