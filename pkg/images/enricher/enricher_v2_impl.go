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
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images"
	"github.com/stackrox/rox/pkg/images/cache"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protoutils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/time/rate"
)

var _ ImageEnricherV2 = (*enricherV2Impl)(nil)

type enricherV2Impl struct {
	cvesSuppressor CVESuppressor
	integrations   integration.Set

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

	imageGetter ImageGetterV2

	asyncRateLimiter *rate.Limiter

	metrics metrics

	scanDelegator delegatedregistry.Delegator
}

// EnrichWithVulnerabilities enriches the given image with vulnerabilities.
func (e *enricherV2Impl) EnrichWithVulnerabilities(imageV2 *storage.ImageV2, components *scannerTypes.ScanComponents, notes []scannerV1.Note) (EnrichmentResult, error) {
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
				log.Debugf("Skipping scanner %q with type %q, components are meant for scanner type %q for image: %q", scanner.Name(), scanner.Type(), components.ScannerType(), imageV2.GetName().GetFullName())
				continue
			}

			res, err := e.enrichWithVulnerabilities(scanner.Name(), imageScanner.DataSource(), vulnScanner, imageV2, components, notes)
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

func (e *enricherV2Impl) enrichWithVulnerabilities(scannerName string, dataSource *storage.DataSource, scanner scannerTypes.ImageVulnerabilityGetter,
	imageV2 *storage.ImageV2, components *scannerTypes.ScanComponents, notes []scannerV1.Note) (ScanResult, error) {
	scanStartTime := time.Now()
	image := utils.ConvertToV1(imageV2)
	scan, err := scanner.GetVulnerabilities(image, components, notes)
	e.metrics.SetImageVulnerabilityRetrievalTime(scanStartTime, scannerName, err)
	if err != nil || scan == nil {
		return ScanNotDone, err
	}

	enrichImageV2(imageV2, scan, dataSource)

	return ScanSucceeded, nil
}

func (e *enricherV2Impl) EnrichWithSignatureVerificationData(ctx context.Context, imageV2 *storage.ImageV2) (EnrichmentResult, error) {
	updated, err := e.enrichWithSignatureVerificationData(ctx, EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly}, imageV2)

	return EnrichmentResult{
		ImageUpdated: updated,
	}, err
}

// delegateEnrichImage returns true if enrichment for this image should be delegated (enriched via Sensor). If true
// and no error image was enriched successfully.
func (e *enricherV2Impl) delegateEnrichImage(ctx context.Context, enrichCtx EnrichmentContext, imageV2 *storage.ImageV2) (bool, error) {
	if !enrichCtx.Delegable {
		// Request should not be delegated.
		return false, nil
	}

	var shouldDelegate bool
	var err error
	clusterID := enrichCtx.ClusterID
	if clusterID == "" {
		clusterID, shouldDelegate, err = e.scanDelegator.GetDelegateClusterID(ctx, imageV2.GetName())
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
	existingImg, exists := e.fetchFromDatabase(ctx, imageV2, enrichCtx.FetchOpt)
	if exists && cachedImageV2IsValid(existingImg) {
		updated := e.updateImageWithExistingImage(imageV2, existingImg, enrichCtx.FetchOpt)
		if updated {
			e.cvesSuppressor.EnrichImageV2WithSuppressedCVEs(imageV2)
			// Errors for signature verification will be logged, so we can safely ignore them for the time being.
			_, _ = e.enrichWithSignatureVerificationData(ctx, enrichCtx, imageV2)

			log.Debugf("Delegated enrichment returning cached image for %q", imageV2.GetName().GetFullName())
			return true, nil
		}
	}

	// Send image to secured cluster for enrichment.
	force := enrichCtx.FetchOpt.forceRefetchCachedValues() || enrichCtx.FetchOpt == UseImageNamesRefetchCachedValues
	scannedImage, err := e.scanDelegator.DelegateScanImageV2(ctx, imageV2.GetName(), clusterID, enrichCtx.Namespace, force)
	if err != nil {
		return true, err
	}

	// Copy the fields from scannedImage into image, EnrichImage expecting modification in place
	imageV2.Reset()
	protocompat.Merge(imageV2, scannedImage)

	e.cvesSuppressor.EnrichImageV2WithSuppressedCVEs(imageV2)
	return true, nil
}

func cachedImageV2IsValid(cachedImageV2 *storage.ImageV2) bool {
	if cachedImageV2 == nil {
		return false
	}

	if metadataIsOutOfDate(cachedImageV2.GetMetadata()) {
		return false
	}

	if cachedImageV2.GetScan() == nil {
		return false
	}

	return true
}

func (e *enricherV2Impl) updateImageWithExistingImage(imageV2 *storage.ImageV2, existingImageV2 *storage.ImageV2, option FetchOption) bool {
	if option == IgnoreExistingImages {
		return false
	}

	// Prefer metadata from image, it is more likely to be up to date compared to existing image.
	if imageV2.GetMetadata() == nil {
		imageV2.SetMetadata(existingImageV2.GetMetadata())
	}
	imageV2.SetNotes(existingImageV2.GetNotes())

	e.useExistingSignature(imageV2, existingImageV2, option)
	e.useExistingSignatureVerificationData(imageV2, existingImageV2, option)
	return e.useExistingScan(imageV2, existingImageV2, option)
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherV2Impl) EnrichImage(ctx context.Context, enrichContext EnrichmentContext, imageV2 *storage.ImageV2) (EnrichmentResult, error) {
	shouldDelegate, err := e.delegateEnrichImage(ctx, enrichContext, imageV2)
	var delegateErr error
	if shouldDelegate {
		if err == nil {
			return EnrichmentResult{ImageUpdated: true, ScanResult: ScanSucceeded}, nil
		}
		if errors.Is(err, delegatedregistry.ErrNoClusterSpecified) {
			// Log the warning and try to keep enriching
			log.Warnf("Skipping delegation for %q (ID %q): %v, enriching via Central", imageV2.GetName().GetFullName(), imageV2.GetId(), err)
			delegateErr = errors.New("no cluster specified for delegated scanning and Central scan attempt failed")
		} else {
			// This enrichment should have been delegated, short circuit.
			return EnrichmentResult{ImageUpdated: false, ScanResult: ScanNotDone}, err
		}
	} else if err != nil {
		log.Warnf("Error attempting to delegate: %v", err)
	}

	errorList := errorhelpers.NewErrorList("image enrichment")
	imageNoteSet := make(map[storage.ImageV2_Note]struct{}, len(imageV2.GetNotes()))
	for _, note := range imageV2.GetNotes() {
		imageNoteSet[note] = struct{}{}
	}

	// Ensure we set the correct image notes when returning, also during short-circuiting.
	defer setImageV2Notes(imageV2, imageNoteSet)

	// Signals whether any updates to the image were made throughout the enrichment flow.
	var updated bool

	didUpdateMetadata, err := e.enrichWithMetadata(ctx, enrichContext, imageV2)
	if imageV2.GetMetadata() == nil {
		imageNoteSet[storage.ImageV2_MISSING_METADATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.ImageV2_MISSING_METADATA)
	}

	// Short-circuit if image metadata could not be retrieved. This indicates that connection or authentication to the
	// registry could not be made. Instead of trying to scan the image / fetch signatures for it, we shall short-circuit
	// here.
	if err != nil {
		errorList.AddErrors(err, delegateErr)
		return EnrichmentResult{ImageUpdated: didUpdateMetadata, ScanResult: ScanNotDone}, errorList.ToError()
	}

	updated = updated || didUpdateMetadata

	// Update the image with existing values depending on the FetchOption provided or whether any are available.
	// This makes sure that we fetch any existing image only once from database.
	useExistingScanIfPossible := e.updateImageFromDatabase(ctx, imageV2, enrichContext.FetchOpt)

	scanResult, err := e.enrichWithScan(ctx, enrichContext, imageV2, useExistingScanIfPossible)
	errorList.AddError(err)
	if scanResult == ScanNotDone && imageV2.GetScan() == nil {
		imageNoteSet[storage.ImageV2_MISSING_SCAN_DATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.ImageV2_MISSING_SCAN_DATA)
	}
	updated = updated || scanResult != ScanNotDone

	didUpdateSignature, err := e.enrichWithSignature(ctx, enrichContext, imageV2)
	errorList.AddError(err)
	if len(imageV2.GetSignature().GetSignatures()) == 0 {
		imageNoteSet[storage.ImageV2_MISSING_SIGNATURE] = struct{}{}
	} else {
		delete(imageNoteSet, storage.ImageV2_MISSING_SIGNATURE)
	}
	updated = updated || didUpdateSignature

	didUpdateSigVerificationData, err := e.enrichWithSignatureVerificationData(ctx, enrichContext, imageV2)
	errorList.AddError(err)
	if len(imageV2.GetSignatureVerificationData().GetResults()) == 0 {
		imageNoteSet[storage.ImageV2_MISSING_SIGNATURE_VERIFICATION_DATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.ImageV2_MISSING_SIGNATURE_VERIFICATION_DATA)
	}

	updated = updated || didUpdateSigVerificationData

	e.cvesSuppressor.EnrichImageV2WithSuppressedCVEs(imageV2)

	if !errorList.Empty() {
		errorList.AddError(delegateErr)
	}

	return EnrichmentResult{
		ImageUpdated: updated,
		ScanResult:   scanResult,
	}, errorList.ToError()
}

func setImageV2Notes(imageV2 *storage.ImageV2, imageNoteSet map[storage.ImageV2_Note]struct{}) {
	imageV2.SetNotes(imageV2.GetNotes()[:0])
	notes := make([]storage.ImageV2_Note, 0, len(imageNoteSet))
	for note := range imageNoteSet {
		notes = append(notes, note)
	}
	sort.SliceStable(notes, func(i, j int) bool {
		return notes[i] < notes[j]
	})
	imageV2.SetNotes(notes)
}

// updateImageFromDatabase will update the values of the given image from an existing image within the database
// depending on whether the values exist and the given FetchOption allows using existing values.
// It will return a bool indicating whether existing values from database will be used for the signature.
func (e *enricherV2Impl) updateImageFromDatabase(ctx context.Context, imageV2 *storage.ImageV2, option FetchOption) bool {
	if option == IgnoreExistingImages {
		return false
	}
	existingImg, exists := e.fetchFromDatabase(ctx, imageV2, option)
	// Short-circuit if no image exists or the FetchOption specifies to not use existing values.
	if !exists {
		return false
	}

	return e.updateImageWithExistingImage(imageV2, existingImg, option)
}

// metadataIsValid returns true of the image's metadata is valid and doesn't need to be refreshed,
// false otherwise.
func (e *enricherV2Impl) metadataIsValid(imageV2 *storage.ImageV2) bool {
	if metadataIsOutOfDate(imageV2.GetMetadata()) {
		return false
	}

	dataSource := imageV2.GetMetadata().GetDataSource()
	if e.integrations.RegistrySet().Get(dataSource.GetId()) == nil {
		// The integration referenced by the datasource does not exist. Metadata should be updated to
		// re-populate the datasource increasing the chances of a successful scan.
		return false
	}

	if dataSource.GetMirror() != "" {
		// If the metadata was pulled from a mirror (which can only occur if the scan was previously delegated),
		// the datasource needs to be updated to represent the correct registry.
		return false
	}

	return true
}

func (e *enricherV2Impl) enrichWithMetadata(ctx context.Context, enrichmentContext EnrichmentContext, imageV2 *storage.ImageV2) (bool, error) {
	// Attempt to short-circuit before checking registries.
	if e.metadataIsValid(imageV2) {
		return false, nil
	}

	if !enrichmentContext.FetchOpt.forceRefetchCachedValues() &&
		enrichmentContext.FetchOpt != UseImageNamesRefetchCachedValues {
		// The metadata in the cache is always up-to-date with respect to the current metadataVersion
		if metadataValue, ok := e.metadataCache.Get(getRefV2(imageV2)); ok {
			e.metrics.IncrementMetadataCacheHit()
			imageV2.SetMetadata(metadataValue.CloneVT())
			return true, nil
		}
		e.metrics.IncrementMetadataCacheMiss()
	}
	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error getting metadata for image: %s", imageV2.GetName().GetFullName()))

	if err := e.checkRegistryForImage(imageV2); err != nil {
		errorList.AddError(err)
		return false, errorList.ToError()
	}

	registries, err := e.getRegistriesForContext(enrichmentContext)
	if err != nil {
		errorList.AddError(err)
		return false, errorList.ToError()
	}

	image := utils.ConvertToV1(imageV2)
	log.Infof("Getting metadata for image %q (ID %q)", imageV2.GetName().GetFullName(), imageV2.GetId())
	for _, registry := range registries {
		updated, err := e.enrichImageWithRegistry(ctx, imageV2, image, registry)
		if err != nil {
			currentRegistryErrors := concurrency.WithLock1(&e.registryErrorsLock, func() int32 {
				currentRegistryErrors := e.errorsPerRegistry[registry] + 1
				e.errorsPerRegistry[registry] = currentRegistryErrors
				return currentRegistryErrors
			})

			if currentRegistryErrors >= consecutiveErrorThreshold { // update health
				ih := &storage.IntegrationHealth{}
				ih.SetId(registry.Source().GetId())
				ih.SetName(registry.Source().GetName())
				ih.SetType(storage.IntegrationHealth_IMAGE_INTEGRATION)
				ih.SetStatus(storage.IntegrationHealth_UNHEALTHY)
				ih.SetLastTimestamp(protocompat.TimestampNow())
				ih.SetErrorMessage(err.Error())
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(ih)
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
			ih := &storage.IntegrationHealth{}
			ih.SetId(id)
			ih.SetName(name)
			ih.SetType(storage.IntegrationHealth_IMAGE_INTEGRATION)
			ih.SetStatus(storage.IntegrationHealth_HEALTHY)
			ih.SetLastTimestamp(protocompat.TimestampNow())
			ih.SetErrorMessage("")
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(ih)
			return true, nil
		}
	}

	if !enrichmentContext.Internal && errorList.Empty() {
		errorList.AddError(errors.Errorf("no matching image registries found: please add an image integration for %s", imageV2.GetName().GetRegistry()))
	}

	return false, errorList.ToError()
}

func getRefV2(image *storage.ImageV2) string {
	if image.GetId() != "" {
		return image.GetId()
	}
	return image.GetName().GetFullName()
}

func (e *enricherV2Impl) enrichImageWithRegistry(ctx context.Context, imageV2 *storage.ImageV2, image *storage.Image,
	registry registryTypes.ImageRegistry) (bool, error) {
	if !registry.Match(imageV2.GetName()) {
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
	metadata.SetDataSource(registry.DataSource())
	if features.SourcedAutogeneratedIntegrations.Enabled() {
		metadata.SetDataSource(imageIntegrationToDataSource(registry.Source()))
	}
	metadata.SetVersion(metadataVersion)
	imageV2.SetMetadata(metadata)

	cachedMetadata := metadata.CloneVT()
	e.metadataCache.Add(getRefV2(imageV2), cachedMetadata)
	id, err := utils.GetImageV2ID(imageV2)
	if err != nil {
		return false, err
	}
	if id != "" {
		e.metadataCache.Add(id, cachedMetadata)
	}
	return true, nil
}

func (e *enricherV2Impl) fetchFromDatabase(ctx context.Context, imgV2 *storage.ImageV2, option FetchOption) (*storage.ImageV2, bool) {
	if option.forceRefetchCachedValues() {
		// When re-fetched values should be used, reset the existing values for signature and signature verification data.
		imgV2.ClearSignature()
		imgV2.ClearSignatureVerificationData()
		return imgV2, false
	}
	// See if the image exists in the DB with a scan, if it does, then use that instead of fetching
	sha := utils.GetSHAV2(imgV2)
	if sha == "" {
		return imgV2, false
	}
	id, err := utils.GetImageV2ID(imgV2)
	if err != nil {
		log.Errorf("error getting ID for image %q: %v", imgV2.GetName().GetFullName(), err)
		return imgV2, false
	}
	existingImage, exists, err := e.imageGetter(sac.WithAllAccess(ctx), id)
	if err != nil {
		log.Errorf("error fetching image %q: %v", id, err)
		return imgV2, false
	}

	// Special case: in the case we want to refetch cached values but retain the image names, we have to
	// first fetch the existing image, if it exists, merge the image names, and then return the modified
	// image. Currently, the scope of the option is to only be used by services which take in an external, "fresh"
	// image via API (e.g. by using roxctl). The option was created to not have an effect on performance for the
	// existing ForceRefetch and ForceRefetechCachedValuesOnly options and their related components,
	// i.e. the reprocessing loop.
	if option == UseImageNamesRefetchCachedValues {
		imgV2.ClearSignatureVerificationData()
		imgV2.ClearSignature()
		return imgV2, false
	}

	return existingImage, exists
}

func (e *enricherV2Impl) useExistingScan(imgV2 *storage.ImageV2, existingImgV2 *storage.ImageV2, option FetchOption) bool {
	if option == ForceRefetchScansOnly {
		return false
	}

	if existingImgV2.GetScan() != nil {
		imgV2.SetScan(existingImgV2.GetScan())
		return true
	}

	return false
}

func (e *enricherV2Impl) useExistingSignature(imgV2 *storage.ImageV2, existingImgV2 *storage.ImageV2, option FetchOption) {
	if option == ForceRefetchSignaturesOnly {
		// When forced to refetch values, disregard existing ones.
		imgV2.ClearSignature()
		return
	}

	if existingImgV2.GetSignature() != nil {
		imgV2.SetSignature(existingImgV2.GetSignature())
	}
}

func (e *enricherV2Impl) useExistingSignatureVerificationData(imgV2 *storage.ImageV2, existingImgV2 *storage.ImageV2,
	option FetchOption) {
	if option == ForceRefetchSignaturesOnly {
		// When forced to refetch values, disregard existing ones.
		imgV2.ClearSignatureVerificationData()
		return
	}

	if existingImgV2.GetSignatureVerificationData() != nil {
		imgV2.SetSignatureVerificationData(existingImgV2.GetSignatureVerificationData())
	}
}

func (e *enricherV2Impl) enrichWithScan(ctx context.Context, enrichmentContext EnrichmentContext,
	imageV2 *storage.ImageV2, useExistingScan bool) (ScanResult, error) {
	// Short-circuit if we are using existing values.
	// We need to have a distinction between existing image values set
	// from database and existing values from the received image. Existing image values set by database indicate the
	// ScanResult ScanSucceeded, whereas existing values on the image that have not been set from database indicate the
	// ScanResult ScanNotDone.
	if useExistingScan {
		return ScanSucceeded, nil
	}

	// Attempt to short-circuit before checking scanners.
	if enrichmentContext.FetchOnlyIfScanEmpty() && imageV2.GetScan() != nil {
		return ScanNotDone, nil
	}

	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return ScanNotDone, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning image: %s", imageV2.GetName().GetFullName()))
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

	image := utils.ConvertToV1(imageV2)
	log.Debugf("Scanning image %q (ID %q)", imageV2.GetName().GetFullName(), imageV2.GetId())
	for _, scanner := range scanners.GetAll() {
		scannerType := scanner.GetScanner().Type()
		// only run scan with scanner specified in scannerTypeHint
		if enrichmentContext.ScannerTypeHint != "" && scannerType != enrichmentContext.ScannerTypeHint {
			continue
		}
		result, err := e.enrichImageWithScanner(ctx, imageV2, image, scanner)
		if err != nil {
			currentScannerErrors := concurrency.WithLock1(&e.scannerErrorsLock, func() int32 {
				currentScannerErrors := e.errorsPerScanner[scanner] + 1
				e.errorsPerScanner[scanner] = currentScannerErrors
				return currentScannerErrors
			})
			if currentScannerErrors >= consecutiveErrorThreshold { // update health
				ih := &storage.IntegrationHealth{}
				ih.SetId(scanner.DataSource().GetId())
				ih.SetName(scanner.DataSource().GetName())
				ih.SetType(storage.IntegrationHealth_IMAGE_INTEGRATION)
				ih.SetStatus(storage.IntegrationHealth_UNHEALTHY)
				ih.SetLastTimestamp(protocompat.TimestampNow())
				ih.SetErrorMessage(err.Error())
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(ih)
			}
			errorList.AddError(err)

			if features.ScannerV4.Enabled() && scanner.GetScanner().Type() == scannerTypes.ScannerV4 {
				// Do not try to scan with additional scanners if Scanner V4 enabled and fails to scan an image.
				// This would result in Clairify scanners being skipped per sorting logic in `GetAll` of
				// `pkg/scanners/set_impl.go`.
				log.Debugf("Scanner V4 encountered an error scanning image %q, skipping remaining scanners", imageV2.GetName().GetFullName())
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
			ih := &storage.IntegrationHealth{}
			ih.SetId(scanner.DataSource().GetId())
			ih.SetName(scanner.DataSource().GetName())
			ih.SetType(storage.IntegrationHealth_IMAGE_INTEGRATION)
			ih.SetStatus(storage.IntegrationHealth_HEALTHY)
			ih.SetLastTimestamp(protocompat.TimestampNow())
			ih.SetErrorMessage("")
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(ih)
			return result, nil
		}
	}
	return ScanNotDone, errorList.ToError()
}

// enrichWithSignatureVerificationData enriches the image with signature verification data and returns a bool,
// indicating whether any verification data was added to the image or not.
// Based on the given FetchOption, it will try to short-circuit using values from existing images where
// possible.
func (e *enricherV2Impl) enrichWithSignatureVerificationData(ctx context.Context, enrichmentContext EnrichmentContext,
	imgV2 *storage.ImageV2) (bool, error) {
	// If no external metadata should be taken into account, we skip the verification.
	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	imgName := imgV2.GetName().GetFullName()

	// Short-circuit if no signature is available.
	if len(imgV2.GetSignature().GetSignatures()) == 0 {
		// If no signature is given but there are signature verification results on the image, make sure we delete
		// the stale signature verification results.
		if len(imgV2.GetSignatureVerificationData().GetResults()) != 0 {
			imgV2.ClearSignatureVerificationData()
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
		if len(imgV2.GetSignatureVerificationData().GetResults()) != 0 {
			imgV2.ClearSignatureVerificationData()
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
	if imgV2.GetSignatureVerificationData() != nil && enrichmentContext.FetchOpt != ForceRefetchSignaturesOnly {
		return false, nil
	}

	// Timeout is based on benchmark test result for 200 integrations with 1 config each (roughly 0.1 sec) + a grace
	// timeout on top. Currently, signature verification is done without remote RPCs, this will need to be
	// changed accordingly when RPCs are required (i.e. cosign keyless).
	verifySignatureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	res := e.signatureVerifier(verifySignatureCtx, sigIntegrations, utils.ConvertToV1(imgV2))
	if res == nil {
		return false, ctx.Err()
	}

	log.Debugf("Verification results found for image %q: %+v", imgName, res)

	isvd := &storage.ImageSignatureVerificationData{}
	isvd.SetResults(res)
	imgV2.SetSignatureVerificationData(isvd)
	return true, nil
}

// enrichWithSignature enriches the image with a signature and returns a bool, indicating whether a signature has been
// updated on the image or not.
// Based on the given FetchOption, it will try to short-circuit using values from existing images where
// possible.
func (e *enricherV2Impl) enrichWithSignature(ctx context.Context, enrichmentContext EnrichmentContext,
	imgV2 *storage.ImageV2) (bool, error) {
	// If no external metadata should be taken into account, we skip the fetching of signatures.
	if enrichmentContext.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	// Short-circuit if possible when we are using existing values.
	if imgV2.GetSignature() != nil {
		return false, nil
	}

	// Fetch signature integrations from the data store.
	sigIntegrations, err := e.signatureIntegrationGetter(sac.WithAllAccess(ctx))
	if err != nil {
		return false, errors.Wrap(err, "fetching signature integrations")
	}

	onlyRedHatSigIntegrationPresent := len(sigIntegrations) == 1 &&
		sigIntegrations[0].GetId() == signatures.DefaultRedHatSignatureIntegration.GetId()

	// Short-circuit if
	//	- no integrations are available, or
	//	- only the default Red Hat sig integration exists, and this is not a Red Hat image
	if len(sigIntegrations) == 0 || (onlyRedHatSigIntegrationPresent && !utils.IsRedHatImageV2(imgV2)) {
		description := "No signature integration available"
		if onlyRedHatSigIntegrationPresent {
			description = fmt.Sprintf("Only Red Hat signature integration available and %q is not a Red Hat image",
				imgV2.GetName().GetFullName())
		}

		log.Debugf("%s, skipping signature enrichment", description)
		// Contrary to the signature verification step we will retain existing signatures.
		return false, nil
	}

	imgFullName := imgV2.GetName().GetFullName()

	if err := e.checkRegistryForImage(imgV2); err != nil {
		return false, errors.Wrapf(err, "checking registry for image %q", imgFullName)
	}

	registries, err := e.getRegistriesForContext(enrichmentContext)
	if err != nil {
		return false, errors.Wrap(err, "getting registries for context")
	}

	if err := checkForMatchingImageIntegrationsV2(registries, imgV2); err != nil {
		// Do not return an error for internal images when no integration is found.
		if enrichmentContext.Internal {
			return false, nil
		}
		return false, errors.Wrapf(err, "checking for matching registries for image %q", imgFullName)
	}

	var fetchedSignatures []*storage.Signature
	matchingImageIntegrations := integration.GetMatchingImageIntegrations(ctx, registries, imgV2.GetName())
	if len(matchingImageIntegrations) == 0 {
		// Instead of propagating an error and stopping the image enrichment, we will instead log the occurrence
		// and skip fetching signatures for this particular image name. We know that we have at least one matching
		// image registry due to the call to checkForMatchingImageIntegrations above, so we do not want to abort
		// enriching the image completely.
		log.Infof("No matching image integration found for image name %q, hence no signatures will be "+
			"attempted to be fetched", imgV2.GetName())
	}

	img := utils.ConvertToV1(imgV2)
	for _, registry := range matchingImageIntegrations {
		// FetchImageSignaturesWithRetries will try fetching of signatures with retries.
		sigs, err := signatures.FetchImageSignaturesWithRetries(ctx, e.signatureFetcher, img, imgFullName, registry)
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
			log.Errorf("Error fetching image signatures for image %q: %v", imgFullName, err)
		} else {
			// Log errox.NotAuthorized errors only in debug mode, since we expect them to occur often.
			log.Debugf("Unauthorized error fetching image signatures for image %q: %v",
				imgFullName, err)
		}
	}

	// Do not signal updates when no signatures have been fetched.
	if len(fetchedSignatures) == 0 {
		// Delete existing signatures on the image if we fetched zero.
		if len(imgV2.GetSignature().GetSignatures()) != 0 {
			log.Debugf("No signatures found but image %q had existing signatures, deleting those", imgFullName)
			imgV2.ClearSignature()
			return true, nil
		}
		log.Debugf("No signatures associated with image %q", imgFullName)
		return false, nil
	}

	uniqueFetchedSignatures := protoutils.SliceUnique(fetchedSignatures)

	log.Debugf("Found signatures for image %q: %+v", imgFullName, uniqueFetchedSignatures)

	is := &storage.ImageSignature{}
	is.SetSignatures(uniqueFetchedSignatures)
	is.SetFetched(protoconv.ConvertTimeToTimestamp(time.Now()))
	imgV2.SetSignature(is)
	return true, nil
}

func (e *enricherV2Impl) checkRegistryForImage(imageV2 *storage.ImageV2) error {
	if imageV2.GetName().GetRegistry() == "" {
		return errox.InvalidArgs.CausedByf("no registry is indicated for image %q",
			imageV2.GetName().GetFullName())
	}
	return nil
}

func (e *enricherV2Impl) getRegistriesForContext(ctx EnrichmentContext) ([]registryTypes.ImageRegistry, error) {
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

func checkForMatchingImageIntegrationsV2(registries []registryTypes.ImageRegistry, imageV2 *storage.ImageV2) error {
	for _, registry := range registries {
		if registry.Match(imageV2.GetName()) {
			return nil
		}
	}
	return errox.NotFound.CausedByf("no matching image integrations found: please add "+
		"an image integration for %q", imageV2.GetName().GetFullName())
}

func (e *enricherV2Impl) enrichImageWithScanner(ctx context.Context, imageV2 *storage.ImageV2, image *storage.Image,
	imageScanner scannerTypes.ImageScannerWithDataSource) (ScanResult, error) {
	scanner := imageScanner.GetScanner()

	if !scanner.Match(imageV2.GetName()) {
		return ScanNotDone, nil
	}

	sema := scanner.MaxConcurrentScanSemaphore()
	if err := acquireSemaphoreWithMetrics(sema, ctx); err != nil {
		return ScanNotDone, errors.Wrapf(err, "acquiring max concurrent scan semaphore with scanner %q", scanner.Name())
	}

	defer func() {
		sema.Release(1)
		images.ScanSemaphoreHoldingSize.WithLabelValues("central", "central-image-enricher", "n/a").Dec()
	}()

	scanStartTime := time.Now()
	scan, err := scanner.GetScan(image)
	e.metrics.SetScanDurationTime(scanStartTime, scanner.Name(), err)
	if err != nil {
		return ScanNotDone, errors.Wrapf(err, "scanning %q with scanner %q", imageV2.GetName().GetFullName(), scanner.Name())
	}
	if scan == nil {
		return ScanNotDone, nil
	}

	enrichImageV2(imageV2, scan, imageScanner.DataSource())

	return ScanSucceeded, nil
}

func enrichImageV2(imageV2 *storage.ImageV2, scan *storage.ImageScan, dataSource *storage.DataSource) {
	// Normalize the vulnerabilities.
	normalizeVulnerabilities(scan)

	scan.SetDataSource(dataSource)

	// Assume:
	//  scan != nil
	//  no error scanning.
	imageV2.SetScan(scan)
	utils.FillScanStatsV2(imageV2)
}
