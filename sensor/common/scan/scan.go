package scan

import (
	"context"
	"errors"
	"fmt"
	"time"

	pkgErrors "github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/docker"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/sensor/common/scannerclient"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
)

const (
	defaultMaxSemaphoreWaitTime = 5 * time.Second
)

var (
	// ErrNoLocalScanner indicates there is no Secured Cluster local Scanner connection.
	// This happens if it's not desired or if there is a connection error.
	ErrNoLocalScanner = errors.New("no local Scanner connection")

	// ErrTooManyParallelScans indicates there are too many scans in progress and wait time
	// has been exceeded.
	ErrTooManyParallelScans = errors.New("too many parallel scans to local scanner")

	// ErrEnrichNotStarted will be wrapped by other more specific errors. It is used to determine
	// if the enrichment was never started and will be no messages sent to Central.
	ErrEnrichNotStarted = errors.New("enrich was not started")

	log = logging.LoggerForModule()
)

// LocalScan wraps the functions required for enriching local images. This allows us to inject different values for testing purposes.
type LocalScan struct {
	// NOTE: If you change these, make sure to also change the respective values within the tests.
	scanImg                           func(context.Context, *storage.Image, registryTypes.ImageRegistry, scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error)
	fetchSignaturesWithRetry          func(context.Context, signatures.SignatureFetcher, *storage.Image, string, registryTypes.Registry) ([]*storage.Signature, error)
	scannerClientSingleton            func(bool) scannerclient.Client
	getRegistryForImageInNamespace    func(*storage.ImageName, string) (registryTypes.ImageRegistry, error)
	getGlobalRegistryForImage         func(*storage.ImageName) (registryTypes.ImageRegistry, error)
	createNoAuthImageRegistry         func(context.Context, *storage.ImageName, registries.Factory) (registryTypes.ImageRegistry, error)
	getMatchingCentralRegIntegrations func(*storage.ImageName) []registryTypes.ImageRegistry

	// scanSemaphore limits the number of active scans.
	scanSemaphore        *semaphore.Weighted
	maxSemaphoreWaitTime time.Duration

	regFactory registries.Factory
}

type registryStore interface {
	GetRegistryForImageInNamespace(*storage.ImageName, string) (registryTypes.ImageRegistry, error)
	GetGlobalRegistryForImage(*storage.ImageName) (registryTypes.ImageRegistry, error)
	GetMatchingCentralRegistryIntegrations(*storage.ImageName) []registryTypes.ImageRegistry
}

// LocalScanCentralClient interface to central's client
type LocalScanCentralClient interface {
	EnrichLocalImageInternal(context.Context, *v1.EnrichLocalImageInternalRequest, ...grpc.CallOption) (*v1.ScanImageInternalResponse, error)
}

// NewLocalScan initializes a LocalScan struct
func NewLocalScan(registryStore registryStore) *LocalScan {
	regFactory := registries.NewFactory(registries.FactoryOptions{
		CreatorFuncs: []registries.CreatorWrapper{
			docker.CreatorWithoutRepoList,
		},
	})
	return &LocalScan{
		scanImg:                           scanImage,
		fetchSignaturesWithRetry:          signatures.FetchImageSignaturesWithRetries,
		scannerClientSingleton:            scannerclient.GRPCClientSingleton,
		getRegistryForImageInNamespace:    registryStore.GetRegistryForImageInNamespace,
		getGlobalRegistryForImage:         registryStore.GetGlobalRegistryForImage,
		scanSemaphore:                     semaphore.NewWeighted(int64(env.MaxParallelImageScanInternal.IntegerSetting())),
		maxSemaphoreWaitTime:              defaultMaxSemaphoreWaitTime,
		createNoAuthImageRegistry:         createNoAuthImageRegistry,
		getMatchingCentralRegIntegrations: registryStore.GetMatchingCentralRegistryIntegrations,
		regFactory:                        regFactory,
	}
}

// EnrichLocalImageInNamespace invokes enrichLocalImageFromRegistry with a slice of credentials from the registryStore based on namespace as well as
// the OCP global pull secret.
//
// If no registry credentials are found an empty registry slice is passed to enrichLocalImageFromRegistry for enriching with 'no auth'.
func (s *LocalScan) EnrichLocalImageInNamespace(ctx context.Context, centralClient LocalScanCentralClient, ci *storage.ContainerImage, namespace string, requestID string, force bool, usingScannerV4 bool) (*storage.Image, error) {
	imgName := ci.GetName()

	regs := s.getRegistries(namespace, imgName)

	log.Debugf("Attempting image enrich for %q in namespace %q with %v regs", ci.GetName().GetFullName(), namespace, len(regs))

	return s.enrichLocalImageFromRegistry(ctx, centralClient, ci, regs, requestID, force, usingScannerV4)
}

func (s *LocalScan) getRegistries(namespace string, imgName *storage.ImageName) []registryTypes.ImageRegistry {
	var regs []registryTypes.ImageRegistry

	// Add registries from Central's image integrations.
	centralIntegrations := s.getMatchingCentralRegIntegrations(imgName)
	if len(centralIntegrations) > 0 {
		regs = append(regs, centralIntegrations...)
	}

	// Add registries from k8s pull secrets.
	if namespace != "" {
		// If namespace provided pull appropriate registry.
		// An err indicates no registry was found, only append if was no err
		if reg, err := s.getRegistryForImageInNamespace(imgName, namespace); err == nil {
			regs = append(regs, reg)
		}
	}

	// Add global pull secret registry
	// An err indicates no registry was found, only append if was no err
	if reg, err := s.getGlobalRegistryForImage(imgName); err == nil {
		regs = append(regs, reg)
	}

	return regs
}

// enrichLocalImageFromRegistry will enrich an image with scan results from local scanner as well as signatures
// from the local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image. A request is always sent
// to central even if errors occur pulling metadata, scanning, or fetching signatures so that the error may be
// recorded.
//
// Will use the first registry from registries that succeeds in pulling metadata, or if registries is empty will
// assume no auth is required.
//
// Will return any errors that may occur during scanning, fetching signatures or during reaching out to Central.
func (s *LocalScan) enrichLocalImageFromRegistry(ctx context.Context, centralClient LocalScanCentralClient, ci *storage.ContainerImage, registries []registryTypes.ImageRegistry, requestID string, force bool, usingScannerV4 bool) (*storage.Image, error) {
	if ci == nil {
		return nil, pkgErrors.Wrap(ErrEnrichNotStarted, "missing image, nothing to enrich")
	}

	// Check if there is a local Scanner.
	// No need to continue if there is no local Scanner.
	if s.scannerClientSingleton(usingScannerV4) == nil {
		return nil, errors.Join(ErrNoLocalScanner, ErrEnrichNotStarted)
	}

	// throttle the # of active scans.
	if err := s.scanSemaphore.Acquire(concurrency.AsContext(concurrency.Timeout(s.maxSemaphoreWaitTime)), 1); err != nil {
		return nil, errors.Join(ErrTooManyParallelScans, ErrEnrichNotStarted)
	}
	defer s.scanSemaphore.Release(1)

	log.Debugf("Enriching image locally %q numRegs %v", ci.GetName().GetFullName(), len(registries))

	if len(registries) == 0 {
		// no registries provided, try with no auth.
		reg, err := s.createNoAuthImageRegistry(ctx, ci.GetName(), s.regFactory)
		if err != nil {
			return nil, errors.Join(pkgErrors.Wrapf(err, "unable to create no auth registry for %q", ci.GetName()), ErrEnrichNotStarted)
		}
		registries = append(registries, reg)
	}

	errorList := errorhelpers.NewErrorList("image enrichment")

	image := types.ToImage(ci)
	image.Notes = make([]storage.Image_Note, 0)

	// Enrich image with metadata from one of registries.
	reg := s.enrichImageWithMetadata(errorList, registries, image)

	// Perform partial scan (image analysis / identify components) via local scanner.
	scannerResp := s.fetchImageAnalysis(ctx, errorList, reg, image, usingScannerV4)

	// Fetch signatures associated with image from registry.
	sigs := s.fetchSignatures(ctx, errorList, reg, image)

	// Send local enriched data to central to receive a fully enrich image. This includes image vulnerabilities and
	// signature verification results.
	centralResp, err := centralClient.EnrichLocalImageInternal(ctx, &v1.EnrichLocalImageInternalRequest{
		ImageId:        image.GetId(),
		ImageName:      image.GetName(),
		Metadata:       image.GetMetadata(),
		Components:     scannerResp.GetComponents(),
		Notes:          scannerResp.GetNotes(),
		ImageSignature: &storage.ImageSignature{Signatures: sigs},
		ImageNotes:     image.GetNotes(),
		Error:          errorList.String(),
		RequestId:      requestID,
		Force:          force,
	})
	if err != nil {
		log.Debugf("Unable to enrich image %q: %v", image.GetName().GetFullName(), err)
		return nil, pkgErrors.Wrapf(err, "enriching image %q via central", image.GetName())
	}

	if errorList.Empty() {
		log.Debugf("Retrieved image enrichment results for %q with id %q", image.GetName().GetFullName(), image.GetId())
	}

	return centralResp.GetImage(), errorList.ToError()
}

// enrichImageWithMetadata will loop through registries returning the first that succeeds in enriching image with metadata.
// If none succeed adds a note to the image and errors to errorList.
func (s *LocalScan) enrichImageWithMetadata(errorList *errorhelpers.ErrorList, registries []registryTypes.ImageRegistry, image *storage.Image) registryTypes.ImageRegistry {
	var errs []error
	for _, reg := range registries {
		metadata, err := reg.Metadata(image)
		if err != nil {
			log.Debugf("Failed fetching metadata for image %q with id %q: %v", image.GetName().GetFullName(), image.GetId(), err)
			errs = append(errs, err)
			continue
		}
		metadata.DataSource = &storage.DataSource{}

		// Ensure the metadata is set on the image we pass to i.e. fetching signatures. If no V2 digest is available for the
		// image, the signature will not be attempted to be fetched.
		// We don't need to do anything on central side, as there the image will correctly have the metadata assigned.
		image.Metadata = metadata
		log.Infof("Received metadata for image %q with id %q using reg %q", image.GetName().GetFullName(), image.GetId(), reg.Name())
		log.Debugf("Metadata for image %q with id %q: %v", image.GetName().GetFullName(), image.GetId(), metadata)
		return reg
	}

	errorList.AddErrors(errs...)
	image.Notes = append(image.Notes, storage.Image_MISSING_METADATA)
	return nil
}

// fetchImageAnalysis analyzes an image via the local scanner. Does nothing if errorList contains errors.
func (s *LocalScan) fetchImageAnalysis(ctx context.Context, errorList *errorhelpers.ErrorList, registry registryTypes.ImageRegistry, image *storage.Image, usingScannerV4 bool) *scannerV1.GetImageComponentsResponse {
	if !errorList.Empty() {
		// do nothing if errors previously encountered.
		return nil
	}

	// Scan the image via local scanner.
	scannerResp, err := s.scanImg(ctx, image, registry, s.scannerClientSingleton(usingScannerV4))
	if err != nil {
		log.Debugf("Scan for image %q with id %v failed: %v", image.GetName().GetFullName(), image.GetId(), err)
		image.Notes = append(image.Notes, storage.Image_MISSING_SCAN_DATA)
		errorList.AddError(pkgErrors.Wrapf(err, "scanning image %q locally", image.GetName()))
		return nil
	}

	return scannerResp
}

// fetchSignatures fetches signatures from the registry for an image. Does nothing if errorList contains errors.
func (s *LocalScan) fetchSignatures(ctx context.Context, errorList *errorhelpers.ErrorList, registry registryTypes.ImageRegistry, image *storage.Image) []*storage.Signature {
	if !errorList.Empty() {
		// do nothing if errors previously encountered.
		return nil
	}

	// Fetch signatures from cluster-local registry.
	sigs, err := s.fetchSignaturesWithRetry(ctx, signatures.NewSignatureFetcher(), image, image.GetName().GetFullName(), registry)
	if err != nil {
		log.Debugf("Failed fetching signatures for image %q: %v", image.GetName().GetFullName(), err)
		image.Notes = append(image.Notes, storage.Image_MISSING_SIGNATURE)
		errorList.AddError(pkgErrors.Wrapf(err, "fetching signature for image %q from registry %q", image.GetName(), registry.Name()))
		return nil
	}

	return sigs
}

// scanImage will scan the given image and return its components.
func scanImage(ctx context.Context, image *storage.Image,
	registry registryTypes.ImageRegistry, scannerClient scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error) {
	// Get the image analysis from the local Scanner.
	scanResp, err := scannerClient.GetImageAnalysis(ctx, image, registry.Config())
	if err != nil {
		return nil, err
	}
	// Return an error indicating a non-successful scan result.
	if scanResp.GetStatus() != scannerV1.ScanStatus_SUCCEEDED {
		return nil, fmt.Errorf("scan failed with status %q", scanResp.GetStatus().String())
	}

	return scanResp, nil
}

// createNoAuthImageRegistry creates an image registry that has no user/pass.
func createNoAuthImageRegistry(ctx context.Context, imgName *storage.ImageName, regFactory registries.Factory) (registryTypes.ImageRegistry, error) {
	registry := imgName.GetRegistry()
	if registry == "" {
		return nil, errors.New("no image registry provided, nothing to do")
	}

	secure, err := tlscheck.CheckTLS(ctx, registry)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	ii := &storage.ImageIntegration{
		Id:         registry,
		Name:       fmt.Sprintf("NoAuth/reg:%v", registry),
		Type:       docker.GenericDockerRegistryType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Insecure: !secure,
			},
		},
	}

	return regFactory.CreateRegistry(ii)
}
