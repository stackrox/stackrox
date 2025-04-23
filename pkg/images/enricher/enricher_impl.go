package enricher

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/cache"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/openshift"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protoutils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scanners/types"
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

	integrationHealthReporter integrationhealth.Reporter

	metadataLimiter *rate.Limiter
	metadataCache   cache.ImageMetadata

	signatureIntegrationGetter SignatureIntegrationGetter
	signatureVerifier          signatureVerifierForIntegrations
	signatureFetcher           signatures.SignatureFetcher

	imageGetter ImageGetter

	asyncRateLimiter *rate.Limiter

	metrics metrics

	scanDelegator delegatedregistry.Delegator
}

// EnrichWithVulnerabilities enriches the given image with vulnerabilities.
func (e *enricherImpl) EnrichWithVulnerabilities(image *storage.Image, components *scannerTypes.ScanComponents, notes []scannerV1.Note) (EnrichmentResult, error) {
	scanners := e.integrations.ScannerSet()
	if scanners.IsEmpty() {
		return EnrichmentResult{
			ScanResult: ScanNotDone,
		}, errors.New("no image scanners are integrated")
	}

	for _, imageScanner := range scanners.GetAll() {
		scanner := imageScanner.GetScanner()
		if vulnScanner, ok := scanner.(scannerTypes.ImageVulnerabilityGetter); ok {
			if scanner.Type() != components.ScannerType() {
				log.Debugf("Skipping scanner %q with type %q, components are meant for scanner type %q for image: %q", scanner.Name(), scanner.Type(), components.ScannerType(), image.GetName().GetFullName())
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
	image *storage.Image, components *scannerTypes.ScanComponents, notes []scannerV1.Note) (ScanResult, error) {
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
	updated, err := e.enrichWithSignatureVerificationData(ctx, EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly}, image)

	return EnrichmentResult{
		ImageUpdated: updated,
	}, err
}

// delegateEnrichImage returns true if enrichment for this image should be delegated (enriched via Sensor). If true
// and no error image was enriched successfully.
func (e *enricherImpl) delegateEnrichImage(ctx context.Context, enrichCtx EnrichmentContext, image *storage.Image) (bool, error) {
	if !enrichCtx.Delegable {
		// Request should not be delegated.
		return false, nil
	}

	var shouldDelegate bool
	var err error
	clusterID := enrichCtx.ClusterID
	if clusterID == "" {
		clusterID, shouldDelegate, err = e.scanDelegator.GetDelegateClusterID(ctx, image.GetName())
	} else {
		// A cluster ID has been passed to the enricher, determine if it's valid for delegation.
		err = e.scanDelegator.ValidateCluster(enrichCtx.ClusterID)
		shouldDelegate = true
	}

	if err != nil || !shouldDelegate {
		// If was an error or should not delegate, short-circuit.
		return shouldDelegate, err
	}

	// Check if image exists in database (will include metadata, sigs, etc.).
	// Ignores in-mem metadata cache because that is not populated via
	// enrichment requests from secured clusters. fetchFromDatabase will check
	// if FetchOpt forces refetch. Assumes signatures in DB are OK, reprocessing
	// or forcing re-scan will trigger updates as necessary.
	existingImg, exists := e.fetchFromDatabase(ctx, image, enrichCtx.FetchOpt)
	if exists && cachedImageIsValid(existingImg) {
		updated := e.updateImageWithExistingImage(image, existingImg, enrichCtx.FetchOpt)
		if updated {
			e.cvesSuppressor.EnrichImageWithSuppressedCVEs(image)
			e.cvesSuppressorV2.EnrichImageWithSuppressedCVEs(image)
			// Errors for signature verification will be logged, so we can safely ignore them for the time being.
			_, _ = e.enrichWithSignatureVerificationData(ctx, enrichCtx, image)

			log.Debugf("Delegated enrichment returning cached image for %q", image.GetName().GetFullName())
			return true, nil
		}
	}

	// Send image to secured cluster for enrichment.
	force := enrichCtx.FetchOpt.forceRefetchCachedValues() || enrichCtx.FetchOpt == UseImageNamesRefetchCachedValues
	scannedImage, err := e.scanDelegator.DelegateScanImage(ctx, image.GetName(), clusterID, force)
	if err != nil {
		return true, err
	}

	// Copy the fields from scannedImage into image, EnrichImage expecting modification in place
	image.Reset()
	protocompat.Merge(image, scannedImage)

	e.cvesSuppressor.EnrichImageWithSuppressedCVEs(image)
	e.cvesSuppressorV2.EnrichImageWithSuppressedCVEs(image)
	return true, nil
}

func cachedImageIsValid(cachedImage *storage.Image) bool {
	if cachedImage == nil {
		return false
	}

	if metadataIsOutOfDate(cachedImage.GetMetadata()) {
		return false
	}

	if cachedImage.GetScan() == nil {
		return false
	}

	return true
}

func (e *enricherImpl) updateImageWithExistingImage(image *storage.Image, existingImage *storage.Image, option FetchOption) bool {
	if option == IgnoreExistingImages {
		return false
	}

	if existingImage.GetMetadata() != nil {
		image.Metadata = existingImage.GetMetadata()
	}
	image.Notes = existingImage.GetNotes()
	hasChangedNames := !protoutils.SlicesEqual(existingImage.GetNames(), image.GetNames())
	image.Names = protoutils.SliceUnique(append(existingImage.GetNames(), image.GetNames()...))

	e.useExistingSignature(image, existingImage, option)
	e.useExistingSignatureVerificationData(image, existingImage, option, hasChangedNames)
	e.useExistingImageName(image, existingImage, option)
	return e.useExistingScan(image, existingImage, option)
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(ctx context.Context, enrichContext EnrichmentContext, image *storage.Image) (EnrichmentResult, error) {
	shouldDelegate, err := e.delegateEnrichImage(ctx, enrichContext, image)
	log.Info("here")
	var delegateErr error
	if shouldDelegate {
		log.Info("here")
		if err == nil {
			log.Info("here")
			return EnrichmentResult{ImageUpdated: true, ScanResult: ScanSucceeded}, nil
		}
		if errors.Is(err, delegatedregistry.ErrNoClusterSpecified) {
			// Log the warning and try to keep enriching
			log.Warnf("Skipping delegation for %q (ID %q): %v, enriching via Central", image.GetName().GetFullName(), image.GetId(), err)
			delegateErr = errors.New("no cluster specified for delegated scanning and Central scan attempt failed")
		} else {
			// This enrichment should have been delegated, short circuit.
			log.Info("here")
			return EnrichmentResult{ImageUpdated: false, ScanResult: ScanNotDone}, err
		}
	} else if err != nil {
		log.Warnf("Error attempting to delegate: %v", err)
	}

	errorList := errorhelpers.NewErrorList("image enrichment")
	imageNoteSet := make(map[storage.Image_Note]struct{}, len(image.Notes))
	for _, note := range image.Notes {
		imageNoteSet[note] = struct{}{}
	}

	// Ensure we set the correct image notes when returning, also during short-circuting.
	defer setImageNotes(image, imageNoteSet)

	// Signals whether any updates to the image were made throughout the enrichment flow.
	var updated bool

	didUpdateMetadata, err := e.enrichWithMetadata(ctx, enrichContext, image)
	if image.GetMetadata() == nil {
		imageNoteSet[storage.Image_MISSING_METADATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_METADATA)
	}

	// Short-circuit if image metadata could not be retrieved. This indicates that connection or authentication to the
	// registry could not be made. Instead of trying to scan the image / fetch signatures for it, we shall short-circuit
	// here.
	if err != nil {
		errorList.AddErrors(err, delegateErr)
		log.Info("here")
		return EnrichmentResult{ImageUpdated: didUpdateMetadata, ScanResult: ScanNotDone}, errorList.ToError()
	}

	updated = updated || didUpdateMetadata

	// Update the image with existing values depending on the FetchOption provided or whether any are available.
	// This makes sure that we fetch any existing image only once from database.
	useExistingScanIfPossible := e.updateImageFromDatabase(ctx, image, enrichContext.FetchOpt)

	log.Infof("useExistingScanIfPossible: %v", useExistingScanIfPossible)
	scanResult, err := e.enrichWithScan(ctx, enrichContext, image, useExistingScanIfPossible)
	errorList.AddError(err)
	if scanResult == ScanNotDone && image.GetScan() == nil {
		imageNoteSet[storage.Image_MISSING_SCAN_DATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_SCAN_DATA)
	}
	updated = updated || scanResult != ScanNotDone

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

	e.cvesSuppressor.EnrichImageWithSuppressedCVEs(image)
	e.cvesSuppressorV2.EnrichImageWithSuppressedCVEs(image)

	if !errorList.Empty() {
		errorList.AddError(delegateErr)
	}

	log.Info("here")
	log.Infof("result: %v", EnrichmentResult{
		ImageUpdated: updated,
		ScanResult:   scanResult,
	})
	return EnrichmentResult{
		ImageUpdated: updated,
		ScanResult:   scanResult,
	}, errorList.ToError()
}

func setImageNotes(image *storage.Image, imageNoteSet map[storage.Image_Note]struct{}) {
	image.Notes = image.Notes[:0]
	notes := make([]storage.Image_Note, 0, len(imageNoteSet))
	for note := range imageNoteSet {
		notes = append(notes, note)
	}
	sort.SliceStable(notes, func(i, j int) bool {
		return notes[i] < notes[j]
	})
	image.Notes = notes
}

// updateImageFromDatabase will update the values of the given image from an existing image within the database
// depending on whether the values exist and the given FetchOption allows using existing values.
// It will return a bool indicating whether existing values from database will be used for the signature.
func (e *enricherImpl) updateImageFromDatabase(ctx context.Context, img *storage.Image, option FetchOption) bool {
	if option == IgnoreExistingImages {
		return false
	}
	existingImg, exists := e.fetchFromDatabase(ctx, img, option)
	// Short-circuit if no image exists or the FetchOption specifies to not use existing values.
	if !exists {
		return false
	}

	return e.updateImageWithExistingImage(img, existingImg, option)
}

func (e *enricherImpl) enrichWithMetadata(ctx context.Context, enrichmentContext EnrichmentContext, image *storage.Image) (bool, error) {
	// Attempt to short-circuit before checking registries.
	metadataOutOfDate := metadataIsOutOfDate(image.GetMetadata())
	if !metadataOutOfDate {
		log.Infof("metadata was not out of date for image: %s", image.GetName())
		return false, nil
	}

	if !enrichmentContext.FetchOpt.forceRefetchCachedValues() &&
		enrichmentContext.FetchOpt != UseImageNamesRefetchCachedValues {
		// The metadata in the cache is always up-to-date with respect to the current metadataVersion
		if metadataValue, ok := e.metadataCache.Get(getRef(image)); ok {
			e.metrics.IncrementMetadataCacheHit()
			image.Metadata = metadataValue.CloneVT()
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

	registries, err := e.getRegistriesForContext(enrichmentContext)
	if err != nil {
		errorList.AddError(err)
		return false, errorList.ToError()
	}

	log.Infof("Getting metadata for image %q (ID %q)", image.GetName().GetFullName(), image.GetId())
	for _, registry := range registries {
		updated, err := e.enrichImageWithRegistry(ctx, image, registry)
		if err != nil {
			currentRegistryErrors := concurrency.WithLock1(&e.registryErrorsLock, func() int32 {
				currentRegistryErrors := e.errorsPerRegistry[registry] + 1
				e.errorsPerRegistry[registry] = currentRegistryErrors
				return currentRegistryErrors
			})

			if currentRegistryErrors >= consecutiveErrorThreshold { // update health
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
					Id:            registry.Source().GetId(),
					Name:          registry.Source().GetName(),
					Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
					Status:        storage.IntegrationHealth_UNHEALTHY,
					LastTimestamp: protocompat.TimestampNow(),
					ErrorMessage:  err.Error(),
				})
			}
			errorList.AddError(err)
			continue
		}
		if updated {
			currentRegistryErrors := concurrency.WithRLock1(&e.registryErrorsLock, func() int32 {
				return e.errorsPerRegistry[registry]
			})
			if currentRegistryErrors > 0 {
				concurrency.WithLock(&e.registryErrorsLock, func() {
					if e.errorsPerRegistry[registry] != currentRegistryErrors {
						return
					}
					e.errorsPerRegistry[registry] = 0
				})
			}
			id, name := registry.DataSource().GetId(), registry.DataSource().GetName()
			if features.SourcedAutogeneratedIntegrations.Enabled() {
				id, name = registry.Source().GetId(), registry.Source().GetName()
			}
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
				Id:            id,
				Name:          name,
				Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
				Status:        storage.IntegrationHealth_HEALTHY,
				LastTimestamp: protocompat.TimestampNow(),
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

func imageIntegrationToDataSource(i *storage.ImageIntegration) *storage.DataSource {
	return &storage.DataSource{
		Id:   i.GetId(),
		Name: i.GetName(),
	}
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
	if features.SourcedAutogeneratedIntegrations.Enabled() {
		metadata.DataSource = imageIntegrationToDataSource(registry.Source())
	}
	metadata.Version = metadataVersion
	image.Metadata = metadata

	cachedMetadata := metadata.CloneVT()
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
		// When re-fetched values should be used, reset the existing values for signature and signature verification data.
		img.Signature = nil
		img.SignatureVerificationData = nil
		return img, false
	}
	// See if the image exists in the DB with a scan, if it does, then use that instead of fetching
	id := utils.GetSHA(img)
	if id == "" {
		return img, false
	}
	existingImage, exists, err := e.imageGetter(sac.WithAllAccess(ctx), id)
	if err != nil {
		log.Errorf("error fetching image %q: %v", id, err)
		return img, false
	}

	// Special case: in the case we want to refetch cached values but retain the image names, we have to
	// first fetch the existing image, if it exists, merge the image names, and then return the modified
	// image. Currently, the scope of the option is to only be used by services which take in an external, "fresh"
	// image via API (e.g. by using roxctl). The option was created to not have an effect on performance for the
	// existing ForceRefetch and ForceRefetechCachedValuesOnly options and their related components,
	// i.e. the reprocessing loop.
	if option == UseImageNamesRefetchCachedValues {
		img.SignatureVerificationData = nil
		img.Signature = nil
		img.Names = protoutils.SliceUnique(append(existingImage.GetNames(), img.GetNames()...))
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

func (e *enricherImpl) useExistingSignatureVerificationData(img *storage.Image, existingImg *storage.Image,
	option FetchOption, hasChangedNames bool) {
	if option == ForceRefetchSignaturesOnly {
		// When forced to refetch values, disregard existing ones.
		img.SignatureVerificationData = nil
		return
	}

	// In case the existing image and the current image have a divergence in names, we will disregard existing
	// signature verification data, ensuring that we will always verify signatures, if any exist.
	if hasChangedNames {
		img.SignatureVerificationData = nil
		return
	}

	if existingImg.GetSignatureVerificationData() != nil {
		img.SignatureVerificationData = existingImg.GetSignatureVerificationData()
	}
}

func (e *enricherImpl) useExistingImageName(img *storage.Image, existingImg *storage.Image, option FetchOption) {
	// We only want to overwrite the top-level image name if we are ignoring cached values.
	if !option.forceRefetchCachedValues() {
		img.Name = existingImg.Name
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

	// verify that there is an integration of type scannerTypeHint
	if enrichmentContext.ScannerTypeHint != "" {
		found := false
		for _, scanner := range scanners.GetAll() {
			scannerType := scanner.GetScanner().Type()
			if scannerType == enrichmentContext.ScannerTypeHint {
				found = true
				break
			}
		}
		if !found {
			return ScanNotDone, errors.Errorf("no scanner integration found for scannerTypeHint %q", enrichmentContext.ScannerTypeHint)
		}
	}

	log.Debugf("Scanning image %q (ID %q)", image.GetName().GetFullName(), image.GetId())
	for _, scanner := range scanners.GetAll() {
		scannerType := scanner.GetScanner().Type()
		// only run scan with scanner specified in scannerTypeHint
		if enrichmentContext.ScannerTypeHint != "" && scannerType != enrichmentContext.ScannerTypeHint {
			continue
		}
		result, err := e.enrichImageWithScanner(ctx, image, scanner)
		if err != nil {
			currentScannerErrors := concurrency.WithLock1(&e.scannerErrorsLock, func() int32 {
				currentScannerErrors := e.errorsPerScanner[scanner] + 1
				e.errorsPerScanner[scanner] = currentScannerErrors
				return currentScannerErrors
			})
			if currentScannerErrors >= consecutiveErrorThreshold { // update health
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
					Id:            scanner.DataSource().Id,
					Name:          scanner.DataSource().Name,
					Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
					Status:        storage.IntegrationHealth_UNHEALTHY,
					LastTimestamp: protocompat.TimestampNow(),
					ErrorMessage:  err.Error(),
				})
			}
			errorList.AddError(err)

			if features.ScannerV4.Enabled() && scanner.GetScanner().Type() == types.ScannerV4 {
				// Do not try to scan with additional scanners if Scanner V4 enabled and fails to scan an image.
				// This would result in Clairify scanners being skipped per sorting logic in `GetAll` of
				// `pkg/scanners/set_impl.go`.
				log.Debugf("Scanner V4 encountered an error scanning image %q, skipping remaining scanners", image.GetName().GetFullName())
				break
			}
			continue
		}
		if result != ScanNotDone {
			currentScannerErrors := concurrency.WithRLock1(&e.scannerErrorsLock, func() int32 {
				return e.errorsPerScanner[scanner]
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
				LastTimestamp: protocompat.TimestampNow(),
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

	// Fetch signature integrations from the data store.
	sigIntegrations, err := e.signatureIntegrationGetter(sac.WithAllAccess(ctx))
	if err != nil {
		return false, errors.Wrap(err, "fetching signature integrations")
	}

	// Short-circuit if no integrations are available.
	if len(sigIntegrations) == 0 {
		// Contrary to the signature verification step we will retain existing signatures.
		return false, nil
	}

	imgName := img.GetName().GetFullName()

	if err := e.checkRegistryForImage(img); err != nil {
		return false, errors.Wrapf(err, "checking registry for image %q", imgName)
	}

	registries, err := e.getRegistriesForContext(enrichmentContext)
	if err != nil {
		return false, errors.Wrap(err, "getting registries for context")
	}

	if err := checkForMatchingImageIntegrations(registries, img); err != nil {
		// Do not return an error for internal images when no integration is found.
		if enrichmentContext.Internal {
			return false, nil
		}
		return false, errors.Wrapf(err, "checking for matching registries for image %q", imgName)
	}

	var fetchedSignatures []*storage.Signature
	for _, name := range img.GetNames() {
		matchingImageIntegrations := integration.GetMatchingImageIntegrations(ctx, registries, name)
		if len(matchingImageIntegrations) == 0 {
			// Instead of propagating an error and stopping the image enrichment, we will instead log the occurrence
			// and skip fetching signatures for this particular image name. We know that we have at least one matching
			// image registry due to the call to checkForMatchingImageIntegrations above, so we do not want to abort
			// enriching the image completely.
			log.Infof("No matching image integration found for image name %q, hence no signatures will be "+
				"attempted to be fetched", name)
			continue
		}

		for _, registry := range matchingImageIntegrations {
			// FetchImageSignaturesWithRetries will try fetching of signatures with retries.
			sigs, err := signatures.FetchImageSignaturesWithRetries(ctx, e.signatureFetcher, img, name.GetFullName(),
				registry)
			fetchedSignatures = append(fetchedSignatures, sigs...)
			// Skip other matching image integrations if we have a successful fetch of signatures for the respective
			// image name, irrespective of whether signatures were found or not.
			// Retrying this for other image integrations won't change the fact that signatures are available or not for
			// this particular image name. Note that we still will fetch image signatures for _all_ other image names.
			if err == nil {
				break
			}

			// We skip logging unauthorized errors. Each matching image integration may either provide no credentials or
			// different credentials, which makes it expected that we receive unauthorized errors on multiple occasions.
			// The best way to handle this would be to keep a list of images which are matching but not authorized for
			// each integration, but this can be tackled at a latter improvement.
			if !errors.Is(err, errox.NotAuthorized) {
				log.Errorf("Error fetching image signatures for image %q: %v", imgName, err)
			} else {
				// Log errox.NotAuthorized erros only in debug mode, since we expect them to occur often.
				log.Debugf("Unauthorized error fetching image signatures for image %q: %v",
					imgName, err)
			}
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

	uniqueFetchedSignatures := protoutils.SliceUnique(fetchedSignatures)

	log.Debugf("Found signatures for image %q: %+v", imgName, uniqueFetchedSignatures)

	img.Signature = &storage.ImageSignature{
		Signatures: uniqueFetchedSignatures,
		Fetched:    protoconv.ConvertTimeToTimestamp(time.Now()),
	}
	return true, nil
}

func (e *enricherImpl) checkRegistryForImage(image *storage.Image) error {
	if image.GetName().GetRegistry() == "" {
		return errox.InvalidArgs.CausedByf("no registry is indicated for image %q",
			image.GetName().GetFullName())
	}
	return nil
}

func (e *enricherImpl) getRegistriesForContext(ctx EnrichmentContext) ([]registryTypes.ImageRegistry, error) {
	var registries []registryTypes.ImageRegistry
	if env.DedupeImageIntegrations.BooleanSetting() {
		registries = e.integrations.RegistrySet().GetAllUnique()
	} else {
		registries = e.integrations.RegistrySet().GetAll()
	}
	if ctx.Internal {
		if !features.SourcedAutogeneratedIntegrations.Enabled() {
			return registries, nil
		}
		if ctx.Source == nil {
			return registries, nil
		}
		filterRegistriesBySource(ctx.Source, registries)
	}

	if len(registries) == 0 {
		return nil, errox.NotFound.CausedBy("no image registries are integrated: please add an image integration")
	}

	log.Debugf("Using the following registries for enrichment: [%s]", strings.Join(registryNames(registries),
		","))

	return registries, nil
}

func registryNames(registries []registryTypes.ImageRegistry) []string {
	names := make([]string, 0, len(registries))
	for _, reg := range registries {
		names = append(names, reg.Name())
	}
	return names
}

// filterRegistriesBySource will filter the registries based on the following conditions:
// 1. If the registry is autogenerated
// 2. If the integration's source matches with the EnrichmentContext.Source
// Note that this function WILL modify the input array.
func filterRegistriesBySource(requestSource *RequestSource, registries []registryTypes.ImageRegistry) {
	if !features.SourcedAutogeneratedIntegrations.Enabled() {
		return
	}

	filteredRegistries := registries[:0]
	for _, registry := range registries {
		integration := registry.Source()
		if !integration.GetAutogenerated() {
			filteredRegistries = append(filteredRegistries, registry)
			continue
		}
		source := integration.GetSource()
		if source.GetClusterId() != requestSource.ClusterID {
			continue
		}
		// Check if the integration source is the global OpenShift registry
		if openshift.GlobalPullSecretIntegration(integration) {
			filteredRegistries = append(filteredRegistries, registry)
			continue
		}
		if source.GetNamespace() != requestSource.Namespace {
			continue
		}
		if !requestSource.ImagePullSecrets.Contains(source.GetImagePullSecretName()) {
			continue
		}
		filteredRegistries = append(filteredRegistries, registry)
	}
}

func checkForMatchingImageIntegrations(registries []registryTypes.ImageRegistry, image *storage.Image) error {
	for _, name := range image.GetNames() {
		for _, registry := range registries {
			if registry.Match(name) {
				return nil
			}
		}
	}
	return errox.NotFound.CausedByf("no matching image integrations found: please add "+
		"an image integration for %q", image.GetName().GetFullName())
}

func normalizeVulnerabilities(scan *storage.ImageScan) {
	for _, c := range scan.GetComponents() {
		for _, v := range c.GetVulns() {
			v.Severity = cvss.VulnToSeverity(cvss.NewFromEmbeddedVulnerability(v))
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
		return ScanNotDone, errors.Wrapf(err, "acquiring max concurrent scan semaphore with scanner %q", scanner.Name())
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
	if i.GetScan() == nil {
		return
	}
	i.SetComponents = &storage.Image_Components{
		Components: int32(len(i.GetScan().GetComponents())),
	}

	var fixedByProvided bool
	var imageTopCVSS float32
	vulns := make(map[string]bool)
	// This enriches the incoming component.  When enriching any additional component fields,
	// be sure to update `ComponentIDV2` to ensure enriched fields like `SetTopCVSS` are not
	// included in the hash calculation
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
