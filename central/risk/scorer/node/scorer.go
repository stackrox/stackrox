package node

import (
	"context"

	"github.com/stackrox/stackrox/central/risk/datastore"
	"github.com/stackrox/stackrox/central/risk/multipliers/node"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating node risk
type Scorer interface {
	Score(ctx context.Context, node *storage.Node) *storage.Risk
}

type nodeScorerImpl struct {
	ConfiguredMultipliers []node.Multiplier
}

// NewNodeScorer returns a new scorer that encompasses multipliers for evaluating node risk
func NewNodeScorer() Scorer {
	return &nodeScorerImpl{
		ConfiguredMultipliers: []node.Multiplier{
			node.NewVulnerabilities(),
		},
	}
}

func (s *nodeScorerImpl) Score(ctx context.Context, node *storage.Node) *storage.Risk {
	if node.GetId() == "" {
		return nil
	}

	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(ctx, node); riskResult != nil {
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
			Id:        node.GetId(),
			Type:      storage.RiskSubjectType_NODE,
			ClusterId: node.GetClusterId(),
		},
	}

	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Errorf("Unable to get Risk ID for node %s: %v", node.GetName(), err)
		return nil
	}
	risk.Id = riskID

	return risk
}
