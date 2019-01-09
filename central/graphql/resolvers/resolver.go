package resolvers

//go:generate go run ./gen

import (
	"context"
	"fmt"
	"reflect"

	violationsDatastore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/apitoken"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	complianceDataStore "github.com/stackrox/rox/central/compliance/datastore"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/store"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	nodeStore "github.com/stackrox/rox/central/node/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
	APITokenBackend         apitoken.Backend
	ClusterDataStore        clusterDatastore.DataStore
	ComplianceDataStore     complianceDataStore.DataStore
	ComplianceStandardStore complianceStandards.Standards
	DeploymentDataStore     deploymentDatastore.DataStore
	ImageDataStore          imageDatastore.DataStore
	GroupDataStore          groupDataStore.Store
	NetworkFlowStore        networkFlowStore.ClusterStore
	NodeGlobalStore         nodeStore.GlobalStore
	NotifierStore           notifierStore.Store
	PolicyDataStore         policyDatastore.DataStore
	ProcessIndicatorStore   processIndicatorStore.DataStore
	SecretsDataStore        secretDataStore.DataStore
	ViolationsDataStore     violationsDatastore.DataStore
}

// New returns a Resolver wired into the relevant data stores
func New() *Resolver {
	return &Resolver{
		APITokenBackend:       apitoken.BackendSingleton(),
		ClusterDataStore:      clusterDatastore.Singleton(),
		ComplianceDataStore:   complianceDataStore.Fake(),
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
}

var (
	alertAuth      = readAuth(resources.Alert)
	apiTokenAuth   = readAuth(resources.APIToken)
	clusterAuth    = readAuth(resources.Cluster)
	complianceAuth = readAuth(resources.Compliance)
	deploymentAuth = readAuth(resources.Deployment)
	groupAuth      = readAuth(resources.Group)
	imageAuth      = readAuth(resources.Image)
	indicatorAuth  = readAuth(resources.Indicator)
	nodeAuth       = readAuth(resources.Node)
	notifierAuth   = readAuth(resources.Notifier)
	policyAuth     = readAuth(resources.Policy)
	secretAuth     = readAuth(resources.Secret)
)

type authorizerOverride struct{}

// SetAuthorizerOverride returns a context that will override the default permissions checking with custom
// logic. This is for testing only. It also feels pretty dangerous.
func SetAuthorizerOverride(ctx context.Context, authorizer authz.Authorizer) context.Context {
	return context.WithValue(ctx, authorizerOverride{}, authorizer)
}

func readAuth(resource permissions.Resource) func(ctx context.Context) error {
	authorizer := user.With(permissions.View(resource))
	return func(ctx context.Context) error {
		override := ctx.Value(authorizerOverride{})
		if override != nil {
			return override.(authz.Authorizer).Authorized(ctx, "graphql")
		}
		return authorizer.Authorized(ctx, "graphql")
	}
}

func stringSlice(inputSlice interface{}) []string {
	r := reflect.ValueOf(inputSlice)
	output := make([]string, r.Len())
	for i := 0; i < r.Len(); i++ {
		output[i] = fmt.Sprint(r.Index(i).Interface())
	}
	return output
}
