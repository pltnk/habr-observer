package updater

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"
)

// updater runs one update cycle. *Service satisfies it; the Runner depends on the
// interface so the loop can be tested without the feed pipeline.
type updater interface {
	UpdateAllFeeds(ctx context.Context) error
}

// Runner drives the update loop: an immediate first cycle, then one every
// interval, each bounded by deadline. Cycles run sequentially, so a slow one
// delays the next rather than overlapping it.
//
// A stall watchdog catches a cycle wedged past its deadline (a dependency
// ignoring cancellation): the worker stays alive but idle, invisible to a
// restart-on-exit supervisor. If no cycle completes within stallTimeout, onStall
// fires — in production, exiting so the supervisor restarts it.
type Runner struct {
	svc          updater
	interval     time.Duration
	deadline     time.Duration
	stallTimeout time.Duration
	log          *slog.Logger
	onStall      func()
}

// NewRunner returns a [Runner]: interval is the cadence between cycles, deadline
// each cycle's hard timeout, and stallTimeout the longest gap between completed
// cycles before onStall fires. If log is nil, [slog.Default] is used; if onStall
// is nil, it exits the process with status 1.
func NewRunner(svc updater, interval, deadline, stallTimeout time.Duration, log *slog.Logger, onStall func()) *Runner {
	if log == nil {
		log = slog.Default()
	}
	if onStall == nil {
		onStall = func() { os.Exit(1) }
	}
	return &Runner{
		svc:          svc,
		interval:     interval,
		deadline:     deadline,
		stallTimeout: stallTimeout,
		log:          log,
		onStall:      onStall,
	}
}

// Run executes update cycles until ctx is canceled. A healthy cycle logs nothing;
// a deadline overrun warns, and a cycle wedged past stallTimeout trips the
// watchdog.
func (r *Runner) Run(ctx context.Context) {
	// Arm the watchdog; each completed cycle resets it, so it fires only on a
	// stall. Stop it on return so shutdown never trips it.
	watchdog := time.AfterFunc(r.stallTimeout, r.stall)
	defer watchdog.Stop()

	runCycle := func() {
		cctx, cancel := context.WithTimeout(ctx, r.deadline)
		defer cancel()

		_ = r.svc.UpdateAllFeeds(cctx) // per-feed failures are logged inside
		watchdog.Reset(r.stallTimeout) // a completed cycle proves liveness

		// Warn once if the cycle overran its own deadline — but not on shutdown
		// (parent canceled) or a healthy cycle; per-feed failures are logged at
		// their source. cctx.Err() is read before the deferred cancel, so it still
		// reflects the deadline.
		if ctx.Err() == nil && errors.Is(cctx.Err(), context.DeadlineExceeded) {
			r.log.Warn("updater: cycle exceeded its deadline")
		}
	}

	runCycle() // run immediately on startup

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCycle()
		}
	}
}

// stall is the watchdog callback, run when no cycle completes within stallTimeout
// (a cycle wedged past its deadline). It logs the stall and invokes onStall — in
// production, exiting so the supervisor restarts the worker.
func (r *Runner) stall() {
	r.log.Error("updater: stalled; exiting to be restarted", "stall_timeout", r.stallTimeout)
	r.onStall()
}
