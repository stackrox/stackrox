package scan

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/images/types"
	"github.com/stackrox/stackrox/pkg/images/utils"
	"github.com/stackrox/stackrox/pkg/logging"
	registryTypes "github.com/stackrox/stackrox/pkg/registries/types"
	"github.com/stackrox/stackrox/pkg/signatures"
	"github.com/stackrox/stackrox/sensor/common/registry"
	"github.com/stackrox/stackrox/sensor/common/scannerclient"
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

// EnrichLocalImage will enrich a cluster-local image with scan results from local scanner as well as signatures
// from the cluster-local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image.
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
func EnrichLocalImage(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage) (*storage.Image, error) {
	imgName := ci.GetName().GetFullName()

	// Check if there is a local Scanner.
	// No need to continue if there is no local Scanner.
	scannerClient := scannerClientSingleton()
	if scannerClient == nil {
		return nil, ErrNoLocalScanner
	}

	// Find the associated registry of the image.
	matchingRegistry, err := getMatchingRegistry(ci.GetName())
	if err != nil {
		return nil, errors.Wrapf(err, "determining image registry for image %q", imgName)
	}

	log.Debugf("Received matching registry for image %q: %q", imgName, matchingRegistry.Name())

	image := types.ToImage(ci)
	// Retrieve the image's metadata.
	metadata, err := matchingRegistry.Metadata(image)
	if err != nil {
		log.Debugf("Failed fetching image metadata for image %q: %v", imgName, err)
		return nil, errors.Wrapf(err, "fetching image metadata for image %q", imgName)
	}

	log.Debugf("Received metadata for image %q: %v", imgName, metadata)

	// Scan the image via local scanner.
	scannerResp, err := scanImg(ctx, image, matchingRegistry, scannerClient)
	if err != nil {
		log.Debugf("Scan for image %q failed: %v", imgName, err)
		return nil, errors.Wrapf(err, "scanning image %q locally", imgName)
	}

	// Fetch signatures from cluster-local registry.
	var sigs []*storage.Signature
	if features.ImageSignatureVerification.Enabled() {
		sigs, err = fetchSignaturesWithRetry(ctx, signatures.NewSignatureFetcher(), image,
			matchingRegistry)
		if err != nil {
			log.Debugf("Failed fetching signatures for image %q: %v", imgName, err)
			return nil, errors.Wrapf(err, "fetching signature for image %q from registry %q",
				imgName, matchingRegistry.Name())
		}
	}

	// Retrieve the image ID with best-effort from image and metadata.
	imgID := utils.GetSHAFromIDAndMetadata(image.GetId(), metadata)

	// Send local enriched data to central to receive a fully enrich image. This includes image vulnerabilities and
	// signature verification results.
	centralResp, err := centralClient.EnrichLocalImageInternal(ctx, &v1.EnrichLocalImageInternalRequest{
		ImageId:        imgID,
		ImageName:      image.GetName(),
		Metadata:       metadata,
		Components:     scannerResp.GetComponents(),
		Notes:          scannerResp.GetNotes(),
		ImageSignature: &storage.ImageSignature{Signatures: sigs},
	})
	if err != nil {
		log.Debugf("Unable to enrich image %q: %v", imgName, err)
		return nil, errors.Wrapf(err, "enriching image %q via central", imgName)
	}

	log.Debugf("Retrieved image enrichment results for %q", imgName)

	return centralResp.GetImage(), nil
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
