package poller

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newTestPoller() *Poller {
	return &Poller{
		// influx not needed for retry tests
	}
}

func TestWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	p := newTestPoller()
	calls := 0

	err := p.withRetry(context.Background(), func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_SuccessAfterRetry(t *testing.T) {
	p := newTestPoller()
	calls := 0
	errTemp := errors.New("temporary failure")

	err := p.withRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errTemp
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_AllRetriesExhausted(t *testing.T) {
	p := newTestPoller()
	errPersistent := errors.New("persistent failure")

	err := p.withRetry(context.Background(), func() error {
		return errPersistent
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errPersistent) {
		t.Fatalf("expected wrapped persistent error, got %v", err)
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	p := newTestPoller()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately so the backoff select picks it up.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := p.withRetry(ctx, func() error {
		return errors.New("always fail")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
