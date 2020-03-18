package enricher

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

	asyncRateLimiter *rate.Limiter
	scanCache        expiringcache.Cache

	metrics metrics
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(ctx EnrichmentContext, image *storage.Image) (EnrichmentResult, error) {
	errorList := errorhelpers.NewErrorList("image enrichment")

	updatedMetadata, err := e.enrichWithMetadata(ctx, image)
	errorList.AddError(err)

	scanResult, err := e.enrichWithScan(ctx, image)
	errorList.AddError(err)
	return EnrichmentResult{
		ImageUpdated: updatedMetadata || (scanResult != ScanNotDone),
		ScanResult:   scanResult,
	}, errorList.ToError()
}

func (e *enricherImpl) enrichWithMetadata(ctx EnrichmentContext, image *storage.Image) (bool, error) {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error getting metadata for image: %s", image.GetName().GetFullName()))
	for _, registry := range e.integrations.RegistrySet().GetAll() {
		updated, err := e.enrichImageWithRegistry(ctx, image, registry)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if updated {
			return true, nil
		}
	}
	return false, errorList.ToError()
}

func getRef(image *storage.Image) string {
	if image.GetId() != "" {
		return image.GetId()
	}
	return image.GetName().GetFullName()
}

func (e *enricherImpl) enrichImageWithRegistry(ctx EnrichmentContext, image *storage.Image, registry registryTypes.ImageRegistry) (bool, error) {
	if !registry.Global() {
		return false, nil
	}
	if !registry.Match(image.GetName()) {
		return false, nil
	}

	if ctx.FetchOnlyIfMetadataEmpty() && image.GetMetadata() != nil {
		return false, nil
	}

	ref := getRef(image)
	if ctx.FetchOpt != ForceRefetch {
		if metadataValue := e.metadataCache.Get(ref); metadataValue != nil {
			e.metrics.IncrementMetadataCacheHit()
			image.Metadata = metadataValue.(*storage.ImageMetadata)
			return true, nil
		}
		e.metrics.IncrementMetadataCacheMiss()
	}

	if ctx.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	// Wait until limiter allows entrance
	_ = e.metadataLimiter.Wait(context.Background())
	metadata, err := registry.Metadata(image)
	if err != nil {
		return false, errors.Wrapf(err, "error getting metadata from registry: %q", registry.Name())
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
	return true, nil
}

func (e *enricherImpl) enrichWithScan(ctx EnrichmentContext, image *storage.Image) (ScanResult, error) {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning image: %s", image.GetName().GetFullName()))
	for _, scanner := range e.integrations.ScannerSet().GetAll() {
		result, err := e.enrichImageWithScanner(ctx, image, scanner)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if result != ScanNotDone {
			return result, nil
		}
	}
	return ScanNotDone, errorList.ToError()
}

func (e *enricherImpl) populateFromCache(ctx EnrichmentContext, image *storage.Image) bool {
	if ctx.FetchOpt == ForceRefetch || ctx.FetchOpt == ForceRefetchScansOnly {
		return false
	}
	ref := getRef(image)
	scanValue := e.scanCache.Get(ref)
	if scanValue == nil {
		e.metrics.IncrementScanCacheMiss()
		return false
	}

	e.metrics.IncrementScanCacheHit()
	image.Scan = scanValue.(*storage.ImageScan)
	FillScanStats(image)
	return true
}

func (e *enricherImpl) enrichImageWithScanner(ctx EnrichmentContext, image *storage.Image, scanner scannerTypes.ImageScanner) (ScanResult, error) {
	if !scanner.Match(image.GetName()) {
		return ScanNotDone, nil
	}

	if ctx.FetchOnlyIfMetadataEmpty() && image.GetScan() != nil {
		return ScanNotDone, nil
	}

	if e.populateFromCache(ctx, image) {
		return ScanSucceeded, nil
	}

	if ctx.FetchOpt == NoExternalMetadata {
		return ScanNotDone, nil
	}

	var scan *storage.ImageScan

	if asyncScanner, ok := scanner.(scannerTypes.AsyncImageScanner); ok && ctx.UseNonBlockingCallsWherePossible {
		_ = e.asyncRateLimiter.Wait(context.Background())

		if e.populateFromCache(ctx, image) {
			return ScanSucceeded, nil
		}

		var err error
		scan, err = asyncScanner.GetOrTriggerScan(image)
		if err != nil {
			return ScanNotDone, errors.Wrapf(err, "Error triggering scan for %q with scanner %q", image.GetName().GetFullName(), scanner.Name())
		}
		if scan == nil {
			return ScanTriggered, nil
		}
	} else {
		if e.populateFromCache(ctx, image) {
			return ScanSucceeded, nil
		}

		sema := scanner.MaxConcurrentScanSemaphore()
		_ = sema.Acquire(context.Background(), 1)
		defer sema.Release(1)

		var err error
		scanStartTime := time.Now()
		scan, err = scanner.GetScan(image)
		e.metrics.SetScanDurationTime(scanStartTime, scanner.Name(), err)

		if err != nil {
			return ScanNotDone, errors.Wrapf(err, "Error scanning %q with scanner %q", image.GetName().GetFullName(), scanner.Name())
		}
		if scan == nil {
			return ScanNotDone, nil
		}

	}

	// Assume:
	//  scan != nil
	//  no error scanning.
	image.Scan = scan
	FillScanStats(image)

	e.scanCache.Add(getRef(image), scan)
	if image.GetId() == "" {
		if digest := image.GetMetadata().GetV2().GetDigest(); digest != "" {
			e.scanCache.Add(digest, scan)
		}
		if digest := image.GetMetadata().GetV1().GetDigest(); digest != "" {
			e.scanCache.Add(digest, scan)
		}
	}
	return ScanSucceeded, nil
}

// FillScanStats fills in the higher level stats from the scan data.
func FillScanStats(i *storage.Image) {
	if i.GetScan() != nil {
		i.SetComponents = &storage.Image_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}

		var fixedByProvided bool
		var imageTopCVSS float32
		vulns := make(map[string]bool)
		for _, c := range i.GetScan().GetComponents() {
			var componentTopCVSS float32
			var hasVulns bool
			for _, v := range c.GetVulns() {
				hasVulns = true
				if _, ok := vulns[v.GetCve()]; !ok {
					vulns[v.GetCve()] = false
				}

				if v.GetCvss() > componentTopCVSS {
					componentTopCVSS = v.GetCvss()
				}

				if v.GetSetFixedBy() == nil {
					continue
				}

				fixedByProvided = true
				if v.GetFixedBy() != "" {
					vulns[v.GetCve()] = true
				}
			}

			if hasVulns {
				c.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: componentTopCVSS,
				}
			}

			if componentTopCVSS > imageTopCVSS {
				imageTopCVSS = componentTopCVSS
			}
		}

		i.SetCves = &storage.Image_Cves{
			Cves: int32(len(vulns)),
		}

		if len(vulns) > 0 {
			i.SetTopCvss = &storage.Image_TopCvss{
				TopCvss: imageTopCVSS,
			}
		}

		if int32(len(vulns)) == 0 || fixedByProvided {
			var numFixableVulns int32
			for _, fixable := range vulns {
				if fixable {
					numFixableVulns++
				}
			}
			i.SetFixable = &storage.Image_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
}
