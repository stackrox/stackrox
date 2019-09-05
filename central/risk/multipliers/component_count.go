package multipliers

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/risk"
	"github.com/stackrox/rox/pkg/set"
)

const (
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

// Score takes a image and evaluates its risk based on image component counts.
func (c *componentCountMultiplier) Score(_ context.Context, msg proto.Message) *storage.Risk_Result {
	image, ok := msg.(*storage.Image)
	if !ok {
		return nil
	}
	// Get the number of components in the image.
	components := set.NewStringSet()
	for _, component := range image.GetScan().GetComponents() {
		components.Add(componentKey(component))
	}
	count := components.Cardinality()

	// This does not contribute to the overall risk of the container
	if count < componentCountFloor {
		return nil
	}

	// Linear increase between 10 components and 20 components from weight of 1 to 1.5.
	score := float32(1.0) + float32(count-componentCountFloor)/float32(componentCountCeil-componentCountFloor)/float32(2)
	if score > maxScore {
		score = maxScore
	}

	// Generate a message depending on whether or not we have full name for the image.
	var message string
	if image.GetName().GetFullName() == "" {
		message = fmt.Sprintf("An image contains %d components", count)
	} else {
		message = fmt.Sprintf("Image %s contains %d components", image.GetName().GetFullName(), count)
	}

	return &storage.Risk_Result{
		Name: risk.ImageComponentCount.DisplayTitle,
		Factors: []*storage.Risk_Result_Factor{
			{Message: message},
		},
		Score: score,
	}
}

func componentKey(comp *storage.ImageScanComponent) string {
	return fmt.Sprintf("%s:%s", comp.GetName(), comp.GetVersion())
}
