package vuln

import (
	"context"
	"time"

	"github.com/quay/zlog"
)

const gcName = `garbage-collection`

// runGCFullPeriodic runs garbage collection until completion, periodically.
func (u *Updater) runGCFullPeriodic() {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/vuln/Updater.runFullGCPeriodic")

	zlog.Info(ctx).Str("full_gc_interval", u.fullGCInterval.String()).Msg("starting periodic full GC")
	t := time.NewTicker(u.fullGCInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			zlog.Info(ctx).Err(ctx.Err()).Msg("stopping periodic full GC")
			return
		case <-t.C:
			zlog.Info(ctx).Msg("full GC started")
			u.runGCFull(ctx)
			zlog.Info(ctx).Msg("full GC completed")
		}
	}
}

// runGCFull runs garbage collection until completion.
func (u *Updater) runGCFull(ctx context.Context) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.runGCFull")

	// Use Lock instead of TryLock to ensure we get the lock
	// and run a full GC.
	ctx, done := u.locker.Lock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Warn(ctx).Err(err).Msg("lock context canceled")
		return
	}

	// Set i to any int64 greater than 0 to start the loop.
	i := int64(1)
	var err error
	for i > 0 {
		select {
		case <-ctx.Done():
			zlog.Error(ctx).Err(ctx.Err()).Msg("performing GC")
			return
		default:
			i, err = u.runGCNoLock(ctx)
			if err != nil {
				zlog.Error(ctx).Err(err).Msg("performing GC")
				return
			}
		}
	}
}

// runGC runs a garbage collection cycle, once.
func (u *Updater) runGC(ctx context.Context) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.runGC")

	// Use TryLock instead of Lock because a GC cycle is already happening.
	ctx, done := u.locker.TryLock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, garbage collection already running")
		return
	}

	_, err := u.runGCNoLock(ctx)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("performing GC")
	}
}

// runGCNoLock runs the actual garbage collection cycle.
// DO NOT CALL THIS UNLESS THE garbage-collection LOCK IS ACQUIRED.
func (u *Updater) runGCNoLock(ctx context.Context) (int64, error) {
	zlog.Info(ctx).Int("retention", u.updateRetention).Msg("GC started")

	i, err := u.store.GC(ctx, u.updateRetention)
	if err != nil {
		return i, err
	}

	zlog.Info(ctx).
		Int64("remaining_ops", i).
		Int("retention", u.updateRetention).
		Msg("GC completed")
	return i, nil
}
