package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
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
		user.With(permissions.Modify(resources.Notifier)): {
			"/v1.NetworkPolicyService/SendNetworkPolicyYaml",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterStore    clusterDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	networkPolicies networkPoliciesStore.Store
	graphEvaluator  networkgraph.Evaluator
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNetworkPolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNetworkPolicyServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func populateYAML(np *v1.NetworkPolicy) {
	k8sNetworkPolicy := networkPolicyConversion.RoxNetworkPolicyWrap{NetworkPolicy: np}.ToKubernetesNetworkPolicy()
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
	networkPolicy, exists, err := s.networkPolicies.GetNetworkPolicy(request.GetId())
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
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(request)
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

	networkPolicies, err := s.getNetworkPolicies(cluster)
	if err != nil {
		return nil, err
	}

	deployments, err := s.getDeployments(cluster, request.GetQuery())
	if err != nil {
		return nil, err
	}

	return s.graphEvaluator.GetGraph(deployments, networkPolicies), nil
}

func (s *serviceImpl) GetNetworkGraphEpoch(context.Context, *v1.Empty) (*v1.GetNetworkGraphEpochResponse, error) {
	return &v1.GetNetworkGraphEpochResponse{
		Epoch: s.graphEvaluator.Epoch(),
	}, nil
}

func (s *serviceImpl) SendNetworkPolicyYaml(ctx context.Context, request *v1.SendNetworkPolicyYamlRequest) (*v1.Empty, error) {
	//TODO (@boo): Add implementation.
	return &v1.Empty{}, status.Error(codes.Unimplemented, "Not implemented")
}

func (s *serviceImpl) getNetworkPolicies(cluster *v1.Cluster) (networkPolicies []*v1.NetworkPolicy, err error) {
	if cluster.GetId() == "" {
		return nil, fmt.Errorf("cluster id must be present, but it isn't: %s", proto.MarshalTextString(cluster))
	}

	networkPolicies, err = s.networkPolicies.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: cluster.GetId()})
	return
}

func (s *serviceImpl) getDeployments(cluster *v1.Cluster, query string) (deployments []*v1.Deployment, err error) {
	var q *v1.Query
	if query != "" {
		q, err = search.ParseRawQuery(query)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		q = search.EmptyQuery()
	}

	q = search.ConjunctionQuery(q, search.NewQueryBuilder().AddStrings(search.ClusterID, cluster.GetId()).ProtoQuery())
	deployments, err = s.deployments.SearchRawDeployments(q)
	return
}
