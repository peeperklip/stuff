package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExponentialRetry_SucceedsImmediately(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	val, err := ExponentialRetry[int](ctx, 3, 10*time.Millisecond, func() (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %v", val)
	}
}

func TestExponentialRetry_SucceedsAfterRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	attempts := 0
	fn := func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("fail")
		}
		return 7, nil
	}

	val, err := ExponentialRetry[int](ctx, 5, 5*time.Millisecond, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 7 {
		t.Fatalf("expected 7, got %v", val)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestExponentialRetry_ExhaustRetries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	fn := func() (int, error) {
		return 0, errors.New("permanent failure")
	}

	_, err := ExponentialRetry[int](ctx, 2, 1*time.Millisecond, fn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "permanent failure" {
		t.Fatalf("expected last error 'permanent failure', got %v", err)
	}
}

func TestExponentialRetry_NoDeadline(t *testing.T) {
	// context without deadline should be rejected
	_, err := ExponentialRetry[int](context.Background(), 2, 1*time.Millisecond, func() (int, error) {
		return 0, nil
	})
	if err == nil {
		t.Fatalf("expected error when no deadline is set")
	}
	if err.Error() != "no deadline set by caller" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestExponentialRetry_ContextDeadlineExceeded(t *testing.T) {
	// make deadline very short and backoff long so ctx.Done() fires during backoff
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	fn := func() (int, error) {
		return 0, errors.New("transient")
	}

	_, err := ExponentialRetry[int](ctx, 5, 100*time.Millisecond, fn)
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}
