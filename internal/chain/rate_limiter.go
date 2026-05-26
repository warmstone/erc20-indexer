package chain

import (
	"context"
	"sync"
	"time"
)

type RateLimiter struct {
	interval    time.Duration
	mu          sync.Mutex
	nextRequest time.Time
}

func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{interval: interval}
}

func (l *RateLimiter) wait(ctx context.Context) error {
	if l == nil || l.interval <= 0 {
		return nil
	}

	wait := l.reserveWait()
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *RateLimiter) reserveWait() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	next := l.nextRequest
	if next.Before(now) {
		l.nextRequest = now.Add(l.interval)
		return 0
	}
	l.nextRequest = next.Add(l.interval)
	return next.Sub(now)
}
