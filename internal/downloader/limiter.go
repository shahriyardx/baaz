package downloader

import (
	"context"
	"sync"
	"time"
)

// Limiter is a token bucket shared by every segment of every job, so the
// cap is on total throughput rather than per connection — eight segments at
// 1 MB/s each is not a 1 MB/s limit.
//
// Written by hand rather than pulling in golang.org/x/time/rate: this is the
// only thing the project would need it for, and go.mod has no dependencies.
type Limiter struct {
	mu       sync.Mutex
	rate     int64 // bytes per second; 0 disables the limit entirely
	burst    int64
	tokens   float64
	last     time.Time
	disabled bool
}

// NewLimiter returns a limiter capped at bytesPerSec. Zero or less means no
// limit, and Wait becomes a cheap no-op.
func NewLimiter(bytesPerSec int64) *Limiter {
	l := &Limiter{}
	l.SetRate(bytesPerSec)
	return l
}

// SetRate changes the cap live. Existing transfers pick it up on their next
// read, so a change in the UI takes effect without restarting anything.
func (l *Limiter) SetRate(bytesPerSec int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if bytesPerSec <= 0 {
		l.disabled = true
		l.rate = 0
		return
	}
	l.disabled = false
	l.rate = bytesPerSec
	// One second of burst: enough to stay smooth across the 128KB read size
	// without letting a long idle period bank a huge spike.
	l.burst = bytesPerSec
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}
	if l.last.IsZero() {
		l.last = time.Now()
	}
}

func (l *Limiter) Rate() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.disabled {
		return 0
	}
	return l.rate
}

// Wait blocks until n bytes may be transferred, or the context ends.
//
// It returns as soon as the bucket can cover n, refilling at the configured
// rate. n larger than the burst is allowed through once the bucket is full,
// so an oversized read cannot deadlock.
func (l *Limiter) Wait(ctx context.Context, n int) error {
	if l == nil || n <= 0 {
		return nil
	}
	for {
		l.mu.Lock()
		if l.disabled {
			l.mu.Unlock()
			return nil
		}
		now := time.Now()
		if l.last.IsZero() {
			l.last = now
		}
		l.tokens += now.Sub(l.last).Seconds() * float64(l.rate)
		l.last = now
		if l.tokens > float64(l.burst) {
			l.tokens = float64(l.burst)
		}
		need := float64(n)
		if need > float64(l.burst) {
			need = float64(l.burst) // oversized read: wait for a full bucket
		}
		if l.tokens >= need {
			l.tokens -= float64(n)
			l.mu.Unlock()
			return nil
		}
		deficit := need - l.tokens
		delay := time.Duration(deficit / float64(l.rate) * float64(time.Second))
		l.mu.Unlock()

		if delay < time.Millisecond {
			delay = time.Millisecond
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
