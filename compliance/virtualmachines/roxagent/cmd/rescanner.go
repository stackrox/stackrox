package cmd

import (
	"context"
	"fmt"
	"sync/atomic"
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
	// no-op).
	scanFn  func(ctx context.Context, hostPath, mappingFilePath string) (*v4.IndexReport, error)
	factsFn func(hostPath string) map[string]string

	// timer is the next scan deadline. OnMappingChanged Reset(0) fires it
	// immediately; a Reset(0) during scanOnce is received on the next
	// select because t.C is unbuffered (Go 1.23+). Nil after Run returns
	// so a later Reset is a no-op.
	timer atomic.Pointer[time.Timer]
}

func newRescanner(cache *vsockserver.ReportCache, hostPath string, provider vsockserver.MappingProvider, interval time.Duration) *rescanner {
	r := &rescanner{
		cache:    cache,
		hostPath: hostPath,
		provider: provider,
		interval: interval,
		scanFn:   scan,
		factsFn:  discoverFacts,
	}
	r.timer.Store(time.NewTimer(interval))
	return r
}

// OnMappingChanged is the updater's onChange callback: it schedules an
// immediate scan attempt.
func (r *rescanner) OnMappingChanged() {
	t := r.timer.Load()
	if t == nil {
		return
	}
	t.Reset(0)
	log.Info("Mapping changed, scheduling immediate rescan")
}

// Run rescans every r.interval, publishing each successful result to cache.
// If a rescan fails, the next attempt is scheduled sooner than the full
// interval - with exponential backoff, capped at both rescanRetryMaxBackoff
// and r.interval itself, so a retry is never slower than just waiting for
// the next scheduled rescan would be. Blocks until ctx is cancelled.
func (r *rescanner) Run(ctx context.Context) {
	t := r.timer.Load()
	defer func() {
		r.timer.Store(nil)
		t.Stop()
	}()

	backoff := rescanRetryBaseBackoff
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Schedule the default next scan before scanOnce so a
			// mapping change during the scan (Reset(0)) is not
			// overwritten by the post-scan Reset.
			t.Reset(r.interval)
			if err := r.scanOnce(ctx); err != nil {
				retryIn := min(backoff, r.interval)
				log.Errorf("Rescan failed: %v; trying again in %v", err, retryIn)
				backoff = min(backoff*2, rescanRetryMaxBackoff)
				t.Reset(retryIn)
				continue
			}
			backoff = rescanRetryBaseBackoff
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

	// Snapshot the hash before scanFn: a URL-backed provider has no busy
	// gate, so a background fetch can replace the live mapping mid-scan.
	mappingHash := r.provider.Hash()
	log.Infof("Starting rescan (mapping hash=%s)", mappingHash)
	report, err := r.scanFn(ctx, r.hostPath, path)
	if err != nil {
		return err
	}

	r.cache.SetReport(report, r.factsFn(r.hostPath), mappingHash)
	log.Infof("Rescan complete, report updated (token=%s, packages=%d)", r.cache.Token(), len(report.GetContents().GetPackages()))
	return nil
}

// runAsync starts Run in a goroutine and returns a channel that is closed
// when Run returns. Callers that cancel ctx should wait on stopped so
// assertions do not race Run's last scanOnce.
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
