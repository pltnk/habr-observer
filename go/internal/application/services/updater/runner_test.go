package updater

import (
	"context"
	"sync"
	"testing"
	"time"
)

// --- fakes -------------------------------------------------------------------

// fakeCycler records each UpdateAllFeeds call and signals on a channel so tests
// can await a precise number of cycles without sleeping.
type fakeCycler struct {
	mu        sync.Mutex
	calls     int
	deadlines int // calls whose ctx carried a deadline
	signal    chan struct{}
}

func (c *fakeCycler) UpdateAllFeeds(ctx context.Context) error {
	c.mu.Lock()
	c.calls++
	if _, ok := ctx.Deadline(); ok {
		c.deadlines++
	}
	c.mu.Unlock()
	if c.signal != nil {
		c.signal <- struct{}{}
	}
	return nil
}

func (c *fakeCycler) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func (c *fakeCycler) deadlineCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deadlines
}

// blockingCycler's cycle never returns until release is closed, simulating a
// feed pipeline wedged past its deadline (a dependency that ignores ctx).
type blockingCycler struct{ release chan struct{} }

func (c *blockingCycler) UpdateAllFeeds(context.Context) error {
	<-c.release
	return nil
}

// --- tests -------------------------------------------------------------------

func TestRunner_RunsImmediatelyThenOnInterval(t *testing.T) {
	t.Parallel()

	cyc := &fakeCycler{signal: make(chan struct{}, 64)}
	// A long stall timeout so the watchdog never fires during this test.
	r := NewRunner(cyc, 5*time.Millisecond, 50*time.Millisecond, time.Hour, quietLogger(), func() {})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	// First cycle is immediate; the second proves the interval ticker fired.
	for i := range 2 {
		select {
		case <-cyc.signal:
		case <-time.After(2 * time.Second):
			t.Fatalf("cycle %d did not run", i+1)
		}
	}
	cancel()
	<-done // Run has returned, so the counts below are stable

	if cyc.count() < 2 {
		t.Fatalf("cycles = %d, want >= 2", cyc.count())
	}
	// Every cycle ran under a per-cycle deadline.
	if got := cyc.deadlineCount(); got != cyc.count() {
		t.Fatalf("cycles with a deadline = %d, want %d (all of them)", got, cyc.count())
	}
}

func TestRunner_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	cyc := &fakeCycler{}
	r := NewRunner(cyc, time.Hour, time.Hour, time.Hour, quietLogger(), func() {})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before Run starts

	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
	// The immediate first cycle still ran; the long interval never fired.
	if got := cyc.count(); got != 1 {
		t.Fatalf("cycles = %d, want exactly 1 (the immediate run)", got)
	}
}

// TestRunner_ExitsOnStall proves the watchdog fires onStall when a cycle wedges
// past the stall timeout — the failure mode a restart-on-exit supervisor cannot
// catch, since the process stays alive.
func TestRunner_ExitsOnStall(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	cyc := &blockingCycler{release: release}

	var once sync.Once
	stalled := make(chan struct{})
	onStall := func() { once.Do(func() { close(stalled) }) }

	// Tiny stall timeout so the wedged first cycle trips the watchdog quickly.
	r := NewRunner(cyc, time.Hour, time.Hour, 20*time.Millisecond, quietLogger(), onStall)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	select {
	case <-stalled:
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog did not fire while a cycle was wedged")
	}

	// Unwedge the cycle and stop the runner so its goroutine exits.
	cancel()
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
