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
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/registries"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/time/rate"
)

const (
	// The number of consecutive errors for a scanner or registry that cause its health status to be UNHEALTHY
	consecutiveErrorThreshold = 3
)

var (
	_ ImageEnricher = (*enricherImpl)(nil)
)

type enricherImpl struct {
	cvesSuppressor   CVESuppressor
	cvesSuppressorV2 CVESuppressor
	integrations     integration.Set

	errorsPerRegistry  map[registryTypes.ImageRegistry]int32
	registryErrorsLock sync.RWMutex
	errorsPerScanner   map[scannerTypes.ImageScannerWithDataSource]int32
	scannerErrorsLock  sync.RWMutex

	integrationHealthReporter integrationhealth.Reporter ``

	metadataLimiter *rate.Limiter
	metadataCache   expiringcache.Cache

	signatureIntegrationGetter SignatureIntegrationGetter
	signatureVerifier          signatureVerifierForIntegrations
	signatureFetcher           signatures.SignatureFetcher

	imageGetter ImageGetter

	asyncRateLimiter *rate.Limiter

	metrics metrics
}

// EnrichWithVulnerabilities enriches the given image with vulnerabilities.
func (e *enricherImpl) EnrichWithVulnerabilities(image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (EnrichmentResult, error) {
	scanners := e.integrations.ScannerSet()
	if scanners.IsEmpty() {
		return EnrichmentResult{
			ScanResult: ScanNotDone,
		}, errors.New("no image scanners are integrated")
	}

	for _, imageScanner := range scanners.GetAll() {
		scanner := imageScanner.GetScanner()
		if vulnScanner, ok := scanner.(scannerTypes.ImageVulnerabilityGetter); ok {
			// Clairify is the only supported ImageVulnerabilityGetter at this time.
			if scanner.Type() != clairify.TypeString {
				log.Errorf("unexpected image vulnerability getter: %s [%s]", scanner.Name(), scanner.Type())
				continue
			}

			res, err := e.enrichWithVulnerabilities(scanner.Name(), imageScanner.DataSource(), vulnScanner, image, components, notes)
			if err != nil {
				return EnrichmentResult{
					ScanResult: ScanNotDone,
				}, errors.Wrapf(err, "retrieving image vulnerabilities from %s [%s]", scanner.Name(), scanner.Type())
			}

			return EnrichmentResult{
				ImageUpdated: res != ScanNotDone,
				ScanResult:   res,
			}, nil
		}
	}

	return EnrichmentResult{
		ScanResult: ScanNotDone,
	}, errors.New("no image vulnerability retrievers are integrated")
}

func (e *enricherImpl) enrichWithVulnerabilities(scannerName string, dataSource *storage.DataSource, scanner scannerTypes.ImageVulnerabilityGetter,
	image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (ScanResult, error) {
	scanStartTime := time.Now()
	scan, err := scanner.GetVulnerabilities(image, components, notes)
	e.metrics.SetImageVulnerabilityRetrievalTime(scanStartTime, scannerName, err)
	if err != nil || scan == nil {
		return ScanNotDone, err
	}

	enrichImage(image, scan, dataSource)

	return ScanSucceeded, nil
}

func (e *enricherImpl) EnrichWithSignatureVerificationData(ctx context.Context, image *storage.Image) (EnrichmentResult, error) {
	if !features.ImageSignatureVerification.Enabled() {
		return EnrichmentResult{}, errors.New("the image signature verification feature is not enabled")
	}

	updated, err := e.enrichWithSignatureVerificationData(ctx, EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly}, image)

	return EnrichmentResult{
		ImageUpdated: updated,
	}, err
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(ctx context.Context, enrichContext EnrichmentContext, image *storage.Image) (EnrichmentResult, error) {
	errorList := errorhelpers.NewErrorList("image enrichment")

	imageNoteSet := make(map[storage.Image_Note]struct{}, len(image.Notes))
	for _, note := range image.Notes {
		imageNoteSet[note] = struct{}{}
	}

	// Signals whether any updates to the image were made throughout the enrichment flow.
	var updated bool

	didUpdateMetadata, err := e.enrichWithMetadata(ctx, enrichContext, image)
	errorList.AddError(err)
	if image.GetMetadata() == nil {
		imageNoteSet[storage.Image_MISSING_METADATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_METADATA)
	}
	updated = updated || didUpdateMetadata

	// Update the image with existing values depending on the FetchOption provided or whether any are available.
	// This makes sure that we fetch any existing image only once from database.
	useExistingScanIfPossible := e.updateImageFromDatabase(ctx, image, enrichContext.FetchOpt)

	scanResult, err := e.enrichWithScan(ctx, enrichContext, image, useExistingScanIfPossible)
	errorList.AddError(err)
	if scanResult == ScanNotDone && image.GetScan() == nil {
		imageNoteSet[storage.Image_MISSING_SCAN_DATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_SCAN_DATA)
	}
	updated = updated || scanResult != ScanNotDone

	if features.ImageSignatureVerification.Enabled() {
		didUpdateSignature, err := e.enrichWithSignature(ctx, enrichContext, image)
		errorList.AddError(err)
		if len(image.GetSignature().GetSignatures()) == 0 {
			imageNoteSet[storage.Image_MISSING_SIGNATURE] = struct{}{}
		} else {
			delete(imageNoteSet, storage.Image_MISSING_SIGNATURE)
		}
		updated = updated || didUpdateSignature

		didUpdateSigVerificationData, err := e.enrichWithSignatureVerificationData(ctx, enrichContext, image)
		errorList.AddError(err)
		if len(image.GetSignatureVerificationData().GetResults()) == 0 {
			imageNoteSet[storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA] = struct{}{}
		} else {
			delete(imageNoteSet, storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA)
		}
		updated = updated || didUpdateSigVerificationData
	}

	image.Notes = image.Notes[:0]
	for note := range imageNoteSet {
		image.Notes = append(image.Notes, note)
	}

	e.cvesSuppressor.EnrichImageWithSuppressedCVEs(image)
	e.cvesSuppressorV2.EnrichImageWithSuppressedCVEs(image)

	return EnrichmentResult{
		ImageUpdated: updated,
		ScanResult:   scanResult,
	}, errorList.ToError()
}

// updateImageFromDatabase will update the values of the given image from an existing image within the database
// depending on whether the values exist and the given FetchOption allows using existing values.
// It will return a bool indicating whether existing values from database will be used for the signature.
func (e *enricherImpl) updateImageFromDatabase(ctx context.Context, img *storage.Image, option FetchOption) bool {
	existingImg, exists := e.fetchFromDatabase(ctx, img, option)
	// Short-circuit if no image exists or the FetchOption specifies to not use existing values.
	if !exists {
		return false
	}

	usesExistingScan := e.useExistingScan(img, existingImg, option)
	e.useExistingSignature(img, existingImg, option)
	e.useExistingSignatureVerificationData(img, existingImg, option)

	return usesExistingScan
}

func (e *enricherImpl) enrichWithMetadata(ctx context.Context, enrichmentContext EnrichmentContext, image *storage.Image) (bool, error) {
	// Attempt to short-circuit before checking registries.
	metadataOutOfDate := metadataIsOutOfDate(image.GetMetadata())
	if !metadataOutOfDate {
		return false, nil
	}

	if enrichmentContext.FetchOpt != ForceRefetch {
		// The metadata in the cache is always up-to-date with respect to the current metadataVersion
		if metadataValue := e.metadataCache.Get(getRef(image)); metadataValue != nil {
			e.metrics.IncrementMetadataCacheHit()
			image.Metadata = metadataValue.(*storage.ImageMetadata).Clone()
			return true, nil
		}
		e.metrics.IncrementMetadataCacheMiss()
	}
	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error getting metadata for image: %s", image.GetName().GetFullName()))

	if err := e.checkRegistryForImage(image); err != nil {
		errorList.AddError(err)
		return false, errorList.ToError()
	}

	registrySet, err := e.getRegistriesForContext(enrichmentContext)
	if err != nil {
		errorList.AddError(err)
		return false, errorList.ToError()
	}

	log.Infof("Getting metadata for image %s", image.GetName().GetFullName())
	for _, registry := range registrySet.GetAll() {
		updated, err := e.enrichImageWithRegistry(ctx, image, registry)
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

	if !enrichmentContext.Internal && errorList.Empty() {
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

func (e *enricherImpl) enrichImageWithRegistry(ctx context.Context, image *storage.Image, registry registryTypes.ImageRegistry) (bool, error) {
	if !registry.Match(image.GetName()) {
		return false, nil
	}

	// Wait until limiter allows entrance
	err := e.metadataLimiter.Wait(ctx)
	if err != nil {
		return false, errors.Wrap(err, "waiting for metadata limiter")
	}
	metadata, err := registry.Metadata(image)
	if err != nil {
		return false, errors.Wrapf(err, "getting metadata from registry: %q", registry.Name())
	}
	metadata.DataSource = registry.DataSource()
	metadata.Version = metadataVersion
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

func (e *enricherImpl) fetchFromDatabase(ctx context.Context, img *storage.Image, option FetchOption) (*storage.Image, bool) {
	if option.forceRefetchCachedValues() {
		// When refetched values should be used, reset the existing values for signature and signature verification data.
		img.Signature = nil
		img.SignatureVerificationData = nil
		return img, false
	}
	// See if the image exists in the DB with a scan, if it does, then use that instead of fetching
	id := utils.GetImageID(img)
	if id == "" {
		return img, false
	}
	existingImage, exists, err := e.imageGetter(sac.WithAllAccess(ctx), id)
	if err != nil {
		log.Errorf("error fetching image %q: %v", id, err)
		return img, false
	}
	return existingImage, exists
}

func (e *enricherImpl) useExistingScan(img *storage.Image, existingImg *storage.Image, option FetchOption) bool {
	if option == ForceRefetchScansOnly {
		return false
	}

	if existingImg.GetScan() != nil {
		img.Scan = existingImg.GetScan()
		return true
	}

	return false
}

func (e *enricherImpl) useExistingSignature(img *storage.Image, existingImg *storage.Image, option FetchOption) {
	if option == ForceRefetchSignaturesOnly {
		// When forced to refetch values, disregard existing ones.
		img.Signature = nil
		return
	}

	if existingImg.GetSignature() != nil {
		img.Signature = existingImg.GetSignature()
	}
}

func (e *enricherImpl) useExistingSignatureVerificationData(img *storage.Image, existingImg *storage.Image, option FetchOption) {
	if option == ForceRefetchSignaturesOnly {
		// When forced to refetch values, disregard existing ones.
		img.SignatureVerificationData = nil
		return
	}

	if existingImg.GetSignatureVerificationData() != nil {
		img.SignatureVerificationData = existingImg.GetSignatureVerificationData()
	}
}

func (e *enricherImpl) enrichWithScan(ctx context.Context, enrichmentContext EnrichmentContext,
	image *storage.Image, useExistingScan bool) (ScanResult, error) {
	// Short-circuit if we are using existing values.
	// We need to have a distinction between existing image values set
	// from database and existing values from the received image. Existing image values set by database indicate the
	// ScanResult ScanSucceeded, whereas existing values on the image that have not been set from database indicate the
	// ScanResult ScanNotDone.
	if useExistingScan {
		return ScanSucceeded, nil
	}

	// Attempt to short-circuit before checking scanners.
	if enrichmentContext.FetchOnlyIfScanEmpty() && image.GetScan() != nil {
		return ScanNotDone, nil
	}

	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return ScanNotDone, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning image: %s", image.GetName().GetFullName()))
	scanners := e.integrations.ScannerSet()
	if !enrichmentContext.Internal && scanners.IsEmpty() {
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

// enrichWithSignatureVerificationData enriches the image with signature verification data and returns a bool,
// indicating whether any verification data was added to the image or not.
// Based on the given FetchOption, it will try to short-circuit using values from existing images where
// possible.
func (e *enricherImpl) enrichWithSignatureVerificationData(ctx context.Context, enrichmentContext EnrichmentContext,
	img *storage.Image) (bool, error) {
	// If no external metadata should be taken into account, we skip the verification.
	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	imgName := img.GetName().GetFullName()

	// Short-circuit if no signature is available.
	if len(img.GetSignature().GetSignatures()) == 0 {
		// If no signature is given but there are signature verification results on the image, make sure we delete
		// the stale signature verification results.
		if len(img.GetSignatureVerificationData().GetResults()) != 0 {
			img.SignatureVerificationData = nil
			log.Debugf("No signatures associated with image %q but existing results were found, "+
				"deleting those.", imgName)
			return true, nil
		}

		log.Debugf("No signatures associated with image %q so no verification will be done", imgName)

		return false, nil
	}

	// Fetch signature integrations from the data store.
	sigIntegrations, err := e.signatureIntegrationGetter(sac.WithAllAccess(ctx))
	if err != nil {
		return false, errors.Wrap(err, "fetching signature integrations")
	}

	// Short-circuit if no integrations are available.
	if len(sigIntegrations) == 0 {
		// If no signature integrations are available, we need to delete any existing verification results from the
		// image and signal an update to the verification data.
		if len(img.GetSignatureVerificationData().GetResults()) != 0 {
			img.SignatureVerificationData = nil
			log.Debugf("No signature integrations available but image %q had existing results, "+
				"deleting those", imgName)
			return true, nil
		}
		// If no integrations are available and no pre-existing results, short-circuit and don't signal updated
		// verification results.
		log.Debug("No signature integration available so no verification will be done")
		return false, nil
	}

	// The image will use cached or existing values. If we are enriching during i.e. change of signature integration,
	// we have to make sure we force a re-verification to not return stale data.
	if img.GetSignatureVerificationData() != nil && enrichmentContext.FetchOpt != ForceRefetchSignaturesOnly {
		return false, nil
	}

	// Timeout is based on benchmark test result for 200 integrations with 1 config each (roughly 0.1 sec) + a grace
	// timeout on top. Currently, signature verification is done without remote RPCs, this will need to be
	// changed accordingly when RPCs are required (i.e. cosign keyless).
	verifySignatureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	res := e.signatureVerifier(verifySignatureCtx, sigIntegrations, img)
	if res == nil {
		return false, ctx.Err()
	}

	log.Debugf("Verification results found for image %q: %+v", imgName, res)

	img.SignatureVerificationData = &storage.ImageSignatureVerificationData{
		Results: res,
	}
	return true, nil
}

// enrichWithSignature enriches the image with a signature and returns a bool, indicating whether a signature has been
// updated on the image or not.
// Based on the given FetchOption, it will try to short-circuit using values from existing images where
// possible.
func (e *enricherImpl) enrichWithSignature(ctx context.Context, enrichmentContext EnrichmentContext,
	img *storage.Image) (bool, error) {
	// If no external metadata should be taken into account, we skip the fetching of signatures.
	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	// Short-circuit if possible when we are using existing values.
	if img.GetSignature() != nil {
		return false, nil
	}

	imgName := img.GetName().GetFullName()

	if err := e.checkRegistryForImage(img); err != nil {
		return false, errors.Wrapf(err, "checking registry for image %q", imgName)
	}

	registrySet, err := e.getRegistriesForContext(enrichmentContext)
	if err != nil {
		return false, errors.Wrap(err, "getting registries for context")
	}

	matchingRegistries, err := getMatchingRegistries(registrySet.GetAll(), img)
	if err != nil {
		return false, errors.Wrapf(err, "getting matching registries for image %q", imgName)
	}

	var fetchedSignatures []*storage.Signature
	for _, matchingReg := range matchingRegistries {
		// FetchImageSignaturesWithRetries will try fetching of signatures with retries.
		sigs, err := signatures.FetchImageSignaturesWithRetries(ctx, e.signatureFetcher, img, matchingReg)
		fetchedSignatures = append(fetchedSignatures, sigs...)
		// Skip other matching registries if we have a successful fetch of signatures, irrespective of whether
		// signatures were found or not. Retrying this for other registries won't change the fact that signatures are
		// available or not.
		if err == nil {
			break
		}

		// We skip logging unauthorized errors. Each matching registry may either provide no credentials or different
		// credentials, which makes it expected that we receive unauthorized errors on multiple occasions.
		// The best way to handle this would be to keep a list of images which are matching but not authorized for each
		// registry, but this can be tackled at a latter improvement.
		if !errors.Is(err, errox.NotAuthorized) {
			log.Errorf("Error fetching image signatures for image %q: %v", imgName, err)
		} else {
			// Log errox.NotAuthorized erros only in debug mode, since we expect them to occur often.
			log.Debugf("Unauthorized error fetching image signatures for image %q: %v",
				imgName, err)
		}
	}

	// Do not signal updates when no signatures have been fetched.
	if len(fetchedSignatures) == 0 {
		// Delete existing signatures on the image if we fetched zero.
		if len(img.GetSignature().GetSignatures()) != 0 {
			log.Debugf("No signatures found but image %q had existing signatures, deleting those", imgName)
			img.Signature = nil
			return true, nil
		}
		log.Debugf("No signatures associated with image %q", imgName)
		return false, nil
	}

	log.Debugf("Found signatures for image %q: %+v", imgName, fetchedSignatures)

	img.Signature = &storage.ImageSignature{
		Signatures: fetchedSignatures,
		Fetched:    protoconv.ConvertTimeToTimestamp(time.Now()),
	}
	return true, nil
}

func (e *enricherImpl) checkRegistryForImage(image *storage.Image) error {
	if image.GetName().GetRegistry() == "" {
		return errox.InvalidArgs.CausedBy(fmt.Sprintf("no registry is indicated for image %q",
			image.GetName().GetFullName()))
	}
	return nil
}

func (e *enricherImpl) getRegistriesForContext(ctx EnrichmentContext) (registries.Set, error) {
	registrySet := e.integrations.RegistrySet()
	if ctx.Internal {
		return registrySet, nil
	}

	if registrySet.IsEmpty() {
		return nil, errox.NotFound.CausedBy("no image registries are integrated: please add an image integration")
	}

	return registrySet, nil
}

func getMatchingRegistries(registries []registryTypes.ImageRegistry,
	image *storage.Image) ([]registryTypes.ImageRegistry, error) {
	var matchingRegistries []registryTypes.ImageRegistry
	for _, registry := range registries {
		if registry.Match(image.GetName()) {
			matchingRegistries = append(matchingRegistries, registry)
		}
	}

	if len(matchingRegistries) == 0 {
		return nil, errox.NotFound.CausedBy(fmt.Sprintf("no matching registries found: please add "+
			"an image integration for %q", image.GetName().GetFullName()))
	}

	return matchingRegistries, nil
}

func normalizeVulnerabilities(scan *storage.ImageScan) {
	for _, c := range scan.GetComponents() {
		for _, v := range c.GetVulns() {
			v.Severity = cvss.VulnToSeverity(v)
		}
	}
}

func (e *enricherImpl) enrichImageWithScanner(ctx context.Context, image *storage.Image, imageScanner scannerTypes.ImageScannerWithDataSource) (ScanResult, error) {
	scanner := imageScanner.GetScanner()

	if !scanner.Match(image.GetName()) {
		return ScanNotDone, nil
	}

	sema := scanner.MaxConcurrentScanSemaphore()
	err := sema.Acquire(ctx, 1)
	if err != nil {
		return ScanNotDone, err
	}
	defer sema.Release(1)

	scanStartTime := time.Now()
	scan, err := scanner.GetScan(image)
	e.metrics.SetScanDurationTime(scanStartTime, scanner.Name(), err)
	if err != nil {
		return ScanNotDone, errors.Wrapf(err, "scanning %q with scanner %q", image.GetName().GetFullName(), scanner.Name())
	}
	if scan == nil {
		return ScanNotDone, nil
	}

	enrichImage(image, scan, imageScanner.DataSource())

	return ScanSucceeded, nil
}

func enrichImage(image *storage.Image, scan *storage.ImageScan, dataSource *storage.DataSource) {
	// Normalize the vulnerabilities.
	normalizeVulnerabilities(scan)

	scan.DataSource = dataSource

	// Assume:
	//  scan != nil
	//  no error scanning.
	image.Scan = scan
	FillScanStats(image)
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
