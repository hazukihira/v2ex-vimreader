package v2ex

import (
	"context"
	"sync"
	"time"
)

// Limiter is a small token bucket shared by all V2EX requests of the process.
type Limiter struct {
	mu       sync.Mutex
	rate     float64
	burst    float64
	tokens   float64
	last     time.Time
	requests int64
}

// NewLimiter creates a token bucket with rate requests per second and burst capacity.
func NewLimiter(rate float64, burst int) *Limiter {
	if rate <= 0 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &Limiter{rate: rate, burst: float64(burst), tokens: float64(burst), last: time.Now()}
}

// Wait blocks until a request token is available or the context is cancelled.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()
		l.tokens += now.Sub(l.last).Seconds() * l.rate
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
		l.last = now
		if l.tokens >= 1 {
			l.tokens--
			l.requests++
			l.mu.Unlock()
			return nil
		}
		wait := time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
		l.mu.Unlock()
		if wait < 20*time.Millisecond {
			wait = 20 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Stats returns the number of requests issued and the current token level.
func (l *Limiter) Stats() (int64, float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.requests, l.tokens
}
