package enricher

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"golang.org/x/time/rate"
)

var (
	log = logging.LoggerForModule()
)

// EnrichmentContext is used to pass options through the enricher without exploding the number of function arguments
type EnrichmentContext struct {
	NoExternalMetadata bool
	EnforcementOnly    bool
}

// ImageEnricher provides functions for enriching images with integrations.
type ImageEnricher interface {
	EnrichImage(ctx EnrichmentContext, image *storage.Image) bool
}

// New returns a new ImageEnricher instance for the given subsystem.
// (The subsystem is just used for Prometheus metrics.)
func New(is integration.Set, subsystem pkgMetrics.Subsystem, metadataCache, scanCache expiringcache.Cache) ImageEnricher {
	return &enricherImpl{
		integrations: is,

		metadataLimiter: rate.NewLimiter(rate.Every(1*time.Second), 3),
		metadataCache:   metadataCache,

		scanLimiter: rate.NewLimiter(rate.Every(3*time.Second), 3),
		scanCache:   scanCache,

		metrics: newMetrics(subsystem),
	}
}
