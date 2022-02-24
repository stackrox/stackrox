package scan

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
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

// ScanImage runs the pipeline required to scan an image with a local Scanner.
// TODO(ROX-9281): add retries for rate-limiting.
//nolint:revive
func ScanImage(ctx context.Context, centralClient v1.ImageServiceClient, ci *storage.ContainerImage) (*storage.Image, error) {
	// 1. Check if Central already knows about this image.
	// If Central already knows about it, then return its results.
	img, err := centralClient.GetImage(ctx, &v1.GetImageRequest{
		Id:               ci.GetId(),
		StripDescription: true,
	})
	if err == nil {
		return img, nil
	}

	// The image either does not exist in Central yet or there was some other error when reaching out.
	// Attempt to scan locally.

	// 2. Check if there is a local Scanner.
	// No need to continue if there is no local Scanner.
	scannerClient := scannerclient.GRPCClientSingleton()
	if scannerClient == nil {
		return nil, ErrNoLocalScanner
	}

	// 3. Find the registry in which this image lives.
	reg, err := registry.Singleton().GetRegistryForImage(ci.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "determining image registry")
	}

	name := ci.GetName().GetFullName()
	image := types.ToImage(ci)

	// 4. Retrieve the metadata for the image from the registry.
	metadata, err := reg.Metadata(image)
	if err != nil {
		log.Debugf("Failed to get metadata for image %s: %v", name, err)
		return nil, errors.Wrap(err, "getting image metadata")
	}
	log.Debugf("Retrieved metadata for image %s: %v", name, metadata)

	// 5. Get the image analysis from the local Scanner.
	scanResp, err := scannerClient.GetImageAnalysis(ctx, image, reg.Config())
	if err != nil {
		return nil, errors.Wrapf(err, "scanning image %s", name)
	}
	if scanResp.GetStatus() != scannerV1.ScanStatus_SUCCEEDED {
		return nil, errors.Wrapf(err, "scan failed for image %s", name)
	}

	// 6. Get the image's vulnerabilities from Central.
	centralResp, err := centralClient.GetImageVulnerabilitiesInternal(ctx, &v1.GetImageVulnerabilitiesInternalRequest{
		ImageId:    image.GetId(),
		ImageName:  image.GetName(),
		Metadata:   metadata,
		Components: scanResp.GetComponents(),
		Notes:      scanResp.GetNotes(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving image vulnerabilities for %s", name)
	}

	// 7. Return the completely scanned image.
	return centralResp.GetImage(), nil
}
