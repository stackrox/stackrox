package scorer

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/risk"
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type scoreImpl struct {
	ConfiguredMultipliers map[string]multipliers.Multiplier
}

// Score takes a deployment and evaluates its risk
func (s *scoreImpl) Score(ctx context.Context, msg proto.Message, riskIndicators ...risk.Indicator) *storage.Risk {
	riskResults, score := s.computeOverallScore(ctx, msg, riskIndicators...)
	if len(riskResults) == 0 {
		return nil
	}
	risk := risk.BuildRiskProtoForEntity(msg)
	risk.Score = score
	risk.Results = riskResults

	return risk
}

func (s *scoreImpl) computeOverallScore(ctx context.Context, msg proto.Message, riskIndicators ...risk.Indicator) ([]*storage.Risk_Result, float32) {
	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)

	for _, riskIndicator := range riskIndicators {
		multiplier, configured := s.ConfiguredMultipliers[riskIndicator.DisplayTitle]
		if !configured {
			logging.Panicf("multiplier for %q not configured", riskIndicator)
		}

		if riskResult := multiplier.Score(ctx, msg); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}

	return riskResults, overallScore
}
