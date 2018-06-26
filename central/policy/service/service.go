package service

import (
	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	notifierStore "bitbucket.org/stack-rox/apollo/central/notifier/store"
	"bitbucket.org/stack-rox/apollo/central/policy/datastore"
	"bitbucket.org/stack-rox/apollo/central/search/options"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetPolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Policy, error)
	ListPolicies(ctx context.Context, request *v1.RawQuery) (*v1.ListPoliciesResponse, error)
	PostPolicy(ctx context.Context, request *v1.Policy) (*v1.Policy, error)
	PutPolicy(ctx context.Context, request *v1.Policy) (*empty.Empty, error)
	PatchPolicy(ctx context.Context, request *v1.PatchPolicyRequest) (*empty.Empty, error)
	DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
	ReassessPolicies(context.Context, *empty.Empty) (*empty.Empty, error)
	DryRunPolicy(ctx context.Context, request *v1.Policy) (*v1.DryRunResponse, error)
	GetPolicyCategories(context.Context, *empty.Empty) (*v1.PolicyCategoriesResponse, error)
	RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) (*empty.Empty, error)
	DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(policies datastore.DataStore, clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore, notifiers notifierStore.Store, detector *detection.Detector) Service {
	return &serviceImpl{
		policies:    policies,
		clusters:    clusters,
		deployments: deployments,
		notifiers:   notifiers,

		detector: detector,
		parser: &search.QueryParser{
			OptionsMap: options.AllOptionsMaps,
		},

		validator: newPolicyValidator(notifiers, clusters),
	}
}
