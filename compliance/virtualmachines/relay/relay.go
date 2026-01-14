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

	// cache stores metadata for each VSOCK ID.
	cache map[string]*cachedReportMetadata
	// limiters stores per-VSOCK rate limiters (leaky bucket, no burst).
	limiters map[string]*rate.Limiter
	// mu guards cache and limiters.
	mu sync.Mutex
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
		limiters:            make(map[string]*rate.Limiter),
	}
	// Register callback for ACKs
	r.umh.OnACK(r.markAcked)
	return r
}

// Run starts the relay, processing incoming reports and retry commands.
func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")

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
			r.handleIncomingReport(ctx, vmReport.GetIndexReport())
		case vsockID, ok := <-r.umh.RetryCommand():
			if !ok {
				log.Warn("UMH retry channel closed; stopping relay")
				return errors.New("UMH retry channel closed unexpectedly")
			}
			// Relay no longer stores reports; rely on agent resubmission.
			log.Debugf("Retry requested for VSOCK %s, ignoring (no cached report retained)", vsockID)

	}
}

// markAcked updates lastAckedAt for the given VSOCK ID if present in cache.
func (r *Relay) markAcked(vsockID string) {
	metrics.AcksReceived.Inc()
	now := time.Now()
	concurrency.WithLock(&r.mu, func() {
		if cached, ok := r.cache[vsockID]; ok {
			cached.lastAckedAt = now
		} else {
			r.cache[vsockID] = &cachedReportMetadata{
				updatedAt:   now,
				lastAckedAt: now,
			}
		}
	})
}

// handleIncomingReport processes an incoming report with rate limiting.
func (r *Relay) handleIncomingReport(ctx context.Context, report *v1.IndexReport) {
	vsockID := report.GetVsockCid()

	// Always cache metadata for the latest report
	r.cacheReport(report)

	// Check rate limit
	if !r.tryConsume(vsockID) {
		// Rate limited: drop and rely on agent retrying later, but track ACK recency to aid diagnostics.
		metadata := r.getCachedReport(vsockID)
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
	r.umh.ObserveSending(vsockID)
	if err := r.reportSender.Send(ctx, report); err != nil {
		log.Errorf("Failed to send report (vsock CID: %s): %v", vsockID, err)
	}
}

// cacheReport stores metadata in the cache, keyed by VSOCK ID.
func (r *Relay) cacheReport(report *v1.IndexReport) {
	vsockID := report.GetVsockCid()

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.cache[vsockID]; ok {
		r.cache[vsockID] = &cachedReportMetadata{
			updatedAt:   time.Now(),
			lastAckedAt: existing.lastAckedAt,
		}
	} else {
		r.cache[vsockID] = &cachedReportMetadata{
			updatedAt: time.Now(),
		}
	}
}

func (r *Relay) getCachedReport(vsockID string) *cachedReportMetadata {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.cache[vsockID]; ok {
		copy := *cached
		return &copy
	}
	return nil
}

// tryConsume checks if we can send a report for this VSOCK ID (leaky bucket).
func (r *Relay) tryConsume(vsockID string) bool {
	if r.maxReportsPerMinute <= 0 {
		// Rate limiting disabled
		return true
	}

	var limiter *rate.Limiter
	concurrency.WithLock(&r.mu, func() {
		var exists bool
		limiter, exists = r.limiters[vsockID]
		if !exists {
			// Create leaky bucket: rate = maxReportsPerMinute/60, burst = 1 (no bursts)
			ratePerSecond := r.maxReportsPerMinute / 60.0
			limiter = rate.NewLimiter(rate.Limit(ratePerSecond), 1)
			r.limiters[vsockID] = limiter
		}
	})

	return limiter.Allow()
}
