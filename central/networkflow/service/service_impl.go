package service

import (
	"context"
	"errors"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkflow"
	networkFlowDS "github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/central/networkflow/datastore/entities"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/objects"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkGraph)): {
			"/v1.NetworkGraphService/GetNetworkGraph",
			"/v1.NetworkGraphService/GetExternalNetworkEntities",
		},
		user.With(permissions.Modify(resources.NetworkGraph)): {
			"/v1.NetworkGraphService/CreateExternalNetworkEntity",
			"/v1.NetworkGraphService/DeleteExternalNetworkEntity",
		},
	})

	defaultSince    = -5 * time.Minute
	deploymentSAC   = sac.ForResource(resources.Deployment)
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterFlows networkFlowDS.ClusterDataStore
	entities     entities.EntityDataStore
	deployments  deploymentDS.DataStore
	clusters     clusterDS.DataStore
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

func (s *serviceImpl) GetExternalNetworkEntities(ctx context.Context, request *v1.GetExternalNetworkEntitiesRequest) (*v1.GetExternalNetworkEntitiesResponse, error) {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		return nil, status.Error(codes.Unimplemented, "support for external sources in network graph is not enabled")
	}

	ret, err := s.entities.GetAllEntitiesForCluster(ctx, request.GetClusterId())
	if errors.Is(err, errorhelpers.ErrInvalidArgs) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err != nil {
		return nil, err
	}

	return &v1.GetExternalNetworkEntitiesResponse{
		Entities: ret,
	}, nil
}

func (s *serviceImpl) CreateExternalNetworkEntity(ctx context.Context, request *v1.CreateNetworkEntityRequest) (*storage.NetworkEntity, error) {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		return nil, status.Error(codes.Unimplemented, "support for external sources in network graph is not enabled")
	}

	// If an error is returned here, it means one of the arguments is invalid.
	id, err := sac.NewClusterScopeResourceID(request.GetClusterId(), uuid.NewV4().String())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.validateCluster(id.ClusterID); err != nil {
		return nil, err
	}

	entity := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   id.ToString(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: request.GetEntity(),
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: request.GetClusterId(),
		},
	}

	err = s.entities.UpsertExternalNetworkEntity(ctx, entity)
	if errors.Is(err, errorhelpers.ErrInvalidArgs) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, errorhelpers.ErrAlreadyExists) {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Create a disconnected flow with this entity as source entity so that the node appears in network graph.
	if err := s.addNetworkEntityToGraph(ctx, entity.GetInfo(), entity.GetScope().GetClusterId()); err != nil {
		return nil, err
	}

	// TODO(ROX-5464): Push updated list of cidr blocks to sensor
	return entity, nil
}

func (s *serviceImpl) addNetworkEntityToGraph(ctx context.Context, entity *storage.NetworkEntityInfo, clusterID string) error {
	flowStore, err := s.getFlowStore(ctx, clusterID)
	if err != nil {
		return err
	}
	if flowStore == nil {
		return status.Errorf(codes.NotFound, "no flows found for cluster %s", clusterID)
	}

	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: entity,
			},
		},
	}
	return flowStore.UpsertFlows(ctx, flows, timestamp.Now())
}

func (s *serviceImpl) DeleteExternalNetworkEntity(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		return nil, status.Error(codes.Unimplemented, "support for external sources in network graph is not enabled")
	}

	if err := s.entities.DeleteExternalNetworkEntity(ctx, request.GetId()); err != nil {
		if errors.Is(err, errorhelpers.ErrInvalidArgs) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, err
	}

	id, err := sac.GetClusterScopedResourceID(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Delete the flows to the network entity so that the next get request does not include it.
	// We cannot tolerate central-collector latency for getting the updated view of all the flows.
	if err := s.removeNetworkEntityFromGraph(ctx, request.GetId(), id.ClusterID); err != nil {
		return nil, err
	}

	// TODO(ROX-5464): Push updated list of cidr blocks to sensor.
	return &v1.Empty{}, nil
}

func (s *serviceImpl) removeNetworkEntityFromGraph(ctx context.Context, entityID string, clusterID string) error {
	flowStore, err := s.getFlowStore(ctx, clusterID)
	if err != nil {
		return err
	}
	if flowStore == nil {
		return status.Errorf(codes.NotFound, "no flows found for cluster %s", clusterID)
	}

	keyMatchFunc := func(props *storage.NetworkFlowProperties) bool {
		return props.GetSrcEntity().GetId() == entityID || props.GetDstEntity().GetId() == entityID
	}
	return flowStore.RemoveMatchingFlows(ctx, keyMatchFunc, nil)
}

func (s *serviceImpl) getFlowStore(ctx context.Context, clusterID string) (networkFlowDS.FlowDataStore, error) {
	flowStore, err := s.clusterFlows.GetFlowStore(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if flowStore == nil {
		return nil, status.Errorf(codes.NotFound, "no flows found for cluster %s", clusterID)
	}
	return flowStore, nil
}

func (s *serviceImpl) validateCluster(clusterID string) error {
	// Use elevated context to perform certain cluster validations.
	clusterReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	if exists, err := s.clusters.Exists(clusterReadCtx, clusterID); err != nil {
		return err
	} else if !exists {
		return status.Errorf(codes.NotFound, "cluster %s not found. It may have been deleted", clusterID)
	}
	return nil
}

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.NetworkGraphRequest) (*v1.NetworkGraph, error) {
	if !features.NetworkGraphPorts.Enabled() && request.GetIncludePorts() {
		return nil, status.Error(codes.Unimplemented, "support for ports in network flow graph is not enabled")
	}
	return s.getNetworkGraph(ctx, request, request.GetIncludePorts())
}

func (s *serviceImpl) getNetworkGraph(ctx context.Context, request *v1.NetworkGraphRequest, withListenPorts bool) (*v1.NetworkGraph, error) {
	if request.GetClusterId() == "" {
		return nil, status.Error(codes.InvalidArgument, "cluster ID must be specified")
	}

	since := request.GetSince()
	if since == nil {
		var err error
		since, err = types.TimestampProto(time.Now().Add(defaultSince))
		if err != nil {
			utils.Should(err)
		}
	}

	// Get the deployments we want to check connectivity between.
	deployments, err := s.getDeployments(ctx, request.GetClusterId(), request.GetQuery())
	if err != nil {
		return nil, err
	}
	if len(deployments) == 0 {
		return &v1.NetworkGraph{}, nil
	}

	// Build a possibly reduced map of only those deployments for which we can see network flows.
	networkFlowsChecker := networkGraphSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ClusterID(request.GetClusterId())
	filteredSlice, err := sac.FilterSliceReflect(ctx, networkFlowsChecker, deployments, func(deployment *storage.ListDeployment) sac.ScopePredicate {
		return sac.ScopeSuffix{sac.NamespaceScopeKey(deployment.GetNamespace())}
	})
	if err != nil {
		return nil, err
	}
	deploymentsWithFlows := objects.ListDeploymentsMapByID(filteredSlice.([]*storage.ListDeployment))

	deploymentsMap := objects.ListDeploymentsMapByID(deployments)

	// We can see all relevant flows if no deployments were filtered out in the previous step.
	canSeeAllFlows := len(deploymentsMap) == len(deploymentsWithFlows)

	builder := newFlowGraphBuilder()
	builder.AddDeployments(deployments)

	// Temporarily elevate permissions to obtain all network flows in cluster.
	networkGraphGenElevatedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(request.GetClusterId())))

	flowStore, err := s.clusterFlows.GetFlowStore(networkGraphGenElevatedCtx, request.GetClusterId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not obtain flows for cluster %s: %v", request.GetClusterId(), err)
	} else if flowStore == nil {
		return nil, status.Errorf(codes.NotFound, "no flows found for cluster %s", request.GetClusterId())
	}

	// canSeeAllDeploymentsInCluster helps us to determine whether we have to handle masked deployments at all or not.
	canSeeAllDeploymentsInCluster, err := deploymentSAC.ReadAllowed(ctx, sac.ClusterScopeKey(request.GetClusterId()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not check permissions: %v", err)
	}

	var pred func(*storage.NetworkFlowProperties) bool
	if request.GetQuery() != "" || !canSeeAllDeploymentsInCluster || !canSeeAllFlows {
		pred = func(props *storage.NetworkFlowProperties) bool {
			srcEnt := props.GetSrcEntity()
			dstEnt := props.GetDstEntity()

			if !withListenPorts && dstEnt.GetType() == storage.NetworkEntityInfo_LISTEN_ENDPOINT {
				return false
			}

			// If we cannot see all flows of all relevant deployments, filter out flows where we can't see network flows
			// on both ends (this takes care of the relevant network flow filtering).
			if !canSeeAllFlows {
				if (srcEnt.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || deploymentsWithFlows[srcEnt.GetId()] == nil) &&
					(dstEnt.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || deploymentsWithFlows[dstEnt.GetId()] == nil) {
					return false
				}
			}

			for _, entity := range []*storage.NetworkEntityInfo{props.GetSrcEntity(), props.GetDstEntity()} {
				if entity.GetType() == storage.NetworkEntityInfo_DEPLOYMENT {
					if canSeeAllDeploymentsInCluster && deploymentsMap[entity.GetId()] == nil {
						// We can see all deployments in the cluster, so any deployment not in the map was simply not
						// selected by the query -> skip flow
						return false
					} else if !canSeeAllDeploymentsInCluster && deploymentsMap[entity.GetId()] != nil {
						// We can't see all deployments in the cluster, so any flow with at least one endpoint in the
						// map might be relevant (if the other endpoint is not in the map, it could still be masked).
						return true
					}
				}
			}

			// If canSeeAllDeploymentsInCluster is true, we *exclude* flows above, otherwise we *include* them. Return
			// the respective default for each action (including anything that's not excluded and vice versa).
			return canSeeAllDeploymentsInCluster
		}
	}

	flows, _, err := flowStore.GetMatchingFlows(networkGraphGenElevatedCtx, pred, since)
	if err != nil {
		return nil, err
	}

	flows, missingInfoFlows := networkflow.UpdateFlowsWithDeployments(flows, deploymentsMap)

	builder.AddFlows(flows)

	filteredFlows, maskedDeployments, err := filterFlowsAndMaskScopeAlienDeployments(ctx, request.GetClusterId(), missingInfoFlows, deploymentsMap, s.deployments, canSeeAllDeploymentsInCluster)
	if err != nil {
		return nil, err
	}

	builder.AddDeployments(maskedDeployments)
	builder.AddFlows(filteredFlows)

	return builder.Build(), nil
}

func filterFlowsAndMaskScopeAlienDeployments(ctx context.Context, clusterID string, flows []*storage.NetworkFlow, deploymentsMap map[string]*storage.ListDeployment, deploymentDS deploymentDS.DataStore, allDeploymentsVisible bool) (filtered []*storage.NetworkFlow, maskedDeployments []*storage.ListDeployment, err error) {
	isVisibleDeployment := func(string) bool { return true }
	if !allDeploymentsVisible {
		// Find out which deployments we *can* see.
		visibleDeployments, err := deploymentDS.Search(ctx, search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
		if err != nil {
			return nil, nil, err
		}
		isVisibleDeployment = search.ResultsToIDSet(visibleDeployments).Contains
	}

	// Pass 1: Find deployments for which we are missing data (deleted or invisible).
	filtered = flows[:0]

	missingDeploymentIDs := set.NewStringSet()
	for _, flow := range flows {
		srcEnt := flow.GetProps().GetSrcEntity()
		dstEnt := flow.GetProps().GetDstEntity()
		// Skip all flows with BOTH endpoints not in the set (treating non-deployment entities as "not in the set").
		if (srcEnt.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || deploymentsMap[srcEnt.GetId()] == nil) &&
			(dstEnt.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || deploymentsMap[dstEnt.GetId()] == nil) {
			continue
		}

		// Skip flows where one of the endpoints is not in the set but visible
		if srcEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && deploymentsMap[srcEnt.GetId()] == nil {
			if isVisibleDeployment(srcEnt.GetId()) {
				continue
			}
			missingDeploymentIDs.Add(srcEnt.GetId())
		}
		if dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && deploymentsMap[dstEnt.GetId()] == nil {
			if isVisibleDeployment(dstEnt.GetId()) {
				continue
			}
			missingDeploymentIDs.Add(dstEnt.GetId())
		}

		filtered = append(filtered, flow)
	}

	flows = filtered
	filtered = flows[:0]

	allDeploymentsReadCtx := sac.WithGlobalAccessScopeChecker(
		ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
			sac.ClusterScopeKeys(clusterID)))

	existingButInvisibleDeploymentsList, err := deploymentDS.SearchListDeployments(allDeploymentsReadCtx,
		search.NewQueryBuilder().AddDocIDSet(missingDeploymentIDs).ProtoQuery())
	if err != nil {
		return nil, nil, err
	}
	existingButInvisibleDeploymentsMap := objects.ListDeploymentsMapByID(existingButInvisibleDeploymentsList)

	// Step 2: Mask deployments a user is not allowed to see.
	masker := newFlowGraphMasker()

	for _, flow := range flows {
		skipFlow := false
		entities := []*storage.NetworkEntityInfo{flow.GetProps().GetSrcEntity(), flow.GetProps().GetDstEntity()}
		for _, entity := range entities {
			if entity.GetType() != storage.NetworkEntityInfo_DEPLOYMENT || deploymentsMap[entity.GetId()] != nil {
				// no masking or skipping required for deployments which are in the set.
				continue
			}

			invisibleDeployment := existingButInvisibleDeploymentsMap[entity.GetId()]
			if invisibleDeployment == nil {
				skipFlow = true // deployment has been deleted
				break
			}

			// To avoid information leak we always show all masked neighbors
			maskedDeployment := masker.GetMaskedDeployment(invisibleDeployment)
			*entity = *networkflow.EntityForDeployment(maskedDeployment)
		}
		if skipFlow {
			continue
		}
		filtered = append(filtered, flow)
	}

	return filtered, masker.GetMaskedDeployments(), nil
}

func (s *serviceImpl) getDeployments(ctx context.Context, clusterID string, query string) (deployments []*storage.ListDeployment, err error) {
	clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()

	q := clusterQuery
	if query != "" {
		q, err = search.ParseQuery(query)
		if err != nil {
			return
		}
		q = search.ConjunctionQuery(q, clusterQuery)
	}

	deployments, err = s.deployments.SearchListDeployments(ctx, q)
	return
}
