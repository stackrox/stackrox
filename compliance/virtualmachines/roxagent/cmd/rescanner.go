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
// and test on its own. Scanning no longer involves any network call - see
// mappingRefresher - so unlike an earlier version of this type, scans
// themselves are not retried here; only the schedule for the next rescan
// attempt is, after one fails.
type rescanner struct {
	cache           *vsockserver.ReportCache
	hostPath        string
	mappingFilePath string
	interval        time.Duration

	// scanFn defaults to the package scan function; tests override it to
	// inject failures. newTick defaults to time.After; tests substitute a
	// function returning a manually driven channel for precise control
	// over Run's loop.
	scanFn  func(ctx context.Context, hostPath, mappingFilePath string) (*v4.IndexReport, error)
	newTick func(d time.Duration) <-chan time.Time
}

func newRescanner(cache *vsockserver.ReportCache, hostPath, mappingFilePath string, interval time.Duration) *rescanner {
	return &rescanner{
		cache:           cache,
		hostPath:        hostPath,
		mappingFilePath: mappingFilePath,
		interval:        interval,
		scanFn:          scan,
		newTick:         time.After,
	}
}

// Run rescans every r.interval, publishing each successful result to cache.
// If a rescan fails, the next attempt is scheduled sooner than the full
// interval - with exponential backoff, capped at both rescanRetryMaxBackoff
// and r.interval itself, so a retry is never slower than just waiting for
// the next scheduled rescan would be. Blocks until ctx is cancelled.
func (r *rescanner) Run(ctx context.Context) {
	backoff := rescanRetryBaseBackoff
	tick := r.newTick(r.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			log.Info("Starting rescan")
			report, err := r.scanFn(ctx, r.hostPath, r.mappingFilePath)
			if err != nil {
				retryIn := min(backoff, r.interval)
				log.Errorf("Rescan failed: %v; trying again in %v", err, retryIn)
				backoff = min(backoff*2, rescanRetryMaxBackoff)
				tick = r.newTick(retryIn)
				continue
			}
			r.cache.SetReport(report, discoverFacts(r.hostPath))
			log.Infof("Rescan complete, report updated. Num packages: %d", len(report.GetContents().GetPackages()))
			backoff = rescanRetryBaseBackoff
			tick = r.newTick(r.interval)
		}
	}
}
