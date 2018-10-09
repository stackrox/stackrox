package service

import (
	"context"

	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkGraph)): {
			"/v1.NetworkGraphService/GetNetworkGraph",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterStore networkFlowStore.ClusterStore
	deployments  deploymentDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNetworkGraphServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNetworkGraphServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetNetworkGraph(context context.Context, request *v1.NetworkGraphRequest) (*v1.NetworkGraph, error) {
	if request.GetClusterId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster ID must be specified")
	}

	if request.GetSince() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Request parameter Since must be specified")
	}

	// Get the deployments we want to check connectivity between.
	deployments, err := s.getDeployments(request.GetClusterId())

	if err != nil {
		return nil, err
	}

	// compute nodes
	var nodes []*v1.NetworkNode

	for _, d := range deployments {
		nodes = append(nodes, &v1.NetworkNode{
			Id:             d.GetId(),
			DeploymentName: d.GetName(),
			Cluster:        d.GetClusterName(),
			Namespace:      d.GetNamespace(),
		})
	}

	flowStore := s.clusterStore.GetFlowStore(request.GetClusterId())

	if flowStore == nil {
		return nil, status.Errorf(codes.NotFound, "no flows found for cluster %s", request.GetClusterId())
	}

	flows, _, err := flowStore.GetAllFlows()
	if err != nil {
		return nil, err
	}

	// compute edges
	var edges []*v1.NetworkEdge

	filteredFlows := filterNetworkFlowsByTime(flows, request.GetSince())
	for _, flow := range filteredFlows {
		srcID := flow.GetProps().GetSrcDeploymentId()
		dstID := flow.GetProps().GetDstDeploymentId()

		edges = append(edges, &v1.NetworkEdge{Source: srcID, Target: dstID})
	}

	return &v1.NetworkGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func filterNetworkFlowsByTime(flows []*v1.NetworkFlow, since *google_protobuf.Timestamp) (filtered []*v1.NetworkFlow) {
	for _, flow := range flows {
		flowTS := flow.LastSeenTimestamp
		if flowTS.GetSeconds() > since.GetSeconds() ||
			(flowTS.GetSeconds() == since.GetSeconds() && flowTS.GetNanos() > since.GetNanos()) {
			filtered = append(filtered, flow)
		}
	}

	return
}

func (s *serviceImpl) getDeployments(clusterID string) (deployments []*v1.Deployment, err error) {
	clusterQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()

	deployments, err = s.deployments.SearchRawDeployments(clusterQuery)
	return
}
