package service

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
	defaultSince = -5 * time.Minute
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
		return nil, status.Errorf(codes.InvalidArgument, "cluster ID must be specified")
	}

	since := request.GetSince()
	if since == nil {
		var err error
		since, err = types.TimestampProto(time.Now().Add(defaultSince))
		if err != nil {
			errorhelpers.PanicOnDevelopment(err)
		}
	}

	// Get the deployments we want to check connectivity between.
	deployments, err := s.getDeployments(request.GetClusterId(), request.GetQuery())

	if err != nil {
		return nil, err
	}

	builder := newFlowGraphBuilder()
	builder.AddDeployments(deployments)

	flowStore := s.clusterStore.GetFlowStore(request.GetClusterId())

	if flowStore == nil {
		return nil, status.Errorf(codes.NotFound, "no flows found for cluster %s", request.GetClusterId())
	}

	flows, _, err := flowStore.GetAllFlows(since)
	if err != nil {
		return nil, err
	}

	// compute edges

	// Filter by deployments, and then by time.
	filteredFlows := filterNetworkFlowsByDeployments(flows, deployments)

	builder.AddFlows(filteredFlows)
	return builder.Build(), nil
}

func filterNetworkFlowsByDeployments(flows []*storage.NetworkFlow, deployments []*storage.Deployment) (filtered []*storage.NetworkFlow) {

	filtered = flows[:0]
	deploymentIDMap := make(map[string]bool)
	for _, d := range deployments {
		deploymentIDMap[d.Id] = true
	}

	for _, flow := range flows {
		srcEnt := flow.GetProps().GetSrcEntity()
		dstEnt := flow.GetProps().GetDstEntity()

		if srcEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && !deploymentIDMap[srcEnt.GetId()] {
			continue
		}
		if dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && !deploymentIDMap[dstEnt.GetId()] {
			continue
		}

		filtered = append(filtered, flow)
	}

	return
}

func (s *serviceImpl) getDeployments(clusterID string, query string) (deployments []*storage.Deployment, err error) {
	clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()

	q := clusterQuery
	if query != "" {
		q, err = search.ParseRawQuery(query)
		if err != nil {
			return
		}
		q = search.ConjunctionQuery(q, clusterQuery)
	}

	deployments, err = s.deployments.SearchRawDeployments(q)
	return
}
