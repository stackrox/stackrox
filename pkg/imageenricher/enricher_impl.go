package imageenricher

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
	"github.com/karlseguin/ccache"
	"golang.org/x/time/rate"
)

const (
	imageDataExpiration = 10 * time.Minute

	maxCacheSize = 500
	itemsToPrune = 100
)

type enricherImpl struct {
	integrations IntegrationSet

	metadataLimiter *rate.Limiter
	metadataCache   *ccache.Cache

	scanLimiter *rate.Limiter
	scanCache   *ccache.Cache
}

// IntegrationSet returns the object holding the set of integrations to use for enrichment.
func (e *enricherImpl) IntegrationSet() IntegrationSet {
	return e.integrations
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(image *v1.Image) bool {
	updatedMetadata := e.enrichWithMetadata(image)
	updatedScan := e.enrichWithScan(image)
	return updatedMetadata || updatedScan
}

func (e *enricherImpl) enrichWithMetadata(image *v1.Image) bool {
	for _, integration := range e.integrations.GetAll() {
		if integration.Registry == nil {
			continue
		}
		if updated := e.enrichImageWithRegistry(image, integration.Registry); updated {
			return true
		}
	}
	return false
}

func (e *enricherImpl) enrichImageWithRegistry(image *v1.Image, registry registries.ImageRegistry) bool {
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

func (e *enricherImpl) enrichWithScan(image *v1.Image) bool {
	for _, integration := range e.integrations.GetAll() {
		if integration.Scanner == nil {
			continue
		}
		if updated := e.enrichImageWithScanner(image, integration.Scanner); updated {
			return true
		}
	}
	return false
}

func (e *enricherImpl) equalComponents(components1, components2 []*v1.ImageScanComponent) bool {
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

func (e *enricherImpl) enrichImageWithScanner(image *v1.Image, scanner scannerTypes.ImageScanner) bool {
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
