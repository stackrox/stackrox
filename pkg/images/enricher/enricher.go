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
	// NoExternalMetadata runs the enforcement through a "fast-path", skipping any calls to external metadata services.
	// This includes image registries and scanners.
	NoExternalMetadata bool
	// EnforcementOnly indicates that we don't care about any violations unless they have enforcement enabled.
	EnforcementOnly bool

	// IgnoreExisting ensures that, if an image has existing metadata or scans, we don't attempt to re-fetch the metadata.
	IgnoreExisting bool

	// UseNonBlockingCallsWherePossible tells the enricher to make non-blocking calls to image scanners where that is
	// possible. Note that, if NoExternalMetadata is true, this param is irrelevant since no external calls are made at all.
	UseNonBlockingCallsWherePossible bool
}

// EnrichmentResult denotes possible return values of the EnrichImage function.
type EnrichmentResult struct {
	// ImageUpdated returns whether or not the image was updated, either with metadata or with a scan.
	ImageUpdated bool
	ImageError   error

	ScanResult ScanResult
	ScanError  error
}

// A ScanResult denotes the result of an attempt to scan an image.
//go:generate stringer -type=ScanResult
type ScanResult int

const (
	// ScanNotDone denotes that the image was not scanned.
	ScanNotDone ScanResult = iota
	// ScanTriggered denotes that the image was not scanned, but that non-blocking API requests were made
	// to request scanning.
	ScanTriggered
	// ScanSucceeded denotes that the image was successfully scanned.
	ScanSucceeded
)

// ImageEnricher provides functions for enriching images with integrations.
type ImageEnricher interface {
	EnrichImage(ctx EnrichmentContext, image *storage.Image) (EnrichmentResult, error)
}

// New returns a new ImageEnricher instance for the given subsystem.
// (The subsystem is just used for Prometheus metrics.)
func New(is integration.Set, subsystem pkgMetrics.Subsystem, metadataCache, scanCache expiringcache.Cache) ImageEnricher {
	return &enricherImpl{
		integrations: is,

		metadataLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:   metadataCache,

		scanLimiter: rate.NewLimiter(rate.Every(1*time.Second), 5),
		scanCache:   scanCache,

		metrics: newMetrics(subsystem),
	}
}
