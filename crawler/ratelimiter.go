package crawler

import (
	"context"
	"time"
)

// RateLimiter controls request rate using channels and tickers
type RateLimiter struct {
	ticker *time.Ticker
	tokens chan struct{}
}

// NewRateLimiter creates a rate limiter with specified requests per second
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	interval := time.Second / time.Duration(requestsPerSecond)
	rl := &RateLimiter{
		ticker: time.NewTicker(interval),
		tokens: make(chan struct{}, requestsPerSecond),
	}

	// Fill initial tokens
	for i := 0; i < requestsPerSecond; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens continuously
	go func() {
		for range rl.ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Channel full, skip
			}
		}
	}()

	return rl
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) {
	select {
	case <-rl.tokens:
		return
	case <-ctx.Done():
		return
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	close(rl.tokens)
}
