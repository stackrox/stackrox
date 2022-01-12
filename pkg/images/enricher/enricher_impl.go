package enricher

import (
	"context"
	"fmt"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/integrationhealth"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/time/rate"
)

const (
	// The number of consecutive errors for a scanner or registry that cause its health status to be UNHEALTHY
	consecutiveErrorThreshold = 3
)

type enricherImpl struct {
	cvesSuppressor   cveSuppressor
	cvesSuppressorV2 cveSuppressor
	integrations     integration.Set

	errorsPerRegistry  map[registryTypes.ImageRegistry]int32
	registryErrorsLock sync.RWMutex
	errorsPerScanner   map[scannerTypes.ImageScanner]int32
	scannerErrorsLock  sync.RWMutex

	integrationHealthReporter integrationhealth.Reporter ``

	metadataLimiter *rate.Limiter
	metadataCache   expiringcache.Cache

	asyncRateLimiter *rate.Limiter
	scanCache        expiringcache.Cache

	metrics metrics
}

func (e *enricherImpl) EnrichWithVulnerabilities(ctx EnrichmentContext, image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (EnrichmentResult, error) {
	// Attempt to short-circuit before checking scanners.
	if ctx.FetchOnlyIfScanEmpty() && image.GetScan() != nil {
		return EnrichmentResult{
			ScanResult: ScanNotDone,
		}, nil
	}
	if e.populateFromCache(ctx, image) {
		return EnrichmentResult{
			ImageUpdated: true,
			ScanResult:   ScanSucceeded,
		}, nil
	}

	scanners := e.integrations.ScannerSet()
	if scanners.IsEmpty() {
		return EnrichmentResult{
			ScanResult:   ScanNotDone,
		}, errors.New("no image scanners are integrated")
	}

	for _, scanner := range scanners.GetAll() {
		if vulnScanner, ok := scanner.(scannerTypes.ImageVulnerabilityGetter); ok {
			// Clairify is the only supported ImageVulnerabilityGetter at this time.
			if scanner.Type() != clairify.TypeString {
				log.Errorf("unexpected image vulnerability getter: %s [%s]", scanner.Name(), scanner.Type())
				continue
			}

			res, err := e.enrichWithVulnerabilities(scanner.Name(), scanner.DataSource(), vulnScanner, image, components, notes)
			if err != nil {
				return EnrichmentResult{
					ScanResult:   ScanNotDone,
				}, errors.Wrapf(err, "retrieving image vulnerabilities from %s [%s]", scanner.Name(), scanner.Type())
			}

			return EnrichmentResult{
				ImageUpdated: res != ScanNotDone,
				ScanResult:   res,
			}, nil
		}
	}

	return EnrichmentResult{
		ScanResult:   ScanNotDone,
	}, errors.New("no image vulnerability retrievers are integrated")
}

func (e *enricherImpl) enrichWithVulnerabilities(scannerName string, dataSource *storage.DataSource, scanner scannerTypes.ImageVulnerabilityGetter,
	image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (ScanResult, error) {
	scanStartTime := time.Now()
	scan, err := scanner.GetVulnerabilities(image, components, notes)
	e.metrics.SetImageVulnerabilityRetrievalTime(scanStartTime, scannerName, err)
	if err != nil {
		return ScanNotDone, err
	}
	if scan == nil {
		return ScanNotDone, nil
	}

	e.enrichImage(image, scan, dataSource)

	return ScanSucceeded, nil
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(ctx EnrichmentContext, image *storage.Image) (EnrichmentResult, error) {
	errorList := errorhelpers.NewErrorList("image enrichment")

	imageNoteSet := make(map[storage.Image_Note]struct{}, len(image.Notes))
	for _, note := range image.Notes {
		imageNoteSet[note] = struct{}{}
	}

	updatedMetadata, err := e.enrichWithMetadata(ctx, image)
	errorList.AddError(err)
	if image.GetMetadata() == nil {
		imageNoteSet[storage.Image_MISSING_METADATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_METADATA)
	}

	scanResult, err := e.enrichWithScan(ctx, image)
	errorList.AddError(err)
	if scanResult == ScanNotDone && image.GetScan() == nil {
		imageNoteSet[storage.Image_MISSING_SCAN_DATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_SCAN_DATA)
	}

	image.Notes = image.Notes[:0]
	for note := range imageNoteSet {
		image.Notes = append(image.Notes, note)
	}

	e.cvesSuppressor.EnrichImageWithSuppressedCVEs(image)
	if features.VulnRiskManagement.Enabled() {
		e.cvesSuppressorV2.EnrichImageWithSuppressedCVEs(image)
	}

	return EnrichmentResult{
		ImageUpdated: updatedMetadata || (scanResult != ScanNotDone),
		ScanResult:   scanResult,
	}, errorList.ToError()
}

func (e *enricherImpl) enrichWithMetadata(ctx EnrichmentContext, image *storage.Image) (bool, error) {
	// Attempt to short-circuit before checking registries.
	if ctx.FetchOnlyIfMetadataEmpty() && image.GetMetadata() != nil {
		return false, nil
	}
	if ctx.FetchOpt != ForceRefetch {
		if metadataValue := e.metadataCache.Get(getRef(image)); metadataValue != nil {
			e.metrics.IncrementMetadataCacheHit()
			image.Metadata = metadataValue.(*storage.ImageMetadata).Clone()
			return true, nil
		}
		e.metrics.IncrementMetadataCacheMiss()
	}
	if ctx.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error getting metadata for image: %s", image.GetName().GetFullName()))

	if image.GetName().GetRegistry() == "" {
		errorList.AddError(errors.New("no registry is indicated for image"))
		return false, errorList.ToError()
	}

	registries := e.integrations.RegistrySet()
	if !ctx.Internal && registries.IsEmpty() {
		errorList.AddError(errors.Errorf("no image registries are integrated: please add an image integration for %s", image.GetName().GetRegistry()))
		return false, errorList.ToError()
	}

	log.Infof("Getting metadata for image %s", image.GetName().GetFullName())
	for _, registry := range registries.GetAll() {
		updated, err := e.enrichImageWithRegistry(image, registry)
		if err != nil {
			var currentRegistryErrors int32
			concurrency.WithLock(&e.registryErrorsLock, func() {
				currentRegistryErrors = e.errorsPerRegistry[registry] + 1
				e.errorsPerRegistry[registry] = currentRegistryErrors
			})

			if currentRegistryErrors >= consecutiveErrorThreshold { // update health
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
					Id:            registry.DataSource().Id,
					Name:          registry.DataSource().Name,
					Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
					Status:        storage.IntegrationHealth_UNHEALTHY,
					LastTimestamp: timestamp.TimestampNow(),
					ErrorMessage:  err.Error(),
				})
			}
			errorList.AddError(err)
			continue
		}
		if updated {
			var currentRegistryErrors int32
			concurrency.WithRLock(&e.registryErrorsLock, func() {
				currentRegistryErrors = e.errorsPerRegistry[registry]
			})
			if currentRegistryErrors > 0 {
				concurrency.WithLock(&e.registryErrorsLock, func() {
					if e.errorsPerRegistry[registry] != currentRegistryErrors {
						return
					}
					e.errorsPerRegistry[registry] = 0
				})
			}
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
				Id:            registry.DataSource().Id,
				Name:          registry.DataSource().Name,
				Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
				Status:        storage.IntegrationHealth_HEALTHY,
				LastTimestamp: timestamp.TimestampNow(),
				ErrorMessage:  "",
			})
			return true, nil
		}
	}

	if !ctx.Internal && len(errorList.ErrorStrings()) == 0 {
		errorList.AddError(errors.Errorf("no matching image registries found: please add an image integration for %s", image.GetName().GetRegistry()))
	}

	return false, errorList.ToError()
}

func getRef(image *storage.Image) string {
	if image.GetId() != "" {
		return image.GetId()
	}
	return image.GetName().GetFullName()
}

func (e *enricherImpl) enrichImageWithRegistry(image *storage.Image, registry registryTypes.ImageRegistry) (bool, error) {
	if !registry.Global() {
		return false, nil
	}
	if !registry.Match(image.GetName()) {
		return false, nil
	}

	// Wait until limiter allows entrance
	_ = e.metadataLimiter.Wait(context.Background())
	metadata, err := registry.Metadata(image)
	if err != nil {
		return false, errors.Wrapf(err, "error getting metadata from registry: %q", registry.Name())
	}
	metadata.DataSource = registry.DataSource()
	image.Metadata = metadata

	cachedMetadata := metadata.Clone()
	e.metadataCache.Add(getRef(image), cachedMetadata)
	if image.GetId() == "" {
		if digest := image.Metadata.GetV2().GetDigest(); digest != "" {
			e.metadataCache.Add(digest, cachedMetadata)
		}
		if digest := image.Metadata.GetV1().GetDigest(); digest != "" {
			e.metadataCache.Add(digest, cachedMetadata)
		}
	}
	return true, nil
}

func (e *enricherImpl) enrichWithScan(ctx EnrichmentContext, image *storage.Image) (ScanResult, error) {
	// Attempt to short-circuit before checking scanners.
	if ctx.FetchOnlyIfScanEmpty() && image.GetScan() != nil {
		return ScanNotDone, nil
	}
	if e.populateFromCache(ctx, image) {
		return ScanSucceeded, nil
	}
	if ctx.FetchOpt == NoExternalMetadata {
		return ScanNotDone, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning image: %s", image.GetName().GetFullName()))
	scanners := e.integrations.ScannerSet()
	if !ctx.Internal && scanners.IsEmpty() {
		errorList.AddError(errors.New("no image scanners are integrated"))
		return ScanNotDone, errorList.ToError()
	}

	for _, scanner := range scanners.GetAll() {
		result, err := e.enrichImageWithScanner(ctx, image, scanner)
		if err != nil {
			var currentScannerErrors int32
			concurrency.WithLock(&e.scannerErrorsLock, func() {
				currentScannerErrors = e.errorsPerScanner[scanner] + 1
				e.errorsPerScanner[scanner] = currentScannerErrors
			})
			if currentScannerErrors >= consecutiveErrorThreshold { // update health
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
					Id:            scanner.DataSource().Id,
					Name:          scanner.DataSource().Name,
					Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
					Status:        storage.IntegrationHealth_UNHEALTHY,
					LastTimestamp: timestamp.TimestampNow(),
					ErrorMessage:  err.Error(),
				})
			}
			errorList.AddError(err)
			continue
		}
		if result != ScanNotDone {
			var currentScannerErrors int32
			concurrency.WithRLock(&e.scannerErrorsLock, func() {
				currentScannerErrors = e.errorsPerScanner[scanner]
			})
			if currentScannerErrors > 0 {
				concurrency.WithLock(&e.scannerErrorsLock, func() {
					if e.errorsPerScanner[scanner] != currentScannerErrors {
						return
					}
					e.errorsPerScanner[scanner] = 0
				})
			}
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
				Id:            scanner.DataSource().Id,
				Name:          scanner.DataSource().Name,
				Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
				Status:        storage.IntegrationHealth_HEALTHY,
				LastTimestamp: timestamp.TimestampNow(),
				ErrorMessage:  "",
			})
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
	image.Scan = scanValue.(*storage.ImageScan).Clone()
	FillScanStats(image)
	return true
}

func normalizeVulnerabilities(scan *storage.ImageScan) {
	for _, c := range scan.GetComponents() {
		for _, v := range c.GetVulns() {
			v.Severity = cvss.VulnToSeverity(v)
		}
	}
}

func (e *enricherImpl) enrichImageWithScanner(ctx EnrichmentContext, image *storage.Image, scanner scannerTypes.ImageScanner) (ScanResult, error) {
	if !scanner.Match(image.GetName()) {
		return ScanNotDone, nil
	}

	var scan *storage.ImageScan

	if asyncScanner, ok := scanner.(scannerTypes.AsyncScanner); ok && ctx.UseNonBlockingCallsWherePossible {
		_ = e.asyncRateLimiter.Wait(context.Background())

		var err error
		scan, err = asyncScanner.GetOrTriggerScan(image)
		if err != nil {
			return ScanNotDone, errors.Wrapf(err, "Error triggering scan for %q with scanner %q", image.GetName().GetFullName(), scanner.Name())
		}
		if scan == nil {
			return ScanTriggered, nil
		}
	} else {
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

	e.enrichImage(image, scan, scanner.DataSource())

	return ScanSucceeded, nil
}

func (e *enricherImpl) enrichImage(image *storage.Image, scan *storage.ImageScan, dataSource *storage.DataSource) {
	// Normalize the vulnerabilities.
	normalizeVulnerabilities(scan)

	scan.DataSource = dataSource

	// Assume:
	//  scan != nil
	//  no error scanning.
	image.Scan = scan
	FillScanStats(image)

	cachedScan := prepareImageScanForCache(scan)
	e.scanCache.Add(getRef(image), cachedScan)

	if image.GetId() == "" {
		if digest := image.GetMetadata().GetV2().GetDigest(); digest != "" {
			e.scanCache.Add(digest, cachedScan)
		}
		if digest := image.GetMetadata().GetV1().GetDigest(); digest != "" {
			e.scanCache.Add(digest, cachedScan)
		}
	}
}

func prepareImageScanForCache(scan *storage.ImageScan) *storage.ImageScan {
	// Clone the cachedScan because the scan is used within the image leading to race conditions
	clonedScan := scan.Clone()
	// Executables are processed and stored in active component updater executable cache and hence we remove them here.
	for _, component := range clonedScan.GetComponents() {
		component.Executables = nil
	}
	return clonedScan
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
