package image

import (
	"context"
	"fmt"

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

// Score takes an image and evaluates its risk based on component counts.
func (c *componentCountMultiplier) Score(_ context.Context, image *storage.Image) *storage.Risk_Result {
	// Get the number of components in the image.
	components := set.NewStringSet()
	for _, component := range image.GetScan().GetComponents() {
		components.Add(componentKey(component))
	}
	count := components.Cardinality()
	score := GetComponentCountRiskScore(count)
	if score == 0.0 {
		return nil
	}
	message := GetComponentCountRiskFactorMsg(image.GetName().GetFullName(), count)
	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage(message)
	rr := &storage.Risk_Result{}
	rr.SetName(ComponentCountHeading)
	rr.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	rr.SetScore(score)
	return rr
}

// ScoreV2 takes an image and evaluates its risk based on component counts.
func (c *componentCountMultiplier) ScoreV2(_ context.Context, image *storage.ImageV2) *storage.Risk_Result {
	// Get the number of components in the image.
	components := set.NewStringSet()
	for _, component := range image.GetScan().GetComponents() {
		components.Add(componentKey(component))
	}
	count := components.Cardinality()
	score := GetComponentCountRiskScore(count)
	if score == 0.0 {
		return nil
	}
	message := GetComponentCountRiskFactorMsg(image.GetName().GetFullName(), count)
	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage(message)
	rr := &storage.Risk_Result{}
	rr.SetName(ComponentCountHeading)
	rr.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	rr.SetScore(score)
	return rr
}

func componentKey(comp *storage.EmbeddedImageScanComponent) string {
	return fmt.Sprintf("%s:%s", comp.GetName(), comp.GetVersion())
}

// GetComponentCountRiskScore return risk score for given component count.
func GetComponentCountRiskScore(componentCount int) (riskScore float32) {
	// This does not contribute to the overall risk of the container
	if componentCount < componentCountFloor {
		return
	}

	// Linear increase between 10 components and 20 components from weight of 1 to 1.5.
	riskScore = float32(1.0) + float32(componentCount-componentCountFloor)/float32(componentCountCeil-componentCountFloor)/float32(2)
	if riskScore > maxScore {
		riskScore = maxScore
	}
	return
}

// GetComponentCountRiskFactorMsg returns message for component count risk
func GetComponentCountRiskFactorMsg(imageFullName string, componentCount int) (message string) {
	// Generate a message depending on whether or not we have full name for the image.
	if imageFullName == "" {
		message = fmt.Sprintf("An image contains %d components", componentCount)
	} else {
		message = fmt.Sprintf("Image %q contains %d components", imageFullName, componentCount)
	}
	return
}
