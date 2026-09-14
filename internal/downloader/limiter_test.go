package downloader

import (
	"context"
	"testing"
	"time"
)

// A nil limiter and a zero rate must both be free, or every unlimited
// download would pay for a lock on each 128KB read.
func TestLimiterUnlimitedIsFree(t *testing.T) {
	var nilL *Limiter
	if err := nilL.Wait(context.Background(), 1<<20); err != nil {
		t.Fatalf("nil limiter should not block: %v", err)
	}
	l := NewLimiter(0)
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := l.Wait(context.Background(), 1<<20); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("unlimited limiter took %v", d)
	}
	if l.Rate() != 0 {
		t.Errorf("Rate() = %d, want 0 for unlimited", l.Rate())
	}
}

// The cap is on the total, so the time taken must track the rate.
func TestLimiterThrottles(t *testing.T) {
	const rate = 200 << 10 // 200 KB/s
	l := NewLimiter(rate)
	l.tokens = 0 // start empty rather than with a full burst

	start := time.Now()
	for i := 0; i < 4; i++ {
		if err := l.Wait(context.Background(), 50<<10); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	// 200KB at 200KB/s ≈ 1s. Generous bounds: this asserts throttling
	// happened at roughly the right scale, not stopwatch accuracy.
	if elapsed < 700*time.Millisecond {
		t.Errorf("200KB at 200KB/s took only %v — not throttled", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Errorf("200KB at 200KB/s took %v — far too slow", elapsed)
	}
}

// Changing the limit must apply to transfers already running.
func TestLimiterRateChangesLive(t *testing.T) {
	l := NewLimiter(1 << 10)
	if l.Rate() != 1<<10 {
		t.Fatalf("Rate() = %d", l.Rate())
	}
	l.SetRate(0)
	start := time.Now()
	if err := l.Wait(context.Background(), 10<<20); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("lifting the limit should free waiters immediately, took %v", d)
	}
}

// A read larger than the burst must not deadlock.
func TestLimiterOversizedReadCompletes(t *testing.T) {
	l := NewLimiter(64 << 10)
	l.tokens = 0
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l.Wait(ctx, 1<<20); err != nil {
		t.Fatalf("oversized read deadlocked: %v", err)
	}
}

// Cancelling a download while it waits on the limiter must return promptly.
func TestLimiterRespectsContext(t *testing.T) {
	l := NewLimiter(1) // 1 byte/sec: effectively stalled
	l.tokens = 0
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(80 * time.Millisecond); cancel() }()
	start := time.Now()
	if err := l.Wait(ctx, 1<<20); err == nil {
		t.Fatal("expected the context error")
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("cancel took %v to take effect", d)
	}
}
