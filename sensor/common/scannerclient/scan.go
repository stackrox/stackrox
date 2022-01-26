package scannerclient

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	ErrNoLocalScanner = errors.New("No local Scanner integrated")
)

// ScanImage runs the pipeline required to scan an image with a local Scanner.
// TODO: add rate-limiting?
func ScanImage(ctx context.Context, centralClient v1.ImageServiceClient, image *storage.ContainerImage) (*storage.Image, error) {
	scannerClient := GRPCClientSingleton()
	if scannerClient == nil {
		return nil, ErrNoLocalScanner
	}

	scannerResp, err := scannerClient.GetImageAnalysis(ctx, image)
	if err != nil {
		return nil, errors.Wrap(err, "scanning image")
	}
	// If the scan did not succeed, then ignore the results.
	if scannerResp.GetStatus() != scannerV1.ScanStatus_SUCCEEDED {
		return nil, nil
	}

	centralResp, err := centralClient.GetImageVulnerabilitiesInternal(ctx, &v1.GetImageVulnerabilitiesInternalRequest{
		ImageId:    image.GetId(),
		ImageName:  image.GetName(),
		Components: scannerResp.GetComponents(),
		Notes:      scannerResp.GetNotes(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving image vulnerabilities")
	}

	return centralResp.GetImage(), nil
}
