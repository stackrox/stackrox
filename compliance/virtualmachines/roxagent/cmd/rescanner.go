package cmd

import (
	"context"
	"fmt"
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
// and test on its own. Scanning does not make a network call; provider
// keeps the mapping file fresh independently, on its own schedule, so only
// the schedule for the next rescan attempt is retried here, not the scan
// itself.
type rescanner struct {
	cache    *vsockserver.ReportCache
	hostPath string
	provider vsockserver.MappingProvider
	interval time.Duration

	// scanFn defaults to the package scan function; tests override it to
	// inject failures. factsFn defaults to the package discoverFacts
	// function; tests override it to avoid exercising the real
	// filesystem, since discoverFacts otherwise reads real host paths
	// (e.g. hostPath="" resolves to "/etc/pki/entitlement" et al., not a
	// no-op). newDelay defaults to time.After (a one-shot timer); tests
	// substitute a function returning a manually driven channel for
	// precise control over Run's loop.
	scanFn   func(ctx context.Context, hostPath, mappingFilePath string) (*v4.IndexReport, error)
	factsFn  func(hostPath string) map[string]string
	newDelay func(d time.Duration) <-chan time.Time

	// wake lets OnMappingChanged nudge Run into scanning immediately
	// instead of waiting out the rest of the current interval. Buffered
	// so a callback that fires while Run is mid-scan is not lost.
	wake chan struct{}
}

func newRescanner(cache *vsockserver.ReportCache, hostPath string, provider vsockserver.MappingProvider, interval time.Duration) *rescanner {
	return &rescanner{
		cache:    cache,
		hostPath: hostPath,
		provider: provider,
		interval: interval,
		scanFn:   scan,
		factsFn:  discoverFacts,
		newDelay: time.After,
		wake:     make(chan struct{}, 1),
	}
}

// OnMappingChanged is the updater's onChange callback: it schedules an
// immediate scan attempt by waking Run's loop, coalescing with any
// already-pending wake so a burst of changes only triggers one extra scan.
func (r *rescanner) OnMappingChanged() {
	select {
	case r.wake <- struct{}{}:
		log.Info("Mapping changed, scheduling immediate rescan")
	default:
		log.Debug("Mapping changed, rescan already scheduled")
	}
}

// Run rescans every r.interval, publishing each successful result to cache.
// If a rescan fails, the next attempt is scheduled sooner than the full
// interval - with exponential backoff, capped at both rescanRetryMaxBackoff
// and r.interval itself, so a retry is never slower than just waiting for
// the next scheduled rescan would be. Blocks until ctx is cancelled.
func (r *rescanner) Run(ctx context.Context) {
	backoff := rescanRetryBaseBackoff
	delay := r.newDelay(r.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.wake:
			if err := r.scanOnce(ctx); err != nil {
				log.Errorf("Rescan triggered by mapping change failed: %v", err)
			}
		case <-delay:
			if err := r.scanOnce(ctx); err != nil {
				retryIn := min(backoff, r.interval)
				log.Errorf("Rescan failed: %v; trying again in %v", err, retryIn)
				backoff = min(backoff*2, rescanRetryMaxBackoff)
				delay = r.newDelay(retryIn)
				continue
			}
			backoff = rescanRetryBaseBackoff
			delay = r.newDelay(r.interval)
		}
	}
}

// scanOnce keeps the mapping file unchanged while this scan reads it, then
// applies any pending Sync so a follow-up scan can use the new mapping.
func (r *rescanner) scanOnce(ctx context.Context) error {
	if !r.provider.Ready() {
		log.Info("Skipping rescan: repository-to-CPE mapping not yet available")
		return nil
	}

	gate, hasGate := r.provider.(vsockserver.ScanBusyGate)
	if hasGate {
		gate.MarkScanBusy()
		defer gate.MarkScanIdleAndApplyPending()
	}

	path, err := r.provider.Path()
	if err != nil {
		return fmt.Errorf("resolving mapping file path: %w", err)
	}

	log.Infof("Starting rescan (mapping hash=%s)", r.provider.Hash())
	report, err := r.scanFn(ctx, r.hostPath, path)
	if err != nil {
		return err
	}

	r.cache.SetReport(report, r.factsFn(r.hostPath), r.provider.Hash())
	log.Infof("Rescan complete, report updated (token=%s, packages=%d)", r.cache.Token(), len(report.GetContents().GetPackages()))
	return nil
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
