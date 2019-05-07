package resolvers

//go:generate go run ./gen

import (
	"context"
	"fmt"
	"reflect"

	violationsDatastore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/apitoken"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	complianceManager "github.com/stackrox/rox/central/compliance/manager"
	"github.com/stackrox/rox/central/compliance/manager/service"
	complianceService "github.com/stackrox/rox/central/compliance/service"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	complianceStore "github.com/stackrox/rox/central/compliance/store"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/store"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	networkFlowStoreSingleton "github.com/stackrox/rox/central/networkflow/store/singleton"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	k8sroleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8srolebindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/role/resources"
	roleStore "github.com/stackrox/rox/central/role/store"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
	ComplianceAggregator        aggregation.Aggregator
	APITokenBackend             apitoken.Backend
	ClusterDataStore            clusterDatastore.DataStore
	ComplianceDataStore         complianceStore.Store
	ComplianceStandardStore     complianceStandards.Repository
	ComplianceService           v1.ComplianceServiceServer
	ComplianceManagementService v1.ComplianceManagementServiceServer
	ComplianceManager           complianceManager.ComplianceManager
	DeploymentDataStore         deploymentDatastore.DataStore
	ImageDataStore              imageDatastore.DataStore
	GroupDataStore              groupDataStore.Store
	NamespaceDataStore          namespaceDataStore.DataStore
	NetworkFlowStore            networkFlowStore.ClusterStore
	NetworkPoliciesStore        networkPoliciesStore.Store
	NodeGlobalDataStore         nodeDataStore.GlobalDataStore
	NotifierStore               notifierStore.Store
	PolicyDataStore             policyDatastore.DataStore
	ProcessIndicatorStore       processIndicatorStore.DataStore
	K8sRoleStore                k8sroleStore.DataStore
	K8sRoleBindingStore         k8srolebindingStore.DataStore
	RoleStore                   roleStore.Store
	SecretsDataStore            secretDataStore.DataStore
	ServiceAccountsDataStore    serviceAccountDataStore.DataStore
	ViolationsDataStore         violationsDatastore.DataStore
}

// New returns a Resolver wired into the relevant data stores
func New() *Resolver {
	resolver := &Resolver{
		ComplianceAggregator:        aggregation.Singleton(),
		APITokenBackend:             apitoken.BackendSingleton(),
		ComplianceDataStore:         complianceStore.Singleton(),
		ComplianceStandardStore:     complianceStandards.RegistrySingleton(),
		ComplianceManagementService: service.Singleton(),
		ComplianceManager:           complianceManager.Singleton(),
		ComplianceService:           complianceService.Singleton(),
		ClusterDataStore:            clusterDatastore.Singleton(),
		DeploymentDataStore:         deploymentDatastore.Singleton(),
		ImageDataStore:              imageDatastore.Singleton(),
		GroupDataStore:              groupDataStore.Singleton(),
		NamespaceDataStore:          namespaceDataStore.Singleton(),
		NetworkPoliciesStore:        networkPoliciesStore.Singleton(),
		NetworkFlowStore:            networkFlowStoreSingleton.Singleton(),
		NodeGlobalDataStore:         nodeDataStore.Singleton(),
		NotifierStore:               notifierStore.Singleton(),
		PolicyDataStore:             policyDatastore.Singleton(),
		ProcessIndicatorStore:       processIndicatorStore.Singleton(),
		K8sRoleStore:                k8sroleStore.Singleton(),
		K8sRoleBindingStore:         k8srolebindingStore.Singleton(),
		RoleStore:                   roleStore.Singleton(),
		SecretsDataStore:            secretDataStore.Singleton(),
		ServiceAccountsDataStore:    serviceAccountDataStore.Singleton(),
		ViolationsDataStore:         violationsDatastore.Singleton(),
	}
	return resolver
}

//lint:file-ignore U1000 It's okay for some of the variables below to be unused.
var (
	readAlerts                 = readAuth(resources.Alert)
	readTokens                 = readAuth(resources.APIToken)
	readClusters               = readAuth(resources.Cluster)
	readCompliance             = readAuth(resources.Compliance)
	readComplianceRuns         = readAuth(resources.ComplianceRuns)
	readComplianceRunSchedule  = readAuth(resources.ComplianceRunSchedule)
	readDeployments            = readAuth(resources.Deployment)
	readGroups                 = readAuth(resources.Group)
	readImages                 = readAuth(resources.Image)
	readIndicators             = readAuth(resources.Indicator)
	readNamespaces             = readAuth(resources.Namespace)
	readNodes                  = readAuth(resources.Node)
	readNotifiers              = readAuth(resources.Notifier)
	readPolicies               = readAuth(resources.Policy)
	readK8sRoles               = readAuth(resources.K8sRole)
	readK8sRoleBindings        = readAuth(resources.K8sRoleBinding)
	readK8sSubjects            = readAuth(resources.K8sSubject)
	readRoles                  = readAuth(resources.Role)
	readSecrets                = readAuth(resources.Secret)
	readServiceAccounts        = readAuth(resources.ServiceAccount)
	writeCompliance            = writeAuth(resources.Compliance)
	writeComplianceRuns        = writeAuth(resources.ComplianceRuns)
	writeComplianceRunSchedule = writeAuth(resources.ComplianceRunSchedule)
)

type authorizerOverride struct{}

// SetAuthorizerOverride returns a context that will override the default permissions checking with custom
// logic. This is for testing only. It also feels pretty dangerous.
func SetAuthorizerOverride(ctx context.Context, authorizer authz.Authorizer) context.Context {
	return context.WithValue(ctx, authorizerOverride{}, authorizer)
}

func applyAuthorizer(authorizer authz.Authorizer) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		override := ctx.Value(authorizerOverride{})
		if override != nil {
			return override.(authz.Authorizer).Authorized(ctx, "graphql")
		}
		return authorizer.Authorized(ctx, "graphql")
	}
}

func readAuth(resource permissions.Resource) func(ctx context.Context) error {
	return applyAuthorizer(user.With(permissions.View(resource)))
}

func writeAuth(resource permissions.Resource) func(ctx context.Context) error {
	return applyAuthorizer(user.With(permissions.Modify(resource)))
}

func stringSlice(inputSlice interface{}) []string {
	r := reflect.ValueOf(inputSlice)
	output := make([]string, r.Len())
	for i := 0; i < r.Len(); i++ {
		output[i] = fmt.Sprint(r.Index(i).Interface())
	}
	return output
}
