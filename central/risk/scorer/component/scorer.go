package component

import (
	"context"

	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/multipliers/component"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scancomponent"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating component risk
type Scorer interface {
	Score(ctx context.Context, component scancomponent.ScanComponent, os string) *storage.Risk
}

// NewComponentScorer returns a new scorer that encompasses multipliers for evaluating component risk
func NewComponentScorer(riskSubject storage.RiskSubjectType, multipliers ...component.Multiplier) Scorer {
	scoreImpl := &componentScorerImpl{
		riskSubjectType:       riskSubject,
		ConfiguredMultipliers: multipliers,
	}

	return scoreImpl
}

type componentScorerImpl struct {
	riskSubjectType       storage.RiskSubjectType
	ConfiguredMultipliers []component.Multiplier
}

// Score takes a component and evaluates its risk
func (s *componentScorerImpl) Score(ctx context.Context, scanComponent scancomponent.ScanComponent, os string) *storage.Risk {
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

	risk := &storage.Risk{
		Score:   overallScore,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:   scancomponent.ComponentID(scanComponent.GetName(), scanComponent.GetVersion(), os),
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
