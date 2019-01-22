package data

import (
	alertStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/compliance/framework"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	policiesStore "github.com/stackrox/rox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

// RepositoryFactory allows creating `ComplianceDataRepository`s to be used in compliance runs.
type RepositoryFactory interface {
	CreateDataRepository(domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn) (framework.ComplianceDataRepository, error)
}

type factory struct {
	alertStore            alertStore.DataStore
	networkPoliciesStore  networkPoliciesStore.Store
	networkGraphEvaluator graph.Evaluator
	policyStore           policiesStore.DataStore
	imageIntegrationStore imageIntegrationStore.DataStore
	processIndicatorStore processIndicatorStore.DataStore
	networkFlowStore      networkFlowStore.ClusterStore
}

// NewDefaultFactory creates a new RepositoryFactory using the default instances for accessing data.
func NewDefaultFactory() RepositoryFactory {
	return &factory{
		alertStore:            alertStore.Singleton(),
		networkPoliciesStore:  networkPoliciesStore.Singleton(),
		networkGraphEvaluator: graph.Singleton(),
		policyStore:           policiesStore.Singleton(),
		imageIntegrationStore: imageIntegrationStore.Singleton(),
		processIndicatorStore: processIndicatorStore.Singleton(),
		networkFlowStore:      networkFlowStore.Singleton(),
	}
}

func (f *factory) CreateDataRepository(domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn) (framework.ComplianceDataRepository, error) {
	return newRepository(domain, scrapeResults, f)
}
