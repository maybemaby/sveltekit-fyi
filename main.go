package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/maybemaby/sveltekit-fyi/internal"
)

func createLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return logger
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
		server := internal.NewServer(ctx)

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
