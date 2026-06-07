package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/maybemaby/sveltekit-fyi/internal"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	wg := &sync.WaitGroup{}

	db, err := internal.ConnectDB(ctx)

	if err != nil {
		panic(err)
	}

	store := internal.NewAppStore(db)

	wg.Go(func() {
		jetstreamErr := internal.ProcessEvents(ctx, store)

		if jetstreamErr != nil {
			fmt.Printf("error processing jetstream events: %v\n", jetstreamErr)
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
				fmt.Printf("error starting http server: %v\n", err)
			}

			close(finished)
		}()

		select {
		case <-finished:
			fmt.Println("http server stopped")
			stop()
		case <-ctx.Done():
			fmt.Println("shutting down http server")
		}
	})

	<-ctx.Done()

	wg.Wait()
}
