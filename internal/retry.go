package internal

import (
	"context"
	"errors"
	"time"
)

var ErrExitEarly = errors.New("exit early signal received")

// retries func fn up to attempts times, sleeping for sleep duration between attempts. If fn returns nil, retry returns nil. If fn returns an error, retry will return the last error returned by fn after all attempts have been exhausted.
func retry(ctx context.Context, attempts int, sleep time.Duration, fn func(retryAttempt int) error) error {
	var err error
	for i := range attempts {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err = fn(i)
			if err == nil {
				return nil
			}

			if errors.Is(err, ErrExitEarly) {
				return err
			}

			time.Sleep(sleep)
		}

	}
	return err
}
