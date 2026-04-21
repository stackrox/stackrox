package relay

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/time/rate"
)

var log = logging.LoggerForModule()

// IndexReportStream manages report collection and produces validated reports.
type IndexReportStream interface {
	// Start begins accepting connections and returns a channel of validated reports.
	// The channel is currently not closed to avoid races during shutdown.
	// TODO: Implement proper shutdown logic that closes the channel.
	Start(ctx context.Context) (<-chan *v1.VMReport, error)
}

// cachedReportMetadata holds metadata about the last update/ACK timestamps for a VSOCK ID.
type cachedReportMetadata struct {
	updatedAt   time.Time
	lastAckedAt time.Time
	limiter     *rate.Limiter
}

// UnconfirmedMessageHandler is the minimal interface used by the relay to track ACK/NACK state.
// Implemented by compliance UMH components.
type UnconfirmedMessageHandler interface {
	HandleACK(resourceID string)
	HandleNACK(resourceID string)
	ObserveSending(resourceID string)
	RetryCommand() <-chan string
	OnACK(callback func(resourceID string))
}

// Relay receives index reports from VMs and forwards them to Sensor.
type Relay struct {
	reportStream IndexReportStream
	reportSender sender.IndexReportSender
	umh          UnconfirmedMessageHandler

	// Rate limiting config
	maxReportsPerMinute float64
	staleAckThreshold   time.Duration

	// cacheEvictTicker owns the metadata eviction ticker in production so it can be
	// stopped when the relay shuts down, avoiding ticker goroutine leaks.
	// Tests may leave this nil and inject cacheEvictTickCh directly.
	cacheEvictTicker *time.Ticker
	// cacheEvictTickCh signals when stale metadata cache entries should be swept.
	// In production this is fed by cacheEvictTicker.C; tests may supply
	// their own channel for deterministic control.
	cacheEvictTickCh <-chan time.Time

	// cache stores metadata (including the per-VSOCK rate limiter) for each VSOCK ID.
	cache map[string]*cachedReportMetadata
	// mu guards cache. A plain Mutex is preferred over RWMutex
	// because nearly every incoming report writes to the map, so there is no
	// read-heavy workload that would benefit from concurrent readers.
	mu sync.Mutex
}

// evictStaleEntries removes metadata entries for VMs that have
// not sent a report within the given threshold. Without periodic eviction the
// maps grow unboundedly as new VM VSOCK IDs appear over time.
func (r *Relay) evictStaleEntries(now time.Time, threshold time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for vsockID, metadata := range r.cache {
		if now.Sub(metadata.updatedAt) > threshold {
			delete(r.cache, vsockID)
		}
	}
}

// New creates a Relay with the given report stream, sender, and unconfirmed message handler.
func New(
	reportStream IndexReportStream,
	reportSender sender.IndexReportSender,
	umh UnconfirmedMessageHandler,
	maxReportsPerMinute float64,
	staleAckThreshold time.Duration,
) *Relay {
	r := &Relay{
		reportStream:        reportStream,
		reportSender:        reportSender,
		umh:                 umh,
		maxReportsPerMinute: maxReportsPerMinute,
		staleAckThreshold:   staleAckThreshold,
		cache:               make(map[string]*cachedReportMetadata),
	}
	if staleAckThreshold <= 0 {
		log.Warnf("VM relay stale ACK threshold is non-positive (%s); disabling stale cache eviction", staleAckThreshold)
	} else {
		cacheEvictInterval := staleAckThreshold / 2
		if cacheEvictInterval <= 0 {
			cacheEvictInterval = staleAckThreshold
		}
		ticker := time.NewTicker(cacheEvictInterval)
		r.cacheEvictTicker = ticker
		r.cacheEvictTickCh = ticker.C
	}
	r.umh.OnACK(r.markAcked)
	return r
}

// Run starts the relay, processing incoming reports and retry commands.
func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")
	if r.cacheEvictTicker != nil {
		defer r.cacheEvictTicker.Stop()
	}

	reportChan, err := r.reportStream.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "starting report stream")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case vmReport := <-reportChan:
			if vmReport == nil {
				log.Warn("Received nil VM report, skipping")
				continue
			}
			r.handleIncomingReport(ctx, vmReport)
		case resourceID, ok := <-r.umh.RetryCommand():
			if !ok {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return errors.New("UMH retry command channel closed")
			}
			log.Infof("UMH retry for resource %s; no payload cache available, skipping resend", resourceID)
		case tick := <-r.cacheEvictTickCh:
			r.evictStaleEntries(tick, r.staleAckThreshold)
		}
	}
}

// markAcked updates lastAckedAt for the given VSOCK ID if present in cache.
func (r *Relay) markAcked(resourceID string) {
	metrics.AcksReceived.Inc()
	now := time.Now()

	concurrency.WithLock(&r.mu, func() {
		if cached, ok := r.cache[resourceID]; ok {
			cached.lastAckedAt = now
		} else {
			r.cache[resourceID] = &cachedReportMetadata{
				updatedAt:   now,
				lastAckedAt: now,
			}
		}
	})
}

// handleIncomingReport processes an incoming report with rate limiting.
func (r *Relay) handleIncomingReport(ctx context.Context, vmReport *v1.VMReport) {
	indexReport := vmReport.GetIndexReport()
	if indexReport == nil || indexReport.GetVsockCid() == "" {
		log.Warn("Received VM report without a valid vsock CID; dropping")
		return
	}
	vsockID := indexReport.GetVsockCid()
	now := time.Now()

	r.cacheReport(vsockID, now)

	if !r.tryConsume(vsockID) {
		if r.staleAckThreshold > 0 && r.isACKStale(vsockID) {
			metrics.ReportsRateLimited.WithLabelValues("stale_ack").Inc()
			log.Warnf("Rate limited for VSOCK %s and last ACK is stale or missing (threshold=%s); dropping report", vsockID, r.staleAckThreshold)
		} else {
			metrics.ReportsRateLimited.WithLabelValues("normal").Inc()
			log.Debugf("Rate limited for VSOCK %s; dropping report and relying on agent retry", vsockID)
		}
		return
	}

	if err := r.reportSender.Send(ctx, vmReport); err != nil {
		r.umh.HandleNACK(vsockID)
		log.Errorf("Failed to send report (vsock CID: %s): %v", vsockID, err)
		return
	}
	r.umh.ObserveSending(vsockID)
}

// cacheReport stores metadata in the cache, keyed by VSOCK ID.
func (r *Relay) cacheReport(vsockID string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.cache[vsockID]; ok {
		existing.updatedAt = now
		return
	}
	r.cache[vsockID] = &cachedReportMetadata{
		updatedAt: now,
	}
}

// isACKStale reports whether the last ACK for vsockID is missing or older than staleAckThreshold.
func (r *Relay) isACKStale(vsockID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	cached, ok := r.cache[vsockID]
	if !ok || cached.lastAckedAt.IsZero() {
		return true
	}
	return time.Since(cached.lastAckedAt) > r.staleAckThreshold
}

// tryConsume checks if we can send a report for this VSOCK ID (leaky bucket).
// The caller must ensure the cache entry exists (e.g. via cacheReport).
func (r *Relay) tryConsume(vsockID string) bool {
	if r.maxReportsPerMinute <= 0 {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := r.cache[vsockID]
	if metadata.limiter == nil {
		ratePerSecond := r.maxReportsPerMinute / 60.0
		metadata.limiter = rate.NewLimiter(rate.Limit(ratePerSecond), 1)
	}
	return metadata.limiter.Allow()
}
