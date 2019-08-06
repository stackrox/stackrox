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
func (e *enricherImpl) EnrichImage(ctx EnrichmentContext, image *storage.Image) EnrichmentResult {
	updatedMetadata := e.enrichWithMetadata(ctx, image)
	scanResult := e.enrichWithScan(ctx, image)
	return EnrichmentResult{
		ImageUpdated: updatedMetadata || (scanResult != ScanNotDone),
		ScanResult:   scanResult,
	}
}

func (e *enricherImpl) enrichWithMetadata(ctx EnrichmentContext, image *storage.Image) bool {
	for _, registry := range e.integrations.RegistrySet().GetAll() {
		if updated := e.enrichImageWithRegistry(ctx, image, registry); updated {
			return true
		}
	}
	return false
}

func getRef(image *storage.Image) string {
	if image.GetId() != "" {
		return image.GetId()
	}
	return image.GetName().GetFullName()
}

func (e *enricherImpl) enrichImageWithRegistry(ctx EnrichmentContext, image *storage.Image, registry registryTypes.ImageRegistry) bool {
	if !registry.Global() {
		return false
	}
	if !registry.Match(image) {
		return false
	}

	if !ctx.IgnoreExisting && image.GetMetadata() != nil {
		return false
	}

	ref := getRef(image)
	if metadataValue := e.metadataCache.Get(ref); metadataValue != nil {
		e.metrics.IncrementMetadataCacheHit()
		image.Metadata = metadataValue.(*storage.ImageMetadata)
		return true
	}
	e.metrics.IncrementMetadataCacheMiss()

	if ctx.NoExternalMetadata {
		return false
	}

	// Wait until limiter allows entrance
	_ = e.metadataLimiter.Wait(context.Background())
	metadata, err := registry.Metadata(image)
	if err != nil {
		log.Error(err)
		return false
	}
	image.Metadata = metadata
	e.metadataCache.Add(ref, metadata)
	if image.GetId() == "" {
		if digest := image.Metadata.GetV2().GetDigest(); digest != "" {
			e.metadataCache.Add(digest, metadata)
		}
		if digest := image.Metadata.GetV1().GetDigest(); digest != "" {
			e.metadataCache.Add(digest, metadata)
		}
	}
	return true
}

func (e *enricherImpl) enrichWithScan(ctx EnrichmentContext, image *storage.Image) ScanResult {
	for _, scanner := range e.integrations.ScannerSet().GetAll() {
		result := e.enrichImageWithScanner(ctx, image, scanner)
		if result != ScanNotDone {
			return result
		}
	}
	return ScanNotDone
}

func (e *enricherImpl) enrichImageWithScanner(ctx EnrichmentContext, image *storage.Image, scanner scannerTypes.ImageScanner) ScanResult {
	if !scanner.Match(image) {
		return ScanNotDone
	}

	if !ctx.IgnoreExisting && image.GetScan() != nil {
		return ScanNotDone
	}

	ref := getRef(image)
	if scanValue := e.scanCache.Get(ref); scanValue != nil {
		e.metrics.IncrementScanCacheHit()
		image.Scan = scanValue.(*storage.ImageScan)
		FillScanStats(image)
		return ScanSucceeded
	}
	e.metrics.IncrementScanCacheMiss()

	if ctx.NoExternalMetadata {
		return ScanNotDone
	}

	// Wait until limiter allows entrance
	_ = e.scanLimiter.Wait(context.Background())

	var scan *storage.ImageScan

	if asyncScanner, ok := scanner.(scannerTypes.AsyncImageScanner); ok && ctx.UseNonBlockingCallsWherePossible {
		var err error
		scan, err = asyncScanner.GetOrTriggerScan(image)
		if err != nil {
			log.Errorf("Error triggering scan for %q: %v", image.GetName().GetFullName(), err)
			return ScanNotDone
		}
		if scan == nil {
			return ScanTriggered
		}
	} else {
		var err error
		scan, err = scanner.GetScan(image)
		if err != nil {
			log.Errorf("Error scanning %q: %v", image.GetName().GetFullName(), err)
			return ScanNotDone
		}
		if scan == nil {
			return ScanNotDone
		}
	}

	// Assume:
	//  scan != nil
	//  no error scanning.
	image.Scan = scan
	FillScanStats(image)

	e.scanCache.Add(ref, scan)
	if image.GetId() == "" {
		if digest := image.GetMetadata().GetV2().GetDigest(); digest != "" {
			e.scanCache.Add(digest, scan)
		}
		if digest := image.GetMetadata().GetV1().GetDigest(); digest != "" {
			e.scanCache.Add(digest, scan)
		}
	}
	return ScanSucceeded
}

// FillScanStats fills in the higher level stats from the scan data.
func FillScanStats(i *storage.Image) {
	if i.GetScan() != nil {
		i.SetComponents = &storage.Image_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}
		var numVulns int32
		var numFixableVulns int32
		var fixedByProvided bool
		for _, c := range i.GetScan().GetComponents() {
			numVulns += int32(len(c.GetVulns()))
			for _, v := range c.GetVulns() {
				if v.GetSetFixedBy() != nil {
					fixedByProvided = true
					if v.GetFixedBy() != "" {
						numFixableVulns++
					}
				}
			}
		}
		i.SetCves = &storage.Image_Cves{
			Cves: numVulns,
		}
		if numVulns == 0 || fixedByProvided {
			i.SetFixable = &storage.Image_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
}
