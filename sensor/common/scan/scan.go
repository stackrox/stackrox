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

	// Used for testing purposes only to not require setting up registry / scanner.
	// NOTE: If you change these, make sure to also change the respective values within the tests.
	scanImg                  = scanImage
	fetchSignaturesWithRetry = signatures.FetchImageSignaturesWithRetries
	getMatchingRegistry      = registry.Singleton().GetRegistryForImage
	scannerClientSingleton   = scannerclient.GRPCClientSingleton
)

// EnrichLocalImageFromRegistry will enrich an image with scan results from local scanner as well as signatures
// from the local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image.
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
func EnrichLocalImageFromRegistry(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage, registry registryTypes.Registry) (*storage.Image, error) {
	// Check if there is a local Scanner.
	// No need to continue if there is no local Scanner.
	scannerClient := scannerClientSingleton()
	if scannerClient == nil {
		return nil, ErrNoLocalScanner
	}

	errorList := errorhelpers.NewErrorList("image enrichment")

	image := types.ToImage(ci)
	image.Notes = make([]storage.Image_Note, 0)

	// Enrich image with metadata from registry
	enrichImageWithMetdata(errorList, registry, image)

	// Perform image analysis (identify components) via local scanner
	scannerResp := fetchImageAnalysis(ctx, errorList, registry, image)

	// Fetch signatures associated with image from registry
	sigs := fetchSignatures(ctx, errorList, registry, image)

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

func enrichImageWithMetdata(errorList *errorhelpers.ErrorList, registry registryTypes.Registry, image *storage.Image) {
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

func fetchImageAnalysis(ctx context.Context, errorList *errorhelpers.ErrorList, registry registryTypes.Registry, image *storage.Image) *scannerV1.GetImageComponentsResponse {
	if !errorList.Empty() {
		// do nothing if errors previously encountered
		return nil
	}

	// Scan the image via local scanner.
	scannerclient := scannerClientSingleton()
	scannerResp, err := scanImg(ctx, image, registry, scannerclient)
	if err != nil {
		log.Debugf("Scan for image %q failed: %v", image.GetName(), err)
		image.Notes = append(image.Notes, storage.Image_MISSING_SCAN_DATA)
		errorList.AddError(errors.Wrapf(err, "scanning image %q locally", image.GetName()))
		return nil
	}

	return scannerResp
}

func fetchSignatures(ctx context.Context, errorList *errorhelpers.ErrorList, registry registryTypes.Registry, image *storage.Image) []*storage.Signature {
	if !errorList.Empty() {
		// do nothing if errors previously encountered
		return nil
	}

	// Fetch signatures from cluster-local registry.
	sigs, err := fetchSignaturesWithRetry(ctx, signatures.NewSignatureFetcher(), image, image.GetName().GetFullName(), registry)
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
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
func EnrichLocalImage(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage) (*storage.Image, error) {
	imgName := ci.GetName().GetFullName()

	// Find the associated registry of the image.
	matchingRegistry, err := getMatchingRegistry(ci.GetName())
	if err != nil {
		return nil, errors.Wrapf(err, "determining image registry for image %q", imgName)
	}

	log.Debugf("Received matching registry for image %q: %q", imgName, matchingRegistry.Name())

	return EnrichLocalImageFromRegistry(ctx, centralClient, ci, matchingRegistry)
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
