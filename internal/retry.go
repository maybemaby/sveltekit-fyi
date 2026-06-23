package internal

import "time"

// retries func fn up to attempts times, sleeping for sleep duration between attempts. If fn returns nil, retry returns nil. If fn returns an error, retry will return the last error returned by fn after all attempts have been exhausted.
func retry(attempts int, sleep time.Duration, fn func(retryAttempt int) error) error {
	var err error
	for i := range attempts {
		err = fn(i)
		if err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return err
}
