package data

import (
	"context"

	alertStore "github.com/stackrox/rox/central/alert/datastore"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	complianceOperatorDataStore "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	imageStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	policiesStore "github.com/stackrox/rox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8sBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/pkg/images/integration"
)

// RepositoryFactory allows creating `ComplianceDataRepository`s to be used in compliance runs.
type RepositoryFactory interface {
	CreateDataRepository(ctx context.Context, domain framework.ComplianceDomain) (framework.ComplianceDataRepository, error)
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

func (f *factory) CreateDataRepository(ctx context.Context, domain framework.ComplianceDomain) (framework.ComplianceDataRepository, error) {
	return newRepository(ctx, domain, f)
}

//go:generate mockgen-wrapper
