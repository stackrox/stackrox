package enricher

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
	"github.com/karlseguin/ccache"
	"golang.org/x/time/rate"
)

const (
	imageDataExpiration = 10 * time.Minute

	maxCacheSize = 500
	itemsToPrune = 100
)

var (
	logger = logging.LoggerForModule()

	// ImageEnricher is the global enricher for images
	ImageEnricher *enricher
)

type enricher struct {
	integrations map[string]*sources.ImageIntegration
	lock         sync.RWMutex

	metadataLimiter *rate.Limiter
	metadataCache   *ccache.Cache

	scanLimiter *rate.Limiter
	scanCache   *ccache.Cache
}

func init() {
	ImageEnricher = newEnricher()
}

func newEnricher() *enricher {
	return &enricher{
		integrations: make(map[string]*sources.ImageIntegration),

		metadataLimiter: rate.NewLimiter(rate.Every(5*time.Second), 3),
		metadataCache:   ccache.New(ccache.Configure().MaxSize(maxCacheSize).ItemsToPrune(itemsToPrune)),
		scanLimiter:     rate.NewLimiter(rate.Every(5*time.Second), 3),
		scanCache:       ccache.New(ccache.Configure().MaxSize(maxCacheSize).ItemsToPrune(itemsToPrune)),
	}
}

// UpdateImageIntegration updates the enricher's map of active image integratinos
func (e *enricher) UpdateImageIntegration(integration *sources.ImageIntegration) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.integrations[integration.GetId()] = integration
}

// RemoveImageIntegration removes a image integration from the enricher's map of active image integrations
func (e *enricher) RemoveImageIntegration(id string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.integrations, id)
}

func (e *enricher) GetRegistryMetadataByImage(image *v1.Image) *registries.Config {
	e.lock.RLock()
	defer e.lock.RUnlock()
	for _, i := range e.integrations {
		if i.Registry != nil && i.Registry.Match(image) {
			return i.Registry.Config()
		}
	}
	return nil
}

// Match determines if an image integration matches
func (e *enricher) Match(image *v1.Image) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	for _, i := range e.integrations {
		if i.Registry != nil && i.Registry.Match(image) {
			return true
		}
	}
	return false
}

// EnrichWithImageIntegration takes in a deployment and integration
func (e *enricher) EnrichWithImageIntegration(image *v1.Image, integration *sources.ImageIntegration) (wasUpdated bool) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	// TODO(cgorman) These may have a real ordering that we need to adhere to
	for _, category := range integration.GetCategories() {
		switch category {
		case v1.ImageIntegrationCategory_REGISTRY:
			if updated := e.enrichImageWithRegistry(image, integration.Registry); updated {
				wasUpdated = updated
			}
		case v1.ImageIntegrationCategory_SCANNER:
			if updated := e.enrichImageWithScanner(image, integration.Scanner); updated {
				wasUpdated = updated
			}
		}
	}
	return
}

func (e *enricher) EnrichImage(image *v1.Image) bool {
	updatedMetadata := e.enrichWithMetadata(image)
	updatedScan := e.enrichWithScan(image)
	return updatedMetadata || updatedScan
}

func (e *enricher) enrichWithMetadata(image *v1.Image) bool {
	for _, integration := range e.integrations {
		if integration.Registry == nil {
			continue
		}
		if updated := e.enrichImageWithRegistry(image, integration.Registry); updated {
			return true
		}
	}
	return false
}

func (e *enricher) enrichImageWithRegistry(image *v1.Image, registry registries.ImageRegistry) bool {
	if !registry.Global() {
		return false
	}
	if !registry.Match(image) {
		return false
	}
	// Wait until limiter allows entrance
	var metadata *v1.ImageMetadata
	metadataItem := e.metadataCache.Get(image.GetName().GetFullName())
	if metadataItem == nil {
		metrics.IncrementMetadataCacheMiss()
		e.metadataLimiter.Wait(context.Background())

		var err error
		metadata, err = registry.Metadata(image)
		if err != nil {
			logger.Error(err)
			return false
		}
		e.metadataCache.Set(image.GetName().GetFullName(), metadata, imageDataExpiration)
	} else {
		metrics.IncrementMetadataCacheHit()
		metadata = metadataItem.Value().(*v1.ImageMetadata)
	}

	if protoconv.CompareProtoTimestamps(image.GetMetadata().GetCreated(), metadata.GetCreated()) != 0 {
		image.Metadata = metadata
		return true
	}

	return false
}

func (e *enricher) enrichWithScan(image *v1.Image) bool {
	for _, integration := range e.integrations {
		if integration.Scanner == nil {
			continue
		}
		if updated := e.enrichImageWithScanner(image, integration.Scanner); updated {
			return true
		}
	}
	return false
}

func (e *enricher) equalComponents(components1, components2 []*v1.ImageScanComponent) bool {
	if components1 == nil && components2 == nil {
		return true
	} else if components1 == nil || components2 == nil {
		return false
	}
	if len(components1) != len(components2) {
		return false
	}
	for i := 0; i < len(components1); i++ {
		c1 := components1[i]
		c2 := components2[i]
		if len(c1.GetVulns()) != len(c2.GetVulns()) {
			return false
		}
		for j := 0; j < len(c1.GetVulns()); j++ {
			v1 := c1.GetVulns()[j]
			v2 := c2.GetVulns()[j]
			if v1.GetCve() != v2.GetCve() || v1.GetCvss() != v2.GetCvss() || v1.GetLink() != v2.GetLink() || v1.GetSummary() != v2.GetSummary() {
				return false
			}
		}
	}
	return true
}

func (e *enricher) enrichImageWithScanner(image *v1.Image, scanner scannerTypes.ImageScanner) bool {
	if !scanner.Global() {
		return false
	}
	if !scanner.Match(image) {
		return false
	}
	var scan *v1.ImageScan
	scanItem := e.scanCache.Get(image.GetName().GetSha())
	if scanItem == nil {
		metrics.IncrementScanCacheMiss()
		e.scanLimiter.Wait(context.Background())

		var err error
		scan, err = scanner.GetLastScan(image)
		if err != nil {
			logger.Errorf("Error getting last scan for %s: %s", image.GetName().GetFullName(), err)
			return false
		}
		e.scanCache.Set(image.GetName().GetSha(), scan, imageDataExpiration)
	} else {
		metrics.IncrementScanCacheHit()
		scan = scanItem.Value().(*v1.ImageScan)
	}

	if protoconv.CompareProtoTimestamps(image.GetScan().GetScanTime(), scan.GetScanTime()) != 0 || !e.equalComponents(image.GetScan().GetComponents(), scan.GetComponents()) {
		image.Scan = scan
		return true
	}
	return false
}
