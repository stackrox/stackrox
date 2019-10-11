package imagecomponent

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/risk/datastore"
	imagecomponent "github.com/stackrox/rox/central/risk/multipliers/image_component"
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
		ConfiguredMultipliers: []imagecomponent.Multiplier{
			imagecomponent.NewVulnerabilities(),
		},
	}

	return scoreImpl
}

type imageComponentScorerImpl struct {
	ConfiguredMultipliers []imagecomponent.Multiplier
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

	imageComponentID := fmt.Sprintf("%s:%s", imageComponent.GetName(), imageComponent.GetVersion())
	risk := &storage.Risk{
		Score:   overallScore,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:   imageComponentID,
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
