package cmd

import (
	"context"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

// Retry policy for rescanner.Run's schedule: how soon to try again after a
// rescan fails, instead of leaving Sensor with a stale cached report for up
// to the full (potentially hours-long) rescan interval.
const (
	rescanRetryBaseBackoff = 2 * time.Minute
	rescanRetryMaxBackoff  = 30 * time.Minute
)

// rescanner owns the scan-and-cache-update concern: periodically rescanning
// the VM filesystem and publishing results to cache, independent of how the
// cached report is served over VSOCK. Kept separate from Server/CARefresher
// wiring in runServe so its retry/backoff policy is easier to reason about
// and test on its own. Scanning does not make a network call; the downloader
// built by newMappingDownloader keeps the mapping file fresh independently,
// on its own schedule, so only the schedule for the next rescan attempt is
// retried here, not the scan itself.
type rescanner struct {
	cache           *vsockserver.ReportCache
	hostPath        string
	mappingFilePath string
	interval        time.Duration

	// reactiveCh, when set by the caller (runServe wires in the RPM
	// watcher's Triggered() channel), lets Run start a rescan immediately
	// instead of waiting for the next periodic tick. Left nil when
	// reactive scanning is unavailable (e.g. the watcher failed to start):
	// a nil channel blocks forever in Run's select, so that case simply
	// never fires and Run falls back to periodic-only scanning.
	reactiveCh <-chan struct{}

	// scanFn defaults to scanWithDiagnostics, which returns the facts
	// alongside the report from the same DiscoveredData probe used for
	// diagnostics logging, rather than making Run probe the filesystem a
	// second time for them. Tests override scanFn to inject failures and
	// avoid exercising the real filesystem, since hostPath="" otherwise
	// resolves to real absolute host paths (e.g. "/etc/pki/entitlement"),
	// not a no-op. newDelay defaults to time.After (a one-shot timer);
	// tests substitute a function returning a manually driven channel for
	// precise control over Run's loop.
	scanFn   func(ctx context.Context, hostPath, mappingFilePath, trigger string) (*v4.IndexReport, map[string]string, error)
	newDelay func(d time.Duration) <-chan time.Time
}

func newRescanner(cache *vsockserver.ReportCache, hostPath, mappingFilePath string, interval time.Duration) *rescanner {
	return &rescanner{
		cache:           cache,
		hostPath:        hostPath,
		mappingFilePath: mappingFilePath,
		interval:        interval,
		scanFn:          scanWithDiagnostics,
		newDelay:        time.After,
	}
}

// Run rescans every r.interval, or immediately whenever r.reactiveCh fires,
// publishing each successful result to cache. Both triggers share this one
// loop and code path — reactive and scheduled rescans differ only in the
// "trigger" fact stamped onto the result (see scanTriggerScheduled/
// scanTriggerReactive) — so a reactive rescan also resets delay, meaning a
// routine rescan never immediately follows one that just refreshed the same
// data. If a rescan fails, the next attempt is scheduled sooner than the
// full interval - with exponential backoff, capped at both
// rescanRetryMaxBackoff and r.interval itself, so a retry is never slower
// than just waiting for the next scheduled rescan would be. Blocks until
// ctx is cancelled.
func (r *rescanner) Run(ctx context.Context) {
	backoff := rescanRetryBaseBackoff
	delay := r.newDelay(r.interval)
	for {
		var trigger string
		select {
		case <-ctx.Done():
			return
		case <-delay:
			trigger = scanTriggerScheduled
		case <-r.reactiveCh:
			trigger = scanTriggerReactive
		}

		log.Infof("Starting %s rescan", trigger)
		report, facts, err := r.scanFn(ctx, r.hostPath, r.mappingFilePath, trigger)
		if err != nil {
			retryIn := min(backoff, r.interval)
			log.Errorf("%s rescan failed: %v; trying again in %v", trigger, err, retryIn)
			backoff = min(backoff*2, rescanRetryMaxBackoff)
			delay = r.newDelay(retryIn)
			continue
		}
		r.cache.SetReport(report, facts)
		log.Infof("Rescan complete, report updated. Num packages: %d", len(report.GetContents().GetPackages()))
		backoff = rescanRetryBaseBackoff
		delay = r.newDelay(r.interval)
	}
}

// runAsync starts Run in a goroutine and returns a channel that is closed
// when Run returns. Callers that cancel ctx should wait on stopped before
// tearing down anything Run still observes (e.g. an injected tick channel).
func (r *rescanner) runAsync(ctx context.Context) (stopped <-chan struct{}) {
	return startRun(ctx, r.Run)
}

// startRun starts run in a goroutine and returns a channel that is closed
// when run returns.
func startRun(ctx context.Context, run func(context.Context)) (stopped <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		run(ctx)
	}()
	return ch
}
