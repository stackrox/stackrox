package deployment

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mathutil"
)

type imageMultiplier struct {
	heading string
}

// NewImageMultiplier creates a deployment multiplier that is based on already computed image data
func NewImageMultiplier(imageHeading string) Multiplier {
	return &imageMultiplier{
		heading: imageHeading,
	}
}

// Score takes a deployment's images and evaluates the heading based on the risk already computed
func (i *imageMultiplier) Score(_ context.Context, _ *storage.Deployment, imageRiskResults map[string][]*storage.Risk_Result) *storage.Risk_Result {
	riskResults := imageRiskResults[i.heading]
	if len(riskResults) == 0 {
		return nil
	}

	var maxScore float32
	var factors []*storage.Risk_Result_Factor
	for _, r := range riskResults {
		factors = append(factors, r.Factors...)
		maxScore = mathutil.MaxFloat32(maxScore, r.Score)
	}

	return &storage.Risk_Result{
		Name:    i.heading,
		Factors: factors,
		Score:   maxScore,
	}
}
