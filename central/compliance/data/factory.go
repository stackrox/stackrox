package data

import (
	"github.com/stackrox/rox/central/compliance/framework"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	policiesStore "github.com/stackrox/rox/central/policy/datastore"
)

// RepositoryFactory allows creating `ComplianceDataRepository`s to be used in compliance runs.
type RepositoryFactory interface {
	CreateDataRepository(domain framework.ComplianceDomain) (framework.ComplianceDataRepository, error)
}

type factory struct {
	networkPoliciesStore  networkPoliciesStore.Store
	networkGraphEvaluator graph.Evaluator
	policyStore           policiesStore.DataStore
	imageIntegrationStore imageIntegrationStore.DataStore
}

// NewDefaultFactory creates a new RepositoryFactory using the default instances for accessing data.
func NewDefaultFactory() RepositoryFactory {
	return &factory{
		networkPoliciesStore:  networkPoliciesStore.Singleton(),
		networkGraphEvaluator: graph.Singleton(),
		policyStore:           policiesStore.Singleton(),
		imageIntegrationStore: imageIntegrationStore.Singleton(),
	}
}

func (f *factory) CreateDataRepository(domain framework.ComplianceDomain) (framework.ComplianceDataRepository, error) {
	return newRepository(domain, f)
}
