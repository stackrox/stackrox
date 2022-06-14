package node

import (
	"github.com/stackrox/stackrox/central/risk/multipliers/component/node"
	"github.com/stackrox/stackrox/central/risk/scorer/component"
	"github.com/stackrox/stackrox/generated/storage"
)

// NewNodeComponentScorer returns a new scorer that encompasses multipliers for evaluating node component risk
func NewNodeComponentScorer() component.Scorer {
	return component.NewComponentScorer(storage.RiskSubjectType_NODE_COMPONENT, node.NewVulnerabilities())
}
