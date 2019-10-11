package deployment

import (
	"context"

	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// riskyComponentCountMultiplier is a scorer for the components in an image that can be used by attackers.
type riskyComponentCountMultiplier struct {
	imageScorer       imageMultiplier.Multiplier
	riskyComponentSet set.StringSet
}

// NewRiskyComponents provides a multiplier that scores the data based on the the number of risky components in image.
func NewRiskyComponents() Multiplier {
	return &riskyComponentCountMultiplier{
		imageScorer:       imageMultiplier.NewRiskyComponents(),
		riskyComponentSet: imageMultiplier.RiskyComponents,
	}
}

// Score takes deployment's images and evaluates its risk based on number risky image component.
func (c *riskyComponentCountMultiplier) Score(_ context.Context, _ *storage.Deployment, images []*storage.Image) *storage.Risk_Result {
	// Get the largest number of risky components in an image
	var largestRiskySet *set.StringSet
	var riskiestImage *storage.Image
	for _, image := range images {
		// Create a name to version map of all the image components.
		presentComponents := set.NewStringSet()
		for _, component := range image.GetScan().GetComponents() {
			presentComponents.Add(component.GetName())
		}

		// Count how many known risky components match a labeled component.
		riskySet := c.riskyComponentSet.Intersect(presentComponents)

		// Keep track of the image with the largest number of risky components.
		if largestRiskySet == nil || riskySet.Cardinality() > largestRiskySet.Cardinality() {
			largestRiskySet = &riskySet
			riskiestImage = image
		}
	}

	return c.imageScorer.Score(allAccessCtx, riskiestImage)
}
