package enricher

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EnrichImageByName takes an image name, and returns the corresponding
// enriched image.
// It returns a status.Error.
func EnrichImageByName(enricher ImageEnricher, enrichmentCtx EnrichmentContext, name string) (*storage.Image, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "image name must be specified")
	}
	containerImage, err := utils.GenerateImageFromString(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	img := types.ToImage(containerImage)

	enrichmentResult, err := enricher.EnrichImage(enrichmentCtx, img)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !enrichmentResult.ImageUpdated || (enrichmentResult.ScanResult != ScanSucceeded) {
		return nil, status.Error(codes.Internal, "scan could not be completed. Please check that an applicable registry and scanner is integrated")
	}

	return img, nil
}
