package data

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
)

// RepositoryFactory allows creating `ComplianceDataRepository`s to be used in compliance runs.
type RepositoryFactory interface {
	CreateDataRepository(domain framework.ComplianceDomain) (framework.ComplianceDataRepository, error)
}

type factory struct {
	networkPoliciesStore  networkPoliciesStore.Store
	networkGraphEvaluator graph.Evaluator
}

// NewDefaultFactory creates a new RepositoryFactory using the default instances for accessing data.
func NewDefaultFactory() RepositoryFactory {
	return &factory{
		networkPoliciesStore:  networkPoliciesStore.Singleton(),
		networkGraphEvaluator: graph.Singleton(),
	}
}

func (f *factory) CreateDataRepository(domain framework.ComplianceDomain) (framework.ComplianceDataRepository, error) {
	return newRepository(domain, f)
}
