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
	// payloadCacheTTL is the configured TTL for the payload cache; when positive,
	// New installs a periodic payload sweep ticker independent of staleAckThreshold.
	payloadCacheTTL time.Duration

	// cacheEvictTicker owns the metadata eviction ticker in production so it can be
	// stopped when the relay shuts down, avoiding ticker goroutine leaks.
	// Tests may leave this nil and inject cacheEvictTickCh directly.
	cacheEvictTicker *time.Ticker
	// cacheEvictTickCh signals when stale metadata cache entries should be swept.
	// In production this is fed by cacheEvictTicker.C; tests may supply
	// their own channel for deterministic control.
	cacheEvictTickCh <-chan time.Time

	// payloadSweepTicker drives periodic TTL sweeps of the payload cache when
	// payloadCacheTTL > 0. It is independent of staleAckThreshold and cacheEvictTicker.
	payloadSweepTicker *time.Ticker
	// payloadSweepTickCh signals when expired payload cache entries should be swept.
	// In production this is fed by payloadSweepTicker.C; tests may supply their own channel.
	payloadSweepTickCh <-chan time.Time

	// cache stores metadata (including the per-VSOCK rate limiter) for each VSOCK ID.
	cache map[string]*cachedReportMetadata
	// mu guards cache. A plain Mutex is preferred over RWMutex
	// because nearly every incoming report writes to the map, so there is no
	// read-heavy workload that would benefit from concurrent readers.
	mu sync.Mutex

	// payloadCache stores full VM reports for UMH-driven retransmission; it uses its own mutex
	// (see reportPayloadCache) and is keyed by the same resource ID as UMH (vsock CID).
	payloadCache *reportPayloadCache
}

// evictStaleEntries removes metadata entries for VMs that have
// not sent a report within the given threshold. Without periodic eviction the
// maps grow unboundedly as new VM VSOCK IDs appear over time.
func (r *Relay) evictStaleEntries(now time.Time, threshold time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove cache entries for VMs that have not sent a report within the given threshold.
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
	payloadCacheMaxSlots int,
	payloadCacheTTL time.Duration,
) *Relay {
	r := &Relay{
		reportStream:        reportStream,
		reportSender:        reportSender,
		umh:                 umh,
		maxReportsPerMinute: maxReportsPerMinute,
		staleAckThreshold:   staleAckThreshold,
		payloadCacheTTL:     payloadCacheTTL,
		cache:               make(map[string]*cachedReportMetadata),
		payloadCache:        newReportPayloadCache(payloadCacheMaxSlots, payloadCacheTTL),
	}
	metrics.IndexReportCacheSlotsCapacity.Set(float64(payloadCacheMaxSlots))
	metrics.IndexReportCacheSlotsUsed.Set(float64(r.payloadCache.Len()))
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
	if payloadCacheTTL > 0 {
		payloadSweepInterval := payloadCacheTTL / 2
		if payloadSweepInterval <= 0 {
			payloadSweepInterval = payloadCacheTTL
		}
		pt := time.NewTicker(payloadSweepInterval)
		r.payloadSweepTicker = pt
		r.payloadSweepTickCh = pt.C
	}
	// Register callback for ACKs
	r.umh.OnACK(r.markAcked)
	return r
}

// Run starts the relay, processing incoming reports and retry commands.
func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")
	if r.cacheEvictTicker != nil {
		defer r.cacheEvictTicker.Stop()
	}
	if r.payloadSweepTicker != nil {
		defer r.payloadSweepTicker.Stop()
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
				return errors.New("UMH retry command channel closed")
			}
			r.handleRetryCommand(ctx, resourceID)
		case tick := <-r.cacheEvictTickCh:
			r.evictStaleEntries(tick, r.staleAckThreshold)
		case tick := <-r.payloadSweepTickCh:
			r.sweepPayloadCache(tick)
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

	ev, removed := r.payloadCache.Remove(resourceID, now)
	if removed {
		observePayloadEvictionMetrics(ev)
	}
	metrics.IndexReportCacheSlotsUsed.Set(float64(r.payloadCache.Len()))
}

// handleIncomingReport processes an incoming report with rate limiting.
func (r *Relay) handleIncomingReport(ctx context.Context, vmReport *v1.VMReport) {
	indexReport := vmReport.GetIndexReport()
	vsockID := indexReport.GetVsockCid()

	now := time.Now()
	for _, ev := range r.payloadCache.Upsert(vsockID, vmReport, now) {
		observePayloadEvictionMetrics(ev)
	}
	metrics.IndexReportCacheSlotsUsed.Set(float64(r.payloadCache.Len()))

	// Always cache metadata for the latest report.
	r.cacheReport(indexReport)

	if !r.tryConsume(vsockID) {
		// Rate limited: drop and rely on agent retrying later, but track ACK recency to aid diagnostics.
		metadata := r.getCachedReportMetadata(vsockID)
		if r.staleAckThreshold > 0 && (metadata == nil || metadata.lastAckedAt.IsZero() || time.Since(metadata.lastAckedAt) > r.staleAckThreshold) {
			metrics.ReportsRateLimited.WithLabelValues("stale_ack").Inc()
			log.Warnf("Rate limited for VSOCK %s and last ACK is stale or missing (threshold=%s); dropping report", vsockID, r.staleAckThreshold)
		} else {
			metrics.ReportsRateLimited.WithLabelValues("normal").Inc()
			if metadata != nil && !metadata.lastAckedAt.IsZero() {
				log.Debugf("Rate limited for VSOCK %s; last ACK %s ago, dropping and relying on agent retry", vsockID, time.Since(metadata.lastAckedAt))
			} else {
				log.Debugf("Rate limited for VSOCK %s; dropping report and relying on agent retry", vsockID)
			}
		}
		return
	}

	// Send the report (notify UMH and forward)
	if err := r.reportSender.Send(ctx, vmReport); err != nil {
		// Send NACK to yourself when sending fails.
		r.umh.HandleNACK(vsockID)
		log.Errorf("Failed to send report (vsock CID: %s): %v", vsockID, err)
		return
	}
	r.umh.ObserveSending(vsockID)
}

// cacheReport stores metadata in the cache, keyed by VSOCK ID.
func (r *Relay) cacheReport(report *v1.IndexReport) {
	vsockID := report.GetVsockCid()
	now := time.Now()

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

// getCachedReportMetadata returns a copy of cached report metadata for the given VSOCK ID.
func (r *Relay) getCachedReportMetadata(vsockID string) *cachedReportMetadata {
	r.mu.Lock()
	defer r.mu.Unlock()

	cached, ok := r.cache[vsockID]
	if !ok {
		return nil
	}

	return &cachedReportMetadata{
		updatedAt:   cached.updatedAt,
		lastAckedAt: cached.lastAckedAt,
	}
}

// tryConsume checks if we can send a report for this VSOCK ID (leaky bucket).
func (r *Relay) tryConsume(vsockID string) bool {
	if r.maxReportsPerMinute <= 0 {
		// Rate limiting disabled
		return true
	}

	now := time.Now()
	limiter := func() *rate.Limiter {
		r.mu.Lock()
		defer r.mu.Unlock()

		metadata, exists := r.cache[vsockID]
		if !exists {
			metadata = &cachedReportMetadata{
				updatedAt: now,
			}
			r.cache[vsockID] = metadata
		}
		if metadata.limiter == nil {
			// Create leaky bucket: rate = maxReportsPerMinute/60, burst = 1 (no bursts)
			ratePerSecond := r.maxReportsPerMinute / 60.0
			metadata.limiter = rate.NewLimiter(rate.Limit(ratePerSecond), 1)
		}
		return metadata.limiter
	}()

	return limiter.Allow()
}

// sweepPayloadCache removes expired payload cache entries and updates cache metrics.
func (r *Relay) sweepPayloadCache(now time.Time) {
	for _, ev := range r.payloadCache.SweepExpired(now) {
		observePayloadEvictionMetrics(ev)
	}
	metrics.IndexReportCacheSlotsUsed.Set(float64(r.payloadCache.Len()))
}

// handleRetryCommand attempts to resend a cached payload for the given resource ID.
func (r *Relay) handleRetryCommand(ctx context.Context, resourceID string) {
	now := time.Now()
	cached, ok := r.payloadCache.Get(resourceID, now)
	if ok {
		metrics.IndexReportCacheLookupsTotal.WithLabelValues("hit").Inc()
		if err := r.reportSender.Send(ctx, cached); err != nil {
			r.umh.HandleNACK(resourceID)
			log.Errorf("Failed to resend cached VM index report (resourceID=%s): %v", resourceID, err)
			return
		}
		r.umh.ObserveSending(resourceID)
		return
	}
	metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss").Inc()
	log.Infof("VM index report payload cache miss on retry for resource %s; not resending", resourceID)
}

func observePayloadEvictionMetrics(ev payloadEviction) {
	metrics.IndexReportCacheResidencySeconds.Observe(ev.residency.Seconds())
	metrics.IndexReportCacheLifetimeSeconds.Observe(ev.lifetime.Seconds())
}
