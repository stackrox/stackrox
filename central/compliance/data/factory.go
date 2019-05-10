package data

import (
	alertStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	complianceStore "github.com/stackrox/rox/central/compliance/store"
	imageStore "github.com/stackrox/rox/central/image/datastore"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	networkFlowStoreSingleton "github.com/stackrox/rox/central/networkflow/store/singleton"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	policiesStore "github.com/stackrox/rox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8sBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

// RepositoryFactory allows creating `ComplianceDataRepository`s to be used in compliance runs.
type RepositoryFactory interface {
	CreateDataRepository(domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn) (framework.ComplianceDataRepository, error)
}

type factory struct {
	alertStore            alertStore.DataStore
	networkPoliciesStore  npDS.DataStore
	networkGraphEvaluator graph.Evaluator
	policyStore           policiesStore.DataStore
	imageStore            imageStore.DataStore
	imageIntegrationStore imageIntegrationStore.DataStore
	processIndicatorStore processIndicatorStore.DataStore
	networkFlowStore      networkFlowStore.ClusterStore
	notifierStore         notifierStore.Store
	complianceStore       complianceStore.Store
	standardsRepo         standards.Repository
	roleDataStore         k8sRoleDataStore.DataStore
	bindingDataStore      k8sBindingDataStore.DataStore
}

// NewDefaultFactory creates a new RepositoryFactory using the default instances for accessing data.
func NewDefaultFactory() RepositoryFactory {
	return &factory{
		alertStore:            alertStore.Singleton(),
		networkPoliciesStore:  npDS.Singleton(),
		networkGraphEvaluator: graph.Singleton(),
		policyStore:           policiesStore.Singleton(),
		imageStore:            imageStore.Singleton(),
		imageIntegrationStore: imageIntegrationStore.Singleton(),
		processIndicatorStore: processIndicatorStore.Singleton(),
		networkFlowStore:      networkFlowStoreSingleton.Singleton(),
		notifierStore:         notifierStore.Singleton(),
		complianceStore:       complianceStore.Singleton(),
		standardsRepo:         standards.RegistrySingleton(),
		roleDataStore:         k8sRoleDataStore.Singleton(),
		bindingDataStore:      k8sBindingDataStore.Singleton(),
	}
}

func (f *factory) CreateDataRepository(domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn) (framework.ComplianceDataRepository, error) {
	return newRepository(domain, scrapeResults, f)
}

//go:generate mockgen-wrapper RepositoryFactory
