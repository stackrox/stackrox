package component

import (
	"context"

	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/multipliers/component"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// Scorer is the object that encompasses the multipliers for evaluating component risk
type ImageScorer interface {
	Score(ctx context.Context, component scancomponent.ScanComponent, os string, imageComponent *storage.EmbeddedImageScanComponent, imageID string) *storage.Risk
}

// NewComponentScorer returns a new scorer that encompasses multipliers for evaluating component risk
func NewImageComponentScorer(riskSubject storage.RiskSubjectType, multipliers ...component.Multiplier) ImageScorer {
	scoreImpl := &componentImageScorerImpl{
		riskSubjectType:       riskSubject,
		ConfiguredMultipliers: multipliers,
	}

	return scoreImpl
}

type componentImageScorerImpl struct {
	riskSubjectType       storage.RiskSubjectType
	ConfiguredMultipliers []component.Multiplier
}

// Score takes a component and evaluates its risk
func (s *componentImageScorerImpl) Score(ctx context.Context, scanComponent scancomponent.ScanComponent, os string, imageComponent *storage.EmbeddedImageScanComponent, imageID string) *storage.Risk {
	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(ctx, scanComponent); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}
	if len(riskResults) == 0 {
		return nil
	}

	var componentID string
	var err error
	if features.FlattenCVEData.Enabled() {
		componentID, err = scancomponent.ComponentIDV2(imageComponent, imageID)
		if err != nil {
			log.Errorf("Unable to score %s: %v", scanComponent.GetName(), err)
			return nil
		}
	} else {
		componentID = scancomponent.ComponentID(scanComponent.GetName(), scanComponent.GetVersion(), os)
	}

	risk := &storage.Risk{
		Score:   overallScore,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:   componentID,
			Type: s.riskSubjectType,
		},
	}

	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Errorf("Scoring %s: %v", scanComponent.GetName(), err)
		return nil
	}
	risk.Id = riskID

	return risk
}
