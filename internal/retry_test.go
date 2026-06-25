package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maybemaby/sveltekit-fyi/internal/assert"
)

func TestRetry(t *testing.T) {
	attempts := 3
	sleep := 10 * time.Millisecond

	attemptCount := 0

	err := retry(t.Context(), attempts, sleep, func(retryAttempt int) error {
		attemptCount++

		return nil
	})

	assert.Nil(t, err)

	assert.Equal(t, attemptCount, 1)
}

func TestRetryExhausted(t *testing.T) {
	attempts := 3
	sleep := 10 * time.Millisecond

	attemptCount := 0

	err := retry(t.Context(), attempts, sleep, func(retryAttempt int) error {
		attemptCount++
		return errors.New("error")
	})

	assert.NotNil(t, err)
	assert.Equal(t, attemptCount, 3)
}

func TestRetryContextCancelled(t *testing.T) {
	attempts := 3
	sleep := 10 * time.Millisecond

	attemptCount := 0

	ctx, cancel := context.WithCancel(t.Context())

	err := retry(ctx, attempts, sleep, func(retryAttempt int) error {
		attemptCount++

		if attemptCount == 2 {
			cancel()
		}

		return errors.New("error")
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, attemptCount, 2)
}

func TestRetryExitEarly(t *testing.T) {
	attempts := 3
	sleep := 10 * time.Millisecond

	attemptCount := 0

	err := retry(t.Context(), attempts, sleep, func(retryAttempt int) error {
		attemptCount++

		if attemptCount == 2 {
			return ErrExitEarly
		}

		return errors.New("error")
	})

	assert.ErrorIs(t, err, ErrExitEarly)
	assert.Equal(t, attemptCount, 2)
}

func TestRetryExitEarlyJoined(t *testing.T) {
	attempts := 3
	sleep := 10 * time.Millisecond

	attemptCount := 0

	err := retry(t.Context(), attempts, sleep, func(retryAttempt int) error {
		attemptCount++

		if attemptCount == 2 {
			return errors.Join(errors.New("some error"), ErrExitEarly)
		}

		return errors.New("error")
	})

	assert.ErrorIs(t, err, ErrExitEarly)
	assert.Equal(t, attemptCount, 2)
}
