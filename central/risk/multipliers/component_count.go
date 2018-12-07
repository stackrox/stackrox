package multipliers

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// ComponentCountHeading is the risk result name for scores calculated by this multiplier.
	ComponentCountHeading = "Number of Components in Image"

	componentCountFloor = 10
	componentCountCeil  = 20
	maxScore            = 1.5
)

// componentCountMultiplier is a scorer for the number of components in an image.
type componentCountMultiplier struct{}

// NewComponentCount provides a multiplier that scores the data based on the the number of components in images.
func NewComponentCount() Multiplier {
	return &componentCountMultiplier{}
}

// Score takes a deployment and evaluates its risk based on image component counts.
func (c *componentCountMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	// Get the number of components in the image.
	components := set.NewStringSet()
	var maxCount int
	var maxImage string
	for _, container := range deployment.GetContainers() {
		for _, component := range container.GetImage().GetScan().GetComponents() {
			components.Add(componentKey(component))
		}
		count := components.Cardinality()
		if count > maxCount {
			maxCount = count
			maxImage = container.GetImage().GetName().GetFullName()
		}
	}

	// This does not contribute to the overall risk of the container
	if maxCount < componentCountFloor {
		return nil
	}

	// Linear increase between 10 components and 20 components from weight of 1 to 1.5.
	score := float32(1.0) + float32(maxCount-componentCountFloor)/float32(componentCountCeil-componentCountFloor)/float32(2)
	if score > maxScore {
		score = maxScore
	}

	// Generate a message depending on whether or not we have full name for the image.
	var message string
	if maxImage == "" {
		message = fmt.Sprintf("an image contains %d components", maxCount)
	} else {
		message = fmt.Sprintf("image %s contains %d components", maxImage, maxCount)
	}

	return &v1.Risk_Result{
		Name: ComponentCountHeading,
		Factors: []*v1.Risk_Result_Factor{
			{Message: message},
		},
		Score: score,
	}
}

func componentKey(comp *storage.ImageScanComponent) string {
	return fmt.Sprintf("%s:%s", comp.GetName(), comp.GetVersion())
}
