package vuln

import (
	"context"
	"log/slog"
	"time"
)

const gcName = `garbage-collection`

// runGCFullPeriodic runs garbage collection until completion, periodically.
func (u *Updater) runGCFullPeriodic() {
	ctx := u.ctx

	slog.InfoContext(ctx, "starting periodic full GC", "full_gc_interval", u.fullGCInterval.String())
	t := time.NewTicker(u.fullGCInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "stopping periodic full GC", "reason", ctx.Err())
			return
		case <-t.C:
			slog.InfoContext(ctx, "full GC started")
			u.runGCFull(ctx)
			slog.InfoContext(ctx, "full GC completed")
		}
	}
}

// runGCFull runs garbage collection until completion.
func (u *Updater) runGCFull(ctx context.Context) {
	// Use Lock instead of TryLock to ensure we get the lock
	// and run a full GC.
	ctx, done := u.locker.Lock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		slog.WarnContext(ctx, "lock context canceled", "reason", err)
		return
	}

	// Set i to any int64 greater than 0 to start the loop.
	i := int64(1)
	var err error
	for i > 0 {
		select {
		case <-ctx.Done():
			slog.ErrorContext(ctx, "performing GC", "reason", ctx.Err())
			return
		default:
			i, err = u.runGCNoLock(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "performing GC", "reason", err)
				return
			}
		}
	}
}

// runGC runs a garbage collection cycle, once.
func (u *Updater) runGC(ctx context.Context) {
	// Use TryLock instead of Lock because a GC cycle is already happening.
	ctx, done := u.locker.TryLock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		slog.DebugContext(ctx, "lock context canceled, garbage collection already running", "reason", err)
		return
	}

	_, err := u.runGCNoLock(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "performing GC", "reason", err)
	}
}

// runGCNoLock runs the actual garbage collection cycle.
// DO NOT CALL THIS UNLESS THE garbage-collection LOCK IS ACQUIRED.
func (u *Updater) runGCNoLock(ctx context.Context) (int64, error) {
	slog.InfoContext(ctx, "GC started", "retention", u.updateRetention)

	i, err := u.store.GC(ctx, u.updateRetention)
	if err != nil {
		return i, err
	}

	slog.InfoContext(ctx, "GC completed", "remaining_ops", i, "retention", u.updateRetention)
	return i, nil
}
