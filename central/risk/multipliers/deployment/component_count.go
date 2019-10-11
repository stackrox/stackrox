package deployment

import (
	"context"
	"fmt"

	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// componentCountMultiplier is a scorer for the number of components in an image.
type componentCountMultiplier struct {
	imageScorer imageMultiplier.Multiplier
}

// NewComponentCount provides a multiplier that scores the data based on the the number of components in images.
func NewComponentCount() Multiplier {
	return &componentCountMultiplier{
		imageScorer: imageMultiplier.NewComponentCount(),
	}
}

// Score takes a deployment and evaluates its risk based on image component counts.
func (c *componentCountMultiplier) Score(_ context.Context, _ *storage.Deployment, images []*storage.Image) *storage.Risk_Result {
	// Get the number of components in the image.
	components := set.NewStringSet()
	var maxCount int
	var maxComponentImage *storage.Image
	for _, image := range images {
		for _, component := range image.GetScan().GetComponents() {
			components.Add(componentKey(component))
		}
		count := components.Cardinality()
		// Keep track of the image with the largest number of components.
		if count > maxCount {
			maxCount = count
			maxComponentImage = image
		}
	}
	return c.imageScorer.Score(allAccessCtx, maxComponentImage)
}

func componentKey(comp *storage.EmbeddedImageScanComponent) string {
	return fmt.Sprintf("%s:%s", comp.GetName(), comp.GetVersion())
}
