package service

import (
	"context"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that searches various categories of resources
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.SearchServiceServer
}

// Builder provides the interface to build a search service.
type Builder interface {
	WithAlertStore(store alertDataStore.DataStore) Builder
	WithDeploymentStore(store deploymentDataStore.DataStore) Builder
	WithImageStore(store imageDataStore.DataStore) Builder
	WithPolicyStore(store policyDataStore.DataStore) Builder
	WithSecretStore(store secretDataStore.DataStore) Builder
	WithServiceAccountStore(store serviceAccountDataStore.DataStore) Builder
	WithNodeStore(store nodeDataStore.GlobalDataStore) Builder
	WithNamespaceStore(store namespaceDataStore.DataStore) Builder
	WithRiskStore(store riskDataStore.DataStore) Builder
	WithRoleStore(store roleDataStore.DataStore) Builder
	WithRoleBindingStore(store roleBindingDataStore.DataStore) Builder
	WithClusterDataStore(store clusterDataStore.DataStore) Builder
	WithAggregator(aggregation.Aggregator) Builder

	Build() Service
}

type serviceBuilder struct {
	alerts          alertDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	images          imageDataStore.DataStore
	policies        policyDataStore.DataStore
	secrets         secretDataStore.DataStore
	serviceAccounts serviceAccountDataStore.DataStore
	nodes           nodeDataStore.GlobalDataStore
	namespaces      namespaceDataStore.DataStore
	risks           riskDataStore.DataStore
	roles           roleDataStore.DataStore
	bindings        roleBindingDataStore.DataStore
	clusters        clusterDataStore.DataStore

	aggregator aggregation.Aggregator
}

// NewBuilder returns an instance of a builder to build a search service
func NewBuilder() Builder {
	return &serviceBuilder{}
}

func (b *serviceBuilder) WithAlertStore(store alertDataStore.DataStore) Builder {
	b.alerts = store
	return b
}

func (b *serviceBuilder) WithDeploymentStore(store deploymentDataStore.DataStore) Builder {
	b.deployments = store
	return b
}

func (b *serviceBuilder) WithImageStore(store imageDataStore.DataStore) Builder {
	b.images = store
	return b
}

func (b *serviceBuilder) WithPolicyStore(store policyDataStore.DataStore) Builder {
	b.policies = store
	return b
}

func (b *serviceBuilder) WithSecretStore(store secretDataStore.DataStore) Builder {
	b.secrets = store
	return b
}

func (b *serviceBuilder) WithServiceAccountStore(store serviceAccountDataStore.DataStore) Builder {
	b.serviceAccounts = store
	return b
}

func (b *serviceBuilder) WithNodeStore(store nodeDataStore.GlobalDataStore) Builder {
	b.nodes = store
	return b
}

func (b *serviceBuilder) WithNamespaceStore(store namespaceDataStore.DataStore) Builder {
	b.namespaces = store
	return b
}

func (b *serviceBuilder) WithRiskStore(store riskDataStore.DataStore) Builder {
	b.risks = store
	return b
}

func (b *serviceBuilder) WithRoleStore(store roleDataStore.DataStore) Builder {
	b.roles = store
	return b
}

func (b *serviceBuilder) WithRoleBindingStore(store roleBindingDataStore.DataStore) Builder {
	b.bindings = store
	return b
}

func (b *serviceBuilder) WithAggregator(aggregator aggregation.Aggregator) Builder {
	b.aggregator = aggregator
	return b
}

func (b *serviceBuilder) WithClusterDataStore(store clusterDataStore.DataStore) Builder {
	b.clusters = store
	return b
}

func (b *serviceBuilder) Build() Service {
	s := serviceImpl{
		alerts:          b.alerts,
		deployments:     b.deployments,
		images:          b.images,
		policies:        b.policies,
		secrets:         b.secrets,
		serviceaccounts: b.serviceAccounts,
		nodes:           b.nodes,
		namespaces:      b.namespaces,
		risks:           b.risks,
		roles:           b.roles,
		bindings:        b.bindings,
		aggregator:      b.aggregator,
		clusters:        b.clusters,
	}
	s.initializeAuthorizer()
	return &s
}

// NewService returns a new search service
func NewService() Service {
	builder := NewBuilder().
		WithAlertStore(alertDataStore.Singleton()).
		WithDeploymentStore(deploymentDataStore.Singleton()).
		WithImageStore(imageDataStore.Singleton()).
		WithPolicyStore(policyDataStore.Singleton()).
		WithSecretStore(secretDataStore.Singleton()).
		WithServiceAccountStore(serviceAccountDataStore.Singleton()).
		WithNodeStore(nodeDataStore.Singleton()).
		WithNamespaceStore(namespaceDataStore.Singleton()).
		WithRiskStore(riskDataStore.Singleton()).
		WithRoleStore(roleDataStore.Singleton()).
		WithRoleBindingStore(roleBindingDataStore.Singleton()).
		WithAggregator(aggregation.Singleton()).
		WithClusterDataStore(clusterDataStore.Singleton())

	return builder.Build()
}
