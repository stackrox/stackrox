package service

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDS "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/aggregator"
	"github.com/stackrox/stackrox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/mappings"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	networkFlowDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
	"github.com/stackrox/stackrox/pkg/networkgraph"
	"github.com/stackrox/stackrox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/stackrox/pkg/networkgraph/tree"
	"github.com/stackrox/stackrox/pkg/objects"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/predicate"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/utils"
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
			"/v1.NetworkGraphService/PatchExternalNetworkEntity",
		},
		user.With(permissions.View(resources.NetworkGraphConfig)): {
			"/v1.NetworkGraphService/GetNetworkGraphConfig",
		},
		user.With(permissions.Modify(resources.NetworkGraphConfig)): {
			"/v1.NetworkGraphService/PutNetworkGraphConfig",
		},
	})

	defaultSince         = -5 * time.Minute
	networkGraphSAC      = sac.ForResource(resources.NetworkGraph)
	netEntityPredFactory = predicate.NewFactory("networkEntity", &storage.NetworkEntity{})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterFlows   networkFlowDS.ClusterDataStore
	entities       networkEntityDS.EntityDataStore
	networkTreeMgr networktree.Manager
	deployments    deploymentDS.DataStore
	clusters       clusterDS.DataStore
	graphConfig    datastore.DataStore
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
	query, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	query, _ = search.FilterQueryWithMap(query, mappings.OptionsMap)
	pred, err := netEntityPredFactory.GeneratePredicate(query)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "failed to parse query %q: %v", query.String(), err.Error())
	}

	ret, err := s.entities.GetAllMatchingEntities(ctx, func(entity *storage.NetworkEntity) bool {
		// Do not respect the graph configuration.
		if entity.GetScope().GetClusterId() == "" || entity.GetScope().GetClusterId() == request.GetClusterId() {
			return pred.Matches(entity)
		}
		return false
	})
	if err != nil {
		return nil, err
	}

	return &v1.GetExternalNetworkEntitiesResponse{
		Entities: ret,
	}, nil
}

func (s *serviceImpl) CreateExternalNetworkEntity(ctx context.Context, request *v1.CreateNetworkEntityRequest) (*storage.NetworkEntity, error) {
	// An error here implies one of the arguments is invalid.
	id, err := externalsrcs.NewClusterScopedID(request.GetClusterId(), request.GetEntity().GetCidr())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	if err := s.validateCluster(request.GetClusterId()); err != nil {
		return nil, err
	}

	entity := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   id.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: request.GetEntity(),
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: request.GetClusterId(),
		},
	}

	err = s.entities.CreateExternalNetworkEntity(ctx, entity, false)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *serviceImpl) DeleteExternalNetworkEntity(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if _, err := s.getEntityAndValidateMutable(ctx, request.GetId()); err != nil {
		return nil, err
	}

	if err := s.entities.DeleteExternalNetworkEntity(ctx, request.GetId()); err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) PatchExternalNetworkEntity(ctx context.Context, request *v1.PatchNetworkEntityRequest) (*storage.NetworkEntity, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "network entity ID must be specified")
	}

	id := request.GetId()
	// Disallow updates to default networks through API.
	entity, err := s.getEntityAndValidateMutable(ctx, id)
	if err != nil {
		return nil, err
	}
	if entity.GetInfo().GetExternalSource() == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot update network entity %q since the stored entity is invalid. Please delete and recreate.", id)
	}

	entity.Info.GetExternalSource().Name = request.GetName()

	if err := s.entities.UpdateExternalNetworkEntity(ctx, entity, false); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *serviceImpl) getEntityAndValidateMutable(ctx context.Context, id string) (*storage.NetworkEntity, error) {
	entity, found, err := s.entities.GetEntity(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "network entity %s not found", id)
	}
	if entity.GetInfo().GetExternalSource().GetDefault() {
		return nil, status.Error(codes.PermissionDenied, "StackRox-generated network entities are immutable")
	}
	return entity, nil
}

// GetNetworkGraphConfig updates Central's network graph config
func (s *serviceImpl) GetNetworkGraphConfig(ctx context.Context, _ *v1.Empty) (*storage.NetworkGraphConfig, error) {
	cfg, err := s.graphConfig.GetNetworkGraphConfig(ctx)
	if err != nil {
		return nil, errors.Errorf("could not obtain network graph configuration: %v", err)
	}
	return cfg, nil
}

// PutNetworkGraphConfig updates Central's network graph config
func (s *serviceImpl) PutNetworkGraphConfig(ctx context.Context, req *v1.PutNetworkGraphConfigRequest) (*storage.NetworkGraphConfig, error) {
	if req.GetConfig() == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "network graph config must be specified")
	}

	if err := s.graphConfig.UpdateNetworkGraphConfig(ctx, req.GetConfig()); err != nil {
		return nil, errors.Errorf("could not update network graph configuration: %v", err)
	}
	return req.GetConfig(), nil
}

func (s *serviceImpl) getFlowStore(ctx context.Context, clusterID string) (networkFlowDS.FlowDataStore, error) {
	flowStore, err := s.clusterFlows.GetFlowStore(ctx, clusterID)
	if err != nil {
		return nil, errors.Errorf("could not obtain flows for cluster %s: %v", clusterID, err)
	}
	if flowStore == nil {
		return nil, errors.Wrapf(errox.NotFound, "no flows found for cluster %s", clusterID)
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
		return errors.Wrapf(errox.NotFound, "cluster %s not found. It may have been deleted", clusterID)
	}
	return nil
}

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.NetworkGraphRequest) (*v1.NetworkGraph, error) {
	return s.getNetworkGraph(ctx, request, request.GetIncludePorts())
}

func (s *serviceImpl) getNetworkGraph(ctx context.Context, request *v1.NetworkGraphRequest, withListenPorts bool) (*v1.NetworkGraph, error) {
	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "cluster ID must be specified")
	}

	requestClone := request.Clone()
	if requestClone.GetSince() == nil {
		since, err := types.TimestampProto(time.Now().Add(defaultSince))
		if err != nil {
			utils.Should(err)
		}
		requestClone.Since = since
	}

	deploymentQuery, scopeQuery, err := networkgraph.GetFilterAndScopeQueries(request.GetClusterId(), requestClone.GetQuery(), requestClone.GetScope())
	if err != nil {
		return nil, err
	}

	count, err := s.deployments.Count(ctx, deploymentQuery)
	if err != nil {
		return nil, err
	}

	if count > maxNumberOfDeploymentsInGraphEnv.IntegerSetting() {
		log.Warnf("Number of deployments is too high to be rendered in Network Graph: %d", count)
		return nil, errors.Errorf(
			"number of deployments (%d) exceeds maximum allowed for Network Graph: %d",
			count,
			maxNumberOfDeploymentsInGraphEnv.IntegerSetting(),
		)
	}

	deployments, err := s.deployments.SearchListDeployments(ctx, deploymentQuery)
	if err != nil {
		return nil, err
	}

	// External sources should be shown only wrt to deployments.
	if len(deployments) == 0 {
		return &v1.NetworkGraph{}, nil
	}

	builder := newFlowGraphBuilder()
	builder.AddDeployments(deployments)

	if err := s.addDeploymentFlowsToGraph(ctx, requestClone, scopeQuery, withListenPorts, builder, deployments); err != nil {
		return nil, err
	}

	depSet := set.NewStringSet()
	for _, deployment := range deployments {
		depSet.Add(deployment.GetId())
	}

	graph := builder.Build()
	for _, node := range graph.GetNodes() {
		if depSet.Contains(node.GetEntity().GetId()) {
			node.QueryMatch = true
		}
	}
	return graph, nil
}

func (s *serviceImpl) addDeploymentFlowsToGraph(
	ctx context.Context,
	request *v1.NetworkGraphRequest,
	scopeQuery *v1.Query,
	withListenPorts bool,
	graphBuilder *flowGraphBuilder,
	deployments []*storage.ListDeployment,
) error {
	// Build a possibly reduced map of only those deployments for which we can see network flows.
	networkFlowsChecker := networkGraphSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ClusterID(request.GetClusterId())
	filteredSlice, err := sac.FilterSliceReflect(ctx, networkFlowsChecker, deployments, func(deployment *storage.ListDeployment) sac.ScopePredicate {
		return sac.ScopeSuffix{sac.NamespaceScopeKey(deployment.GetNamespace())}
	})
	if err != nil {
		return err
	}
	deploymentsWithFlows := objects.ListDeploymentsMapByID(filteredSlice.([]*storage.ListDeployment))
	deploymentsMap := objects.ListDeploymentsMapByID(deployments)

	// We can see all relevant flows if no deployments were filtered out in the previous step.
	canSeeAllFlows := len(deploymentsMap) == len(deploymentsWithFlows)

	// Temporarily elevate permissions to obtain all network flows in cluster.
	networkGraphGenElevatedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(request.GetClusterId())))

	flowStore, err := s.getFlowStore(networkGraphGenElevatedCtx, request.GetClusterId())
	if err != nil {
		return err
	}

	var pred func(*storage.NetworkFlowProperties) bool
	if request.GetQuery() != "" || !canSeeAllFlows {
		pred = func(props *storage.NetworkFlowProperties) bool {
			srcEnt := props.GetSrcEntity()
			dstEnt := props.GetDstEntity()

			// Exclude all flows having both external endpoints. Although if one endpoint is an invisible external source,
			// we still want to show the flow  given that the other endpoint is visible, however, attribute it to INTERNET.
			if networkgraph.AllExternal(srcEnt, dstEnt) {
				return false
			}

			if !withListenPorts && dstEnt.GetType() == storage.NetworkEntityInfo_LISTEN_ENDPOINT {
				return false
			}

			// If we cannot see all flows of all relevant deployments, filter out flows where we can't see network flows
			// on both ends (this takes care of the relevant network flow filtering).
			if !canSeeAllFlows {
				if !networkgraph.AnyDeploymentInFilter(srcEnt, dstEnt, deploymentsWithFlows) {
					return false
				}
			}

			return networkgraph.AnyDeploymentInFilter(srcEnt, dstEnt, deploymentsMap)
		}
	}

	flows, _, err := flowStore.GetMatchingFlows(networkGraphGenElevatedCtx, pred, request.GetSince())
	if err != nil {
		return err
	}

	networkTree := tree.NewMultiNetworkTree(
		s.networkTreeMgr.GetReadOnlyNetworkTree(ctx, request.GetClusterId()),
		s.networkTreeMgr.GetDefaultNetworkTree(ctx),
	)

	// Aggregate all external conns into supernet conns for which external entities do not exists (as a result of deletion).
	aggr, err := aggregator.NewSubnetToSupernetConnAggregator(networkTree)
	utils.Should(err)
	flows = aggr.Aggregate(flows)

	flows, missingInfoFlows := networkgraph.UpdateFlowsWithEntityDesc(flows, deploymentsMap,
		func(id string) *storage.NetworkEntityInfo {
			if networkTree == nil {
				return nil
			}
			return networkTree.Get(id)
		},
	)

	// Aggregate all external flows by node names to control the number of external nodes.
	flows = aggregator.NewDuplicateNameExtSrcConnAggregator().Aggregate(flows)
	missingInfoFlows = aggregator.NewDuplicateNameExtSrcConnAggregator().Aggregate(missingInfoFlows)
	graphBuilder.AddFlows(flows)

	filteredFlows, visibleNeighbors, maskedDeployments, err := filterFlowsAndMaskScopeAlienDeployments(ctx,
		request.GetClusterId(), scopeQuery, missingInfoFlows, deploymentsMap, s.deployments)
	if err != nil {
		return err
	}
	graphBuilder.AddDeployments(visibleNeighbors)
	graphBuilder.AddDeployments(maskedDeployments)
	graphBuilder.AddFlows(filteredFlows)
	return nil
}

func filterFlowsAndMaskScopeAlienDeployments(
	ctx context.Context,
	clusterID string,
	scopeQuery *v1.Query,
	flows []*storage.NetworkFlow,
	deploymentsMap map[string]*storage.ListDeployment,
	deploymentDS deploymentDS.DataStore,
) (filtered []*storage.NetworkFlow, visibleNeighbors []*storage.ListDeployment, maskedDeployments []*storage.ListDeployment, err error) {
	// Find out which deployments we *can* see.
	results, err := deploymentDS.SearchListDeployments(ctx, scopeQuery)
	if err != nil {
		return nil, nil, nil, err
	}
	visibleDeployments := objects.ListDeploymentsMapByID(results)

	// Pass 1: Find deployments for which we are missing data (deleted or invisible).
	filtered = flows[:0]

	visibleNeighboringDeployments := set.NewStringSet()
	missingDeploymentIDs := set.NewStringSet()
	for _, flow := range flows {
		srcEnt, dstEnt := flow.GetProps().GetSrcEntity(), flow.GetProps().GetDstEntity()
		// Skip all flows with BOTH endpoints not in the set.
		if !networkgraph.AnyDeploymentInFilter(srcEnt, dstEnt, deploymentsMap) {
			continue
		}

		// Determine if neighbor is visible or not.
		if srcEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && deploymentsMap[srcEnt.GetId()] == nil {
			if visibleDeployments[srcEnt.GetId()] == nil {
				missingDeploymentIDs.Add(srcEnt.GetId())
			} else {
				visibleNeighboringDeployments.Add(srcEnt.GetId())
			}
		}
		if dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && deploymentsMap[dstEnt.GetId()] == nil {
			if visibleDeployments[dstEnt.GetId()] == nil {
				missingDeploymentIDs.Add(dstEnt.GetId())
			} else {
				visibleNeighboringDeployments.Add(dstEnt.GetId())
			}
		}
		filtered = append(filtered, flow)
	}

	flows = filtered
	filtered = flows[:0]

	var existingButInvisibleDeploymentsMap map[string]*storage.ListDeployment
	if len(missingDeploymentIDs) > 0 {
		allDeploymentsReadCtx := sac.WithGlobalAccessScopeChecker(
			ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Deployment),
				sac.ClusterScopeKeys(clusterID)))

		existingButInvisibleDeploymentsList, err := deploymentDS.SearchListDeployments(allDeploymentsReadCtx,
			search.ConjunctionQuery(
				scopeQuery,
				search.NewQueryBuilder().AddDocIDSet(missingDeploymentIDs).ProtoQuery()),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		existingButInvisibleDeploymentsMap = objects.ListDeploymentsMapByID(existingButInvisibleDeploymentsList)
	}

	// Step 2: Mask deployments a user is not allowed to see.
	masker := newFlowGraphMasker()

	for _, flow := range flows {
		skipFlow := false
		entities := []*storage.NetworkEntityInfo{flow.GetProps().GetSrcEntity(), flow.GetProps().GetDstEntity()}
		for _, entity := range entities {
			// no masking or skipping required for non-deployment type entities.
			if entity.GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
				continue
			}

			// no masking or skipping required for deployments already in the set.
			if deploymentsMap[entity.GetId()] != nil {
				continue
			}

			// no masking or skipping required for neighboring deployments.
			if visibleNeighboringDeployments.Contains(entity.GetId()) {
				continue
			}

			invisibleDeployment := existingButInvisibleDeploymentsMap[entity.GetId()]
			if invisibleDeployment == nil {
				skipFlow = true // deployment has been deleted or does not satisfy scope.
				break
			}

			// To avoid information leak we always show all masked neighbors
			maskedDeployment := masker.GetMaskedDeployment(invisibleDeployment)
			*entity = *networkgraph.NetworkEntityForDeployment(maskedDeployment)
		}
		if skipFlow {
			continue
		}
		filtered = append(filtered, flow)
	}

	for _, visibleDeployment := range visibleDeployments {
		if visibleNeighboringDeployments.Contains(visibleDeployment.GetId()) {
			visibleNeighbors = append(visibleNeighbors, visibleDeployment)
		}
	}
	return filtered, visibleNeighbors, masker.GetMaskedDeployments(), nil
}
