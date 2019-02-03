package resolvers

//go:generate go run ./gen

import (
	"context"
	"fmt"
	"reflect"

	violationsDatastore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/apitoken"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	complianceManager "github.com/stackrox/rox/central/compliance/manager"
	"github.com/stackrox/rox/central/compliance/manager/service"
	complianceService "github.com/stackrox/rox/central/compliance/service"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	complianceStore "github.com/stackrox/rox/central/compliance/store"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/store"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
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
	NetworkFlowStore            networkFlowStore.ClusterStore
	NodeGlobalStore             nodeStore.GlobalStore
	NotifierStore               notifierStore.Store
	PolicyDataStore             policyDatastore.DataStore
	ProcessIndicatorStore       processIndicatorStore.DataStore
	SecretsDataStore            secretDataStore.DataStore
	ViolationsDataStore         violationsDatastore.DataStore
}

// New returns a Resolver wired into the relevant data stores
func New() *Resolver {
	resolver := &Resolver{
		APITokenBackend:       apitoken.BackendSingleton(),
		ClusterDataStore:      clusterDatastore.Singleton(),
		DeploymentDataStore:   deploymentDatastore.Singleton(),
		ImageDataStore:        imageDatastore.Singleton(),
		GroupDataStore:        groupDataStore.Singleton(),
		NetworkFlowStore:      networkFlowStore.Singleton(),
		NodeGlobalStore:       nodeStore.Singleton(),
		NotifierStore:         notifierStore.Singleton(),
		PolicyDataStore:       policyDatastore.Singleton(),
		ProcessIndicatorStore: processIndicatorStore.Singleton(),
		SecretsDataStore:      secretDataStore.Singleton(),
		ViolationsDataStore:   violationsDatastore.Singleton(),
	}
	if features.Compliance.Enabled() {
		resolver.ComplianceStandardStore = complianceStandards.RegistrySingleton()
		resolver.ComplianceDataStore = complianceStore.Singleton()
		resolver.ComplianceManagementService = service.Singleton()
		resolver.ComplianceManager = complianceManager.Singleton()
		resolver.ComplianceService = complianceService.Singleton()
	}
	return resolver
}

var (
	readAlerts                = readAuth(resources.Alert)
	readTokens                = readAuth(resources.APIToken)
	readClusters              = readAuth(resources.Cluster)
	readCompliance            = readAuth(resources.Compliance)
	readComplianceRuns        = readAuth(resources.ComplianceRuns)
	readComplianceRunSchedule = readAuth(resources.ComplianceRunSchedule)
	readDeployments           = readAuth(resources.Deployment)
	readGroups                = readAuth(resources.Group)
	readImages                = readAuth(resources.Image)
	readIndicators            = readAuth(resources.Indicator)
	readNodes                 = readAuth(resources.Node)
	readNotifiers             = readAuth(resources.Notifier)
	readPolicies              = readAuth(resources.Policy)
	readSecrets               = readAuth(resources.Secret)

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
