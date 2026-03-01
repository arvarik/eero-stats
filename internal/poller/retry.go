package poller

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// withRetry executes op with exponential backoff, retrying up to maxRetries
// times. It respects context cancellation between attempts, returning ctx.Err()
// if the context is cancelled while waiting for a retry.
func (p *Poller) withRetry(ctx context.Context, op func() error) error {
	const maxRetries = 3
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err = op()
		if err == nil {
			return nil
		}

		if attempt < maxRetries-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
			slog.Warn("API call failed, retrying",
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"backoff", backoff,
				"error", err,
			)

			t := time.NewTimer(backoff)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			}
		}
	}
	return fmt.Errorf("after %d attempts, last error: %w", maxRetries, err)
}
