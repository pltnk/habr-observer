package updater

import (
	"context"
	"sync"
	"testing"
	"time"
)

// --- fakes -------------------------------------------------------------------

// fakeUpdater records each UpdateAllFeeds call and signals on a channel so tests
// can await a precise number of cycles without sleeping.
type fakeUpdater struct {
	mu        sync.Mutex
	calls     int
	deadlines int // calls whose ctx carried a deadline
	signal    chan struct{}
}

func (u *fakeUpdater) UpdateAllFeeds(ctx context.Context) error {
	u.mu.Lock()
	u.calls++
	if _, ok := ctx.Deadline(); ok {
		u.deadlines++
	}
	u.mu.Unlock()
	if u.signal != nil {
		u.signal <- struct{}{}
	}
	return nil
}

func (u *fakeUpdater) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.calls
}

func (u *fakeUpdater) deadlineCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.deadlines
}

// blockingUpdater's cycle never returns until release is closed, simulating a
// feed pipeline wedged past its deadline (a dependency that ignores ctx).
type blockingUpdater struct{ release chan struct{} }

func (u *blockingUpdater) UpdateAllFeeds(context.Context) error {
	<-u.release
	return nil
}

// --- tests -------------------------------------------------------------------

func TestRunner_RunsImmediatelyThenOnInterval(t *testing.T) {
	t.Parallel()

	upd := &fakeUpdater{signal: make(chan struct{}, 64)}
	// A long stall timeout so the watchdog never fires during this test.
	r := NewRunner(upd, 5*time.Millisecond, 50*time.Millisecond, time.Hour, quietLogger(), func() {})

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
		case <-upd.signal:
		case <-time.After(2 * time.Second):
			t.Fatalf("cycle %d did not run", i+1)
		}
	}
	cancel()
	<-done // Run has returned, so the counts below are stable

	if upd.count() < 2 {
		t.Fatalf("cycles = %d, want >= 2", upd.count())
	}
	// Every cycle ran under a per-cycle deadline.
	if got := upd.deadlineCount(); got != upd.count() {
		t.Fatalf("cycles with a deadline = %d, want %d (all of them)", got, upd.count())
	}
}

func TestRunner_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	upd := &fakeUpdater{}
	r := NewRunner(upd, time.Hour, time.Hour, time.Hour, quietLogger(), func() {})

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
	if got := upd.count(); got != 1 {
		t.Fatalf("cycles = %d, want exactly 1 (the immediate run)", got)
	}
}

// TestRunner_ExitsOnStall proves the watchdog fires onStall when a cycle wedges
// past the stall timeout — the failure mode a restart-on-exit supervisor cannot
// catch, since the process stays alive.
func TestRunner_ExitsOnStall(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	upd := &blockingUpdater{release: release}

	var once sync.Once
	stalled := make(chan struct{})
	onStall := func() { once.Do(func() { close(stalled) }) }

	// Tiny stall timeout so the wedged first cycle trips the watchdog quickly.
	r := NewRunner(upd, time.Hour, time.Hour, 20*time.Millisecond, quietLogger(), onStall)

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
