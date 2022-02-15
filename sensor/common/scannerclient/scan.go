package scannerclient

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	// ErrNoLocalScanner indicates there is no Secured Cluster-local Scanner.
	ErrNoLocalScanner = errors.New("No local Scanner integrated")

	log = logging.LoggerForModule()
)

// ScanImage runs the pipeline required to scan an image with a local Scanner.
// TODO(ROX-9281): add retries for rate-limiting.
func ScanImage(ctx context.Context, centralClient v1.ImageServiceClient, image *storage.ContainerImage) (*storage.Image, error) {
	scannerClient := GRPCClientSingleton()
	if scannerClient == nil {
		return nil, ErrNoLocalScanner
	}

	namespace := utils.ExtractOpenShiftProject(image.GetName())

	imgData, err := scannerClient.GetImageAnalysis(ctx, image)
	if err != nil {
		return nil, errors.Wrapf(err, "scanning image %q in namespace %q", image.GetName().GetFullName(), namespace)
	}
	if imgData.GetStatus() != scannerV1.ScanStatus_SUCCEEDED {
		return nil, errors.Wrapf(err, "scan failed for image %q in namespace %q", image.GetName().GetFullName(), namespace)
	}

	centralResp, err := centralClient.GetImageVulnerabilitiesInternal(ctx, &v1.GetImageVulnerabilitiesInternalRequest{
		ImageId:    image.GetId(),
		ImageName:  image.GetName(),
		Metadata:   imgData.Metadata,
		Components: imgData.GetComponents(),
		Notes:      imgData.GetNotes(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving image vulnerabilities for %s in namespace %q", image.GetName().GetFullName(), namespace)
	}

	return centralResp.GetImage(), nil
}
