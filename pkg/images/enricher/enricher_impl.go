package enricher

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/integration"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"golang.org/x/time/rate"
)

type enricherImpl struct {
	integrations integration.Set

	metadataLimiter *rate.Limiter
	metadataCache   expiringcache.Cache

	scanLimiter *rate.Limiter
	scanCache   expiringcache.Cache

	metrics metrics
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(image *storage.Image) bool {
	updatedMetadata := e.enrichWithMetadata(image)
	updatedScan := e.enrichWithScan(image)
	return updatedMetadata || updatedScan
}

func (e *enricherImpl) enrichWithMetadata(image *storage.Image) bool {
	for _, registry := range e.integrations.RegistrySet().GetAll() {
		if updated := e.enrichImageWithRegistry(image, registry); updated {
			return true
		}
	}
	return false
}

func (e *enricherImpl) enrichImageWithRegistry(image *storage.Image, registry registryTypes.ImageRegistry) bool {
	if !registry.Global() {
		return false
	}
	if !registry.Match(image) {
		return false
	}

	if metadataValue := e.metadataCache.Get(image.GetId()); metadataValue != nil {
		image.Metadata = metadataValue.(*storage.ImageMetadata)
		return true
	}

	// Wait until limiter allows entrance
	_ = e.metadataLimiter.Wait(context.Background())
	metadata, err := registry.Metadata(image)
	if err != nil {
		logger.Error(err)
		return false
	}
	image.Metadata = metadata
	e.metadataCache.Add(image.GetId(), metadata)
	return true
}

func (e *enricherImpl) enrichWithScan(image *storage.Image) bool {
	for _, scanner := range e.integrations.ScannerSet().GetAll() {
		if updated := e.enrichImageWithScanner(image, scanner); updated {
			return true
		}
	}
	return false
}

func (e *enricherImpl) enrichImageWithScanner(image *storage.Image, scanner scannerTypes.ImageScanner) bool {
	if !scanner.Global() {
		return false
	}
	if !scanner.Match(image) {
		return false
	}
	if scanValue := e.scanCache.Get(image.GetId()); scanValue != nil {
		image.Scan = scanValue.(*storage.ImageScan)
		return true
	}
	// Wait until limiter allows entrance
	_ = e.scanLimiter.Wait(context.Background())
	scan, err := scanner.GetLastScan(image)
	if err != nil {
		logger.Errorf("Error getting last scan for %s: %s", image.GetName().GetFullName(), err)
		return false
	}
	image.Scan = scan
	e.scanCache.Add(image.GetId(), scan)
	return true
}
