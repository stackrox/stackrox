package enricher

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// FetchOption determines what attempts should be made to retrieve the metadata
type FetchOption int

// These are all the possible fetch options for the enricher
const (
	UseCachesIfPossible FetchOption = iota
	NoExternalMetadata
	IgnoreExistingImages
	ForceRefetch
)

const (
	maxConcurrentScans = 6
)

var (
	log = logging.LoggerForModule()
)

// EnrichmentContext is used to pass options through the enricher without exploding the number of function arguments
type EnrichmentContext struct {
	// FetchOpt define constraints about using external data
	FetchOpt FetchOption

	// EnforcementOnly indicates that we don't care about any violations unless they have enforcement enabled.
	EnforcementOnly bool

	// UseNonBlockingCallsWherePossible tells the enricher to make non-blocking calls to image scanners where that is
	// possible. Note that, if NoExternalMetadata is true, this param is irrelevant since no external calls are made at all.
	UseNonBlockingCallsWherePossible bool
}

// FetchOnlyIfMetadataEmpty checks the fetch opts and return whether or not we can used a cached or saved
// version of the external metadata
func (e EnrichmentContext) FetchOnlyIfMetadataEmpty() bool {
	return e.FetchOpt != IgnoreExistingImages && e.FetchOpt != ForceRefetch
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

		syncSemaphore:    semaphore.NewWeighted(maxConcurrentScans),
		asyncRateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 5),
		scanCache:        scanCache,

		metrics: newMetrics(subsystem),
	}
}
