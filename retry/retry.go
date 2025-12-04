package retry

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

func ExponentialRetry[T any](ctx context.Context, maxRetries uint, baseBackoff time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	_, ok := ctx.Deadline()
	if !ok {
		return zero, errors.New("no deadline set by caller")
	}

	for attempt := uint(0); attempt <= maxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		// if we've exhausted retries, return the last error
		if attempt == maxRetries {
			return zero, err
		}
		backoff := baseBackoff * time.Duration(1<<attempt)
		select {
		case <-time.After(backoff):
			// try again
			continue
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				slog.Info("deadline exceeded")
			} else {
				slog.Info("canceled or timeout")
			}
			return zero, ctx.Err()
		}
	}
	return zero, errors.New("exponential retry failed")
}
