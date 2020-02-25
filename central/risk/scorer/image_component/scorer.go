package imagecomponent

import (
	"context"

	"github.com/stackrox/rox/central/imagecomponent"
	"github.com/stackrox/rox/central/risk/datastore"
	imageComponentMultipliers "github.com/stackrox/rox/central/risk/multipliers/image_component"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating image component risk
type Scorer interface {
	Score(ctx context.Context, imageComponent *storage.EmbeddedImageScanComponent) *storage.Risk
}

// NewImageComponentScorer returns a new scorer that encompasses multipliers for evaluating image component risk
func NewImageComponentScorer() Scorer {
	scoreImpl := &imageComponentScorerImpl{
		ConfiguredMultipliers: []imageComponentMultipliers.Multiplier{
			imageComponentMultipliers.NewVulnerabilities(),
		},
	}

	return scoreImpl
}

type imageComponentScorerImpl struct {
	ConfiguredMultipliers []imageComponentMultipliers.Multiplier
}

// Score takes a image component and evaluates its risk
func (s *imageComponentScorerImpl) Score(ctx context.Context, imageComponent *storage.EmbeddedImageScanComponent) *storage.Risk {
	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(ctx, imageComponent); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}
	if len(riskResults) == 0 {
		return nil
	}

	imageComponentID := imagecomponent.ComponentID{
		Name:    imageComponent.GetName(),
		Version: imageComponent.GetVersion(),
	}
	risk := &storage.Risk{
		Score:   overallScore,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:   imageComponentID.ToString(),
			Type: storage.RiskSubjectType_IMAGE_COMPONENT,
		},
	}

	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Error(err)
		return nil
	}
	risk.Id = riskID

	return risk
}
