package service

import (
	"context"
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/central/networkgraph"
	"bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
	"bitbucket.org/stack-rox/apollo/central/role/resources"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/auth/permissions"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/perrpc"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkPolicy)): {
			"/v1.NetworkPolicyService/GetNetworkPolicy",
			"/v1.NetworkPolicyService/ListNetworkPolicies",
			"/v1.NetworkPolicyService/GetNetworkGraph",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	store          store.Store
	graphEvaluator networkgraph.GraphEvaluator
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

func (s *serviceImpl) ListNetworkPolicies(context.Context, *empty.Empty) (*v1.NetworkPoliciesResponse, error) {
	networkPolicies, err := s.store.GetNetworkPolicies()
	if err != nil {
		return nil, err
	}
	return &v1.NetworkPoliciesResponse{
		NetworkPolicies: networkPolicies,
	}, nil
}

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, query *v1.RawQuery) (*v1.GetNetworkGraphResponse, error) {
	return s.graphEvaluator.GetGraph()
}
