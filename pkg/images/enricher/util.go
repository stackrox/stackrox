package enricher

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
)

// EnrichImageByName takes an image name, and returns the corresponding enriched image.
func EnrichImageByName(ctx context.Context, enricher ImageEnricher, enrichmentCtx EnrichmentContext, name string) (*storage.Image, error) {
	if name == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "image name must be specified")
	}
	containerImage, err := utils.GenerateImageFromString(name)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	img := types.ToImage(containerImage)

	enrichmentResult, err := enricher.EnrichImage(ctx, enrichmentCtx, img)
	if err != nil {
		return nil, err
	}

	if !enrichmentResult.ImageUpdated || (enrichmentResult.ScanResult != ScanSucceeded) {
		return nil, errors.New("scan could not be completed. Please check that an applicable registry and scanner is integrated")
	}

	return img, nil
}
