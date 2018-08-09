package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkPolicy)): {
			"/v1.NetworkPolicyService/GetNetworkPolicy",
			"/v1.NetworkPolicyService/GetNetworkPolicies",
			"/v1.NetworkPolicyService/GetNetworkGraph",
			"/v1.NetworkPolicyService/GetNetworkGraphEpoch",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	store          store.Store
	clusterStore   datastore.DataStore
	graphEvaluator networkgraph.Evaluator
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNetworkPolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterNetworkPolicyServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}

func populateYAML(np *v1.NetworkPolicy) {
	k8sNetworkPolicy := protoconv.ProtoNetworkPolicyWrap{NetworkPolicy: np}.ConvertNetworkPolicy()
	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

	stringBuilder := &strings.Builder{}
	err := encoder.Encode(k8sNetworkPolicy, stringBuilder)
	if err != nil {
		np.Yaml = fmt.Sprintf("Could not render Network Policy YAML: %s", err)
		return
	}
	np.Yaml = stringBuilder.String()
}

func (s *serviceImpl) GetNetworkPolicy(ctx context.Context, request *v1.ResourceByID) (*v1.NetworkPolicy, error) {
	networkPolicy, exists, err := s.store.GetNetworkPolicy(request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "network policy with id '%s' does not exist", request.GetId())
	}
	populateYAML(networkPolicy)
	return networkPolicy, nil
}

func (s *serviceImpl) GetNetworkPolicies(ctx context.Context, request *v1.GetNetworkPoliciesRequest) (*v1.NetworkPoliciesResponse, error) {
	if request.GetClusterId() != "" {
		_, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !exists {
			return nil, status.Errorf(codes.InvalidArgument, "cluster with id '%s' doesn't exist", request.GetClusterId())
		}
	}
	networkPolicies, err := s.store.GetNetworkPolicies(request)
	if err != nil {
		return nil, err
	}
	return &v1.NetworkPoliciesResponse{
		NetworkPolicies: networkPolicies,
	}, nil
}

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.GetNetworkGraphRequest) (*v1.GetNetworkGraphResponse, error) {
	if request.GetClusterId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster ID must be specified")
	}

	cluster, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster with ID '%s' does not exist", request.GetClusterId())
	}

	parsedSearch := new(v1.ParsedSearchRequest)
	if request.GetQuery() != "" {
		parsedSearch, err = (&search.QueryParser{}).ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return s.graphEvaluator.GetGraph(cluster, parsedSearch)
}

func (s *serviceImpl) GetNetworkGraphEpoch(context.Context, *empty.Empty) (*v1.GetNetworkGraphEpochResponse, error) {
	return &v1.GetNetworkGraphEpochResponse{
		Epoch: s.graphEvaluator.Epoch(),
	}, nil
}
