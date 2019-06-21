package service

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
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
	defaultSince    = -5 * time.Minute
	deploymentSAC   = sac.ForResource(resources.Deployment)
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterFlows nfDS.ClusterDataStore
	deployments  dDS.DataStore
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

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.NetworkGraphRequest) (*v1.NetworkGraph, error) {
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
	deployments, err := s.getDeployments(ctx, request.GetClusterId(), request.GetQuery())

	if err != nil {
		return nil, err
	}

	builder := newFlowGraphBuilder()
	builder.AddDeployments(deployments)

	flowStore := s.clusterFlows.GetFlowStore(ctx, request.GetClusterId())

	if flowStore == nil {
		return nil, status.Errorf(codes.NotFound, "no flows found for cluster %s", request.GetClusterId())
	}

	// Temporarily elevate permissions to obtain all network flows in cluster.
	networkGraphGenElevatedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	flows, _, err := flowStore.GetAllFlows(networkGraphGenElevatedCtx, since)
	if err != nil {
		return nil, err
	}

	// compute edges

	// Filter by deployments, and then by time.
	filteredFlows, maskedDeployments, err := FilterFlowsAndMaskScopeAlienDeployments(ctx, request.GetClusterId(), flows, deployments)
	if err != nil {
		return nil, err
	}

	builder.AddDeployments(maskedDeployments)
	builder.AddFlows(filteredFlows)

	return builder.Build(), nil
}

// FilterFlowsAndMaskScopeAlienDeployments filters incoming flows based on access scope and masks all node/deployments outside scope
func FilterFlowsAndMaskScopeAlienDeployments(ctx context.Context, clusterID string, flows []*storage.NetworkFlow, deployments []*storage.Deployment) (filtered []*storage.NetworkFlow, maskedDeployments []*storage.Deployment, err error) {
	masker := newFlowGraphMasker()
	filtered = flows[:0]
	deploymentIDMap := make(map[string]bool)
	for _, d := range deployments {
		deploymentIDMap[d.Id] = true
	}

	for _, flow := range flows {
		srcEnt := flow.GetProps().GetSrcEntity()
		dstEnt := flow.GetProps().GetDstEntity()
		if (srcEnt.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || !deploymentIDMap[srcEnt.GetId()]) &&
			(dstEnt.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || !deploymentIDMap[dstEnt.GetId()]) {
			continue
		}
		sc := networkGraphSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS)
		allowed, err := AccessAllowed(ctx, sc, clusterID, flow)
		if err != nil {
			return nil, nil, err
		}
		if !allowed {
			continue
		}
		filtered = append(filtered, flow)
	}

	flows = filtered
	filtered = flows[:0]
	for _, flow := range flows {
		skipFlow := false
		entities := []*storage.NetworkEntityInfo{flow.GetProps().GetSrcEntity(), flow.GetProps().GetDstEntity()}
		for _, entity := range entities {
			if entity.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || deploymentIDMap[entity.GetId()] {
				continue
			}
			// Deployment is visible but not in the map, which means it was not selected by the query
			scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(clusterID), sac.NamespaceScopeKey(entity.GetDeployment().GetNamespace())}
			ok, err := deploymentSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, scopeKeys...).Allowed(ctx)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				skipFlow = true
				break
			}
			// To avoid information leak we always show all masked neighbors
			maskedDeployment := masker.GetMaskedDeploymentForCluster(entity.GetDeployment().GetCluster(), entity.GetDeployment().GetNamespace(), entity.GetId())
			maskedDeployments = append(maskedDeployments, maskedDeployment)
			*entity = *masker.GetFlowEntityForDeployment(maskedDeployment)
		}
		if skipFlow {
			continue
		}
		filtered = append(filtered, flow)
	}

	return filtered, maskedDeployments, nil
}

func (s *serviceImpl) getDeployments(ctx context.Context, clusterID string, query string) (deployments []*storage.Deployment, err error) {
	clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()

	q := clusterQuery
	if query != "" {
		q, err = search.ParseRawQuery(query)
		if err != nil {
			return
		}
		q = search.ConjunctionQuery(q, clusterQuery)
	}

	deployments, err = s.deployments.SearchRawDeployments(ctx, q)
	return
}

// AccessAllowed checks if access to network flow should be allowed
func AccessAllowed(ctx context.Context, scoperChecker sac.ScopeChecker, clusterID string, flow *storage.NetworkFlow) (bool, error) {
	srcEnt := flow.GetProps().GetSrcEntity()
	dstEnt := flow.GetProps().GetDstEntity()

	var scopeKeys [][]sac.ScopeKey
	if srcEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT {
		scopeKeys = append(scopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(clusterID), sac.NamespaceScopeKey(srcEnt.GetDeployment().GetNamespace())})
	}
	if dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT {
		scopeKeys = append(scopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(clusterID), sac.NamespaceScopeKey(dstEnt.GetDeployment().GetNamespace())})
	}

	if len(scopeKeys) == 0 {
		return true, nil
	}

	return scoperChecker.AnyAllowed(ctx, scopeKeys)
}
