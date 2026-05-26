package chain

import (
	"context"
	"time"
)

const maxBackoff = 30 * time.Second

func sleepBackOff(ctx context.Context, base time.Duration, attempt int) error {
	if base <= 0 {
		return nil
	}
	wait := base
	for i := 0; i < attempt; i++ {
		wait *= 2
		if wait >= maxBackoff {
			wait = maxBackoff
			break
		}
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
