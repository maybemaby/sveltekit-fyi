package main

import (
	"context"
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

	wg.Go(func() { internal.ProcessEvents(ctx, store) })

	<-ctx.Done()

	wg.Wait()
}
