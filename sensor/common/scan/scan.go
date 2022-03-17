package scan

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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

// EnrichLocalImage will enrich a cluster-local image with scan results from local scanner as well as signatures
// from the cluster-local registry. Afterwards, missing enriched data such as signature verification results and image
// vulnerabilities will be fetched from central, returning the fully enriched image.
// It will return any errors that may occur during scanning, fetching signatures or during reaching out to central.
// nolint:revive
func EnrichLocalImage(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage) (*storage.Image, error) {
	// 1. Check if Central already knows about this image.
	// If Central already knows about it, then return its results.
	img, err := centralClient.GetImage(ctx, &v1.GetImageRequest{
		Id:               ci.GetId(),
		StripDescription: true,
	})
	if err == nil {
		return img, nil
	}

	// If we received an error, we will try and enrich data locally.

	imgName := ci.GetName()

	// Find the associated registry of the image.
	matchingRegistry, err := registry.Singleton().GetRegistryForImage(ci.GetName())
	if err != nil {
		return nil, errors.Wrapf(err, "determining image registry for image %q", imgName)
	}

	log.Debugf("Received matching registry for image %q: %q", imgName, matchingRegistry.Name())

	image := types.ToImage(ci)
	// Retrieve the image's metadata.
	metadata, err := matchingRegistry.Metadata(img)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching image metadata for image %q", imgName)
	}

	log.Debugf("Received metadata for image %q: %v", imgName, metadata)

	// Retrieve the image ID with best-effort from image and metadata.
	imgID := utils.GetSHAFromIDAndMetadata(img.GetId(), metadata)

	// Scan the image via local scanner.
	scannerResp, err := scanImage(ctx, image, matchingRegistry)
	if err != nil {
		return nil, errors.Wrapf(err, "scanning image %q locally", imgName)
	}

	// Fetch signatures from cluster-local registry.
	var sigs []*storage.Signature
	if features.ImageSignatureVerification.Enabled() {
		sigs, err = signatures.FetchImageSignaturesFromImage(ctx, signatures.NewSignatureFetcher(), image,
			matchingRegistry)
		if err != nil {
			return nil, errors.Wrapf(err, "fetching signature for image %q from registry %q",
				imgName, matchingRegistry.Name())
		}
	}

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
		return nil, errors.Wrapf(err, "enriching image %q via central", imgName)
	}

	return centralResp.GetImage(), nil
}

// scanImage will scan the given image and return its components.
// It will return ErrNoLocalScanner if no local scanner is available. It will return any errors that occurred during
// receiving scan results from local scanner or if the scan status was non-successful.
func scanImage(ctx context.Context, image *storage.Image,
	registry registryTypes.Registry) (*scannerV1.GetImageComponentsResponse, error) {
	// Check if there is a local Scanner.
	// No need to continue if there is no local Scanner.
	scannerClient := scannerclient.GRPCClientSingleton()
	if scannerClient == nil {
		return nil, ErrNoLocalScanner
	}

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
