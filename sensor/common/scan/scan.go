package scan

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/scannerclient"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	// ErrNoLocalScanner indicates there is no Secured Cluster local Scanner connection.
	// This happens if it's not desired or if there is a connection error.
	ErrNoLocalScanner = errors.New("No local Scanner connection")

	log = logging.LoggerForModule()
)

// LocalScan wraps the functions required in EnrichLocalImage. This allows us to inject different values for testing purposes
type LocalScan struct {
	// NOTE: If you change these, make sure to also change the respective values within the tests.
	scanImg                        func(context.Context, *storage.Image, registryTypes.Registry, *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error)
	fetchSignaturesWithRetry       func(context.Context, signatures.SignatureFetcher, *storage.Image, string, registryTypes.Registry) ([]*storage.Signature, error)
	scannerClientSingleton         func() *scannerclient.Client
	getMatchingRegistry            func(*storage.ImageName) (registryTypes.Registry, error)
	getRegistryForImageInNamespace func(*storage.ImageName, string) (registryTypes.Registry, error)
	upsertNoAuthRegistry           func(context.Context, string, *storage.ImageName) (registryTypes.Registry, error)
}

// NewLocalScan initializes a LocalScan struct
func NewLocalScan(registryStore *registry.Store) *LocalScan {
	return &LocalScan{
		scanImg:                        scanImage,
		fetchSignaturesWithRetry:       signatures.FetchImageSignaturesWithRetries,
		scannerClientSingleton:         scannerclient.GRPCClientSingleton,
		getMatchingRegistry:            registryStore.GetRegistryForImage,
		getRegistryForImageInNamespace: registryStore.GetRegistryForImageInNamespace,
		upsertNoAuthRegistry:           registryStore.UpsertNoAuthRegistry,
	}
}

// EnrichLocalImageFromRegistry will enrich an image with scan results from local scanner as well as signatures
// from the local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image.
//
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
func (s *LocalScan) EnrichLocalImageFromRegistry(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage, registry registryTypes.Registry) (*storage.Image, error) {
	// Check if there is a local Scanner.
	// No need to continue if there is no local Scanner.
	if s.scannerClientSingleton() == nil {
		return nil, ErrNoLocalScanner
	}

	errorList := errorhelpers.NewErrorList("image enrichment")

	image := types.ToImage(ci)
	image.Notes = make([]storage.Image_Note, 0)

	// Enrich image with metadata from registry
	s.enrichImageWithMetdata(errorList, registry, image)

	// Perform partial scan (image analysis / identify components) via local scanner
	scannerResp := s.fetchImageAnalysis(ctx, errorList, registry, image)

	// Fetch signatures associated with image from registry
	sigs := s.fetchSignatures(ctx, errorList, registry, image)

	// Send local enriched data to central to receive a fully enrich image. This includes image vulnerabilities and
	// signature verification results.
	centralResp, err := centralClient.EnrichLocalImageInternal(ctx, &v1.EnrichLocalImageInternalRequest{
		ImageId:        utils.GetSHA(image),
		ImageName:      image.GetName(),
		Metadata:       image.GetMetadata(),
		Components:     scannerResp.GetComponents(),
		Notes:          scannerResp.GetNotes(),
		ImageSignature: &storage.ImageSignature{Signatures: sigs},
		ImageNotes:     image.GetNotes(),
		Error:          errorList.String(),
	})
	if err != nil {
		log.Debugf("Unable to enrich image %q: %v", image.GetName(), err)
		return nil, errors.Wrapf(err, "enriching image %q via central", image.GetName())
	}

	if errorList.Empty() {
		log.Debugf("Retrieved image enrichment results for %q", image.GetName())
	}

	return centralResp.GetImage(), errorList.ToError()
}

func (s *LocalScan) enrichImageWithMetdata(errorList *errorhelpers.ErrorList, registry registryTypes.Registry, image *storage.Image) {
	metadata, err := registry.Metadata(image)
	if err != nil {
		log.Debugf("Failed fetching image metadata for image %q: %v", image.GetName(), err)
		image.Notes = append(image.Notes, storage.Image_MISSING_METADATA)
		errorList.AddError(errors.Wrapf(err, "fetching image metadata for image %q", image.GetName()))
		return
	}

	// Ensure the metadata is set on the image we pass to i.e. fetching signatures. If no V2 digest is available for the
	// image, the signature will not be attempted to be fetched.
	// We don't need to do anything on central side, as there the image will correctly have the metadata assigned.
	image.Metadata = metadata
	log.Debugf("Received metadata for image %q: %v", image.GetName(), metadata)
}

func (s *LocalScan) fetchImageAnalysis(ctx context.Context, errorList *errorhelpers.ErrorList, registry registryTypes.Registry, image *storage.Image) *scannerV1.GetImageComponentsResponse {
	if !errorList.Empty() {
		// do nothing if errors previously encountered
		return nil
	}

	// Scan the image via local scanner.
	scannerResp, err := s.scanImg(ctx, image, registry, s.scannerClientSingleton())
	if err != nil {
		log.Debugf("Scan for image %q failed: %v", image.GetName(), err)
		image.Notes = append(image.Notes, storage.Image_MISSING_SCAN_DATA)
		errorList.AddError(errors.Wrapf(err, "scanning image %q locally", image.GetName()))
		return nil
	}

	return scannerResp
}

func (s *LocalScan) fetchSignatures(ctx context.Context, errorList *errorhelpers.ErrorList, registry registryTypes.Registry, image *storage.Image) []*storage.Signature {
	if !errorList.Empty() {
		// do nothing if errors previously encountered
		return nil
	}

	// Fetch signatures from cluster-local registry.
	sigs, err := s.fetchSignaturesWithRetry(ctx, signatures.NewSignatureFetcher(), image, image.GetName().GetFullName(), registry)
	if err != nil {
		log.Debugf("Failed fetching signatures for image %q: %v", image.GetName(), err)
		image.Notes = append(image.Notes, storage.Image_MISSING_SIGNATURE)
		errorList.AddError(errors.Wrapf(err, "fetching signature for image %q from registry %q", image.GetName(), registry.Name()))
		return nil
	}

	return sigs
}

// EnrichLocalImage will enrich a cluster-local image with scan results from local scanner as well as signatures
// from the cluster-local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image.
//
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
//
// Registry credentials are extracted from getMatchingRegistry based on ci.Name, returns error if no credentials are found
func (s *LocalScan) EnrichLocalImage(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage) (*storage.Image, error) {
	imgName := ci.GetName().GetFullName()

	// Find the associated registry of the image.
	matchingRegistry, err := s.getMatchingRegistry(ci.GetName())
	if err != nil {
		return nil, errors.Wrapf(err, "determining image registry for image %q", imgName)
	}

	log.Debugf("Received matching registry for image %q: %q", imgName, matchingRegistry.Name())

	return s.EnrichLocalImageFromRegistry(ctx, centralClient, ci, matchingRegistry)
}

// EnrichLocalImageInNamespace will enrich a cluster-local image with scan results from local scanner as well as signatures
// from the cluster-local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image.
//
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
//
// Registry credentials are extracted from getRegistryForImageInNamespace based on namespace, if no credentials are found
// assumes no auth is needed
func (s *LocalScan) EnrichLocalImageInNamespace(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage, namespace string) (*storage.Image, error) {
	var reg registryTypes.Registry
	imgName := ci.GetName()

	reg, err := s.getRegistryForImageInNamespace(imgName, namespace)
	if err != nil {
		// no registry was found, assume this image represents a registry that does not require authentication.
		// add the registry to regStore and use it for scanning going forward
		reg, err = s.upsertNoAuthRegistry(ctx, namespace, imgName)
		if err != nil {
			return nil, err
		}
	}

	return s.EnrichLocalImageFromRegistry(ctx, centralClient, ci, reg)
}

// scanImage will scan the given image and return its components.
func scanImage(ctx context.Context, image *storage.Image,
	registry registryTypes.Registry, scannerClient *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error) {
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
