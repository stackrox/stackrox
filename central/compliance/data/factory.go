package data

import (
	"context"

	alertStore "github.com/stackrox/stackrox/central/alert/datastore"
	complianceDS "github.com/stackrox/stackrox/central/compliance/datastore"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/central/compliance/standards"
	complianceOperatorDataStore "github.com/stackrox/stackrox/central/complianceoperator/checkresults/datastore"
	imageStore "github.com/stackrox/stackrox/central/image/datastore"
	"github.com/stackrox/stackrox/central/imageintegration"
	imageIntegrationStore "github.com/stackrox/stackrox/central/imageintegration/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	npDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	"github.com/stackrox/stackrox/central/networkpolicies/graph"
	notifierDataStore "github.com/stackrox/stackrox/central/notifier/datastore"
	policiesStore "github.com/stackrox/stackrox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/stackrox/central/processindicator/datastore"
	k8sRoleDataStore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	k8sBindingDataStore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/pkg/images/integration"
)

// RepositoryFactory allows creating `ComplianceDataRepository`s to be used in compliance runs.
type RepositoryFactory interface {
	CreateDataRepository(ctx context.Context, domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn) (framework.ComplianceDataRepository, error)
}

type factory struct {
	alertStore                    alertStore.DataStore
	networkPoliciesStore          npDS.DataStore
	networkGraphEvaluator         graph.Evaluator
	policyStore                   policiesStore.DataStore
	imageStore                    imageStore.DataStore
	imageIntegrationStore         imageIntegrationStore.DataStore
	imageIntegrationsSet          integration.Set
	processIndicatorStore         processIndicatorStore.DataStore
	networkFlowDataStore          nfDS.ClusterDataStore
	netTreeMgr                    networktree.Manager
	notifierDataStore             notifierDataStore.DataStore
	complianceStore               complianceDS.DataStore
	standardsRepo                 standards.Repository
	roleDataStore                 k8sRoleDataStore.DataStore
	bindingDataStore              k8sBindingDataStore.DataStore
	complianceOperatorResultStore complianceOperatorDataStore.DataStore
}

// NewDefaultFactory creates a new RepositoryFactory using the default instances for accessing data.
func NewDefaultFactory() RepositoryFactory {
	return &factory{
		alertStore:                    alertStore.Singleton(),
		networkPoliciesStore:          npDS.Singleton(),
		networkGraphEvaluator:         graph.Singleton(),
		policyStore:                   policiesStore.Singleton(),
		imageStore:                    imageStore.Singleton(),
		imageIntegrationStore:         imageIntegrationStore.Singleton(),
		imageIntegrationsSet:          imageintegration.Set(),
		processIndicatorStore:         processIndicatorStore.Singleton(),
		networkFlowDataStore:          nfDS.Singleton(),
		netTreeMgr:                    networktree.Singleton(),
		notifierDataStore:             notifierDataStore.Singleton(),
		complianceStore:               complianceDS.Singleton(),
		standardsRepo:                 standards.RegistrySingleton(),
		roleDataStore:                 k8sRoleDataStore.Singleton(),
		bindingDataStore:              k8sBindingDataStore.Singleton(),
		complianceOperatorResultStore: complianceOperatorDataStore.Singleton(),
	}
}

func (f *factory) CreateDataRepository(ctx context.Context, domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn) (framework.ComplianceDataRepository, error) {
	return newRepository(ctx, domain, scrapeResults, f)
}

//go:generate mockgen-wrapper
