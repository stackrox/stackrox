package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDetection "github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichanddetect"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/policy/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"golang.org/x/net/context"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetPolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Policy, error)
	ListPolicies(ctx context.Context, request *v1.RawQuery) (*v1.ListPoliciesResponse, error)
	PostPolicy(ctx context.Context, request *v1.Policy) (*v1.Policy, error)
	PutPolicy(ctx context.Context, request *v1.Policy) (*v1.Empty, error)
	PatchPolicy(ctx context.Context, request *v1.PatchPolicyRequest) (*v1.Empty, error)
	DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error)

	ReassessPolicies(context.Context, *v1.Empty) (*v1.Empty, error)
	DryRunPolicy(ctx context.Context, request *v1.Policy) (*v1.DryRunResponse, error)

	GetPolicyCategories(context.Context, *v1.Empty) (*v1.PolicyCategoriesResponse, error)
	RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) (*v1.Empty, error)
	DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) (*v1.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(policies datastore.DataStore,
	clusters clusterDataStore.DataStore,
	deployments deploymentDataStore.DataStore,
	notifiers notifierStore.Store,
	processes processIndicatorDataStore.DataStore,
	buildTimePolicies imageDetection.PolicySet,
	manager lifecycle.Manager,
	processor notifierProcessor.Processor,
	enricherAndDetector enrichanddetect.EnricherAndDetector) Service {
	return &serviceImpl{
		policies:    policies,
		clusters:    clusters,
		deployments: deployments,
		processes:   processes,

		buildTimePolicies: buildTimePolicies,
		lifecycleManager:  manager,

		processor:           processor,
		enricherAndDetector: enricherAndDetector,

		validator: newPolicyValidator(notifiers, clusters),
	}
}
