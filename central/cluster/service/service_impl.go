package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/cluster/index/mappings"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/probesources"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/timeutil"
	"google.golang.org/grpc"
)

var (
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Cluster)): {
			"/v1.ClustersService/GetCluster",
			"/v1.ClustersService/GetKernelSupportAvailable",
			"/v1.ClustersService/GetClusterDefaultValues",
		},
		user.With(permissions.Modify(resources.Cluster)): {
			"/v1.ClustersService/PostCluster",
			"/v1.ClustersService/PutCluster",
			"/v1.ClustersService/DeleteCluster",
		},
		allow.Anonymous(): {
			"/v1.ClustersService/GetClusters",
		},
	}))

	log = logging.LoggerForModule()
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	v1.UnimplementedClustersServiceServer

	datastore          datastore.DataStore
	riskManager        manager.Manager
	probeSources       probesources.ProbeSources
	sysConfigDatastore configDatastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClustersServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterClustersServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// PostCluster creates a new cluster.
func (s *serviceImpl) PostCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id field should be empty when posting a new cluster")
	}
	id, err := s.datastore.AddCluster(ctx, request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	return s.getCluster(ctx, request.GetId())
}

// PutCluster updates an existing cluster.
func (s *serviceImpl) PutCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id must be provided")
	}
	err := s.datastore.UpdateCluster(ctx, request)
	if err != nil {
		return nil, err
	}
	return s.getCluster(ctx, request.GetId())
}

// GetCluster returns the specified cluster.
func (s *serviceImpl) GetCluster(ctx context.Context, request *v1.ResourceByID) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id must be provided")
	}
	return s.getCluster(ctx, request.GetId())
}

func (s *serviceImpl) getCluster(ctx context.Context, id string) (*v1.ClusterResponse, error) {
	cluster, ok, err := s.datastore.GetCluster(ctx, id)
	if err != nil {
		return nil, errors.Errorf("Could not get cluster: %s", err)
	}
	if !ok {
		return nil, errors.Wrap(errox.NotFound, "Not found")
	}

	if !features.DecommissionedClusterRetention.Enabled() {
		return &v1.ClusterResponse{
			Cluster: cluster,
		}, nil
	}

	clusterRetentionInfo, err := s.getClusterRetentionInfo(ctx, cluster)
	if err != nil {
		return nil, err
	}

	return &v1.ClusterResponse{
		Cluster:              cluster,
		ClusterRetentionInfo: clusterRetentionInfo,
	}, nil
}

func (s *serviceImpl) getClusterRetentionInfo(ctx context.Context, cluster *storage.Cluster) (*v1.DecommissionedClusterRetentionInfo, error) {
	if cluster.GetHealthStatus().GetSensorHealthStatus() != storage.ClusterHealthStatus_UNHEALTHY {
		return nil, nil
	}

	sysConfig, err := s.sysConfigDatastore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	clusterRetentionConfig := sysConfig.GetPrivateConfig().GetDecommissionedClusterRetention()
	if clusterRetentionConfig == nil || clusterRetentionConfig.GetRetentionDurationDays() == 0 {
		// If retention is disabled, there is no "days remaining" calculation to be done.
		return nil, nil
	}

	if sliceutils.MapsIntersect(clusterRetentionConfig.GetIgnoreClusterLabels(), cluster.GetLabels()) {
		return &v1.DecommissionedClusterRetentionInfo{
			RetentionInfo: &v1.DecommissionedClusterRetentionInfo_IsExcluded{
				IsExcluded: true,
			},
		}, nil
	}

	timeNow := time.Now()
	retentionDays := clusterRetentionConfig.GetRetentionDurationDays()

	configCreateTime, err := types.TimestampFromProto(clusterRetentionConfig.GetCreatedAt())
	if err != nil {
		return nil, err
	}

	lastContactTime, err := types.TimestampFromProto(cluster.GetHealthStatus().GetLastContact())
	if err != nil {
		return nil, err
	}

	daysRemaining := int32(0)
	if lastContactTime.Before(configCreateTime) {
		daysRemaining = retentionDays - int32(timeutil.TimeDiffDays(timeNow, configCreateTime))
	} else {
		daysRemaining = retentionDays - int32(timeutil.TimeDiffDays(timeNow, lastContactTime))
	}

	return &v1.DecommissionedClusterRetentionInfo{
		RetentionInfo: &v1.DecommissionedClusterRetentionInfo_DaysUntilDeletion{
			DaysUntilDeletion: daysRemaining,
		},
	}, nil
}

// GetClusters returns the currently defined clusters.
func (s *serviceImpl) GetClusters(ctx context.Context, req *v1.GetClustersRequest) (*v1.ClustersList, error) {
	q, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "invalid query %q: %v", req.GetQuery(), err)
	}

	clusters, err := s.datastore.SearchRawClusters(ctx, q)
	if err != nil {
		return nil, err
	}

	clustersMap := make(map[string]*storage.Cluster, 0)
	for ix := range clusters {
		c := clusters[ix]
		clustersMap[c.GetId()] = c
	}

	// Extend cluster list with Clusters belonging to any scope linked to a scoped user permission
	hasResourceWithFullScope := false
	targetClusterIDs := make(map[string]struct{}, 0)
	for _, p := range resources.AllResourcesViewPermissions() {
		if p.Resource.GetScope() == permissions.GlobalScope {
			continue
		}
		scope, err := sac.ForResource(p.Resource).ScopeChecker(ctx, p.Access).EffectiveAccessScope(p)
		if err != nil {
			continue
		}
		if scope == nil {
			continue
		}
		if scope.State == effectiveaccessscope.Included {
			hasResourceWithFullScope = true
			break
		}
		for _, id := range scope.GetClusterIDs() {
			log.Infof("Resource %q has access to cluster %q", p.Resource.GetResource(), id)
			targetClusterIDs[id] = struct{}{}
		}
	}
	var elevatedCtx context.Context
	if hasResourceWithFullScope {
		elevatedCtx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
			),
		)
	} else {
		clusterIDs := make([]string, 0, len(targetClusterIDs))
		for k := range targetClusterIDs {
			clusterIDs = append(clusterIDs, k)
		}
		elevatedCtx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys(clusterIDs...),
			),
		)
	}
	extraClusterQuery := search.NewQueryBuilder().
		AddStringsHighlighted("Cluster", search.WildcardString).ProtoQuery()
	extraClusterResults, err := s.datastore.Search(elevatedCtx, extraClusterQuery)
	if err != nil {
		// In case of error, do not display any extra cluster info
		extraClusterResults = nil
	}
	extraClusters := make([]*storage.Cluster, 0, len(extraClusterResults))
	var clusterOptions search.OptionsMap
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		clusterOptions = schema.ClustersSchema.OptionsMap
	} else {
		clusterOptions = mappings.OptionsMap
	}
	for _, r := range extraClusterResults {
		if _, found := clustersMap[r.ID]; found {
			continue
		}
		clusterName := fmt.Sprintf("Cluster with id %q", r.ID)
		matches := r.Matches[clusterOptions.MustGet(search.Cluster.String()).GetFieldPath()]
		for _, m := range matches {
			if len(m) > 0 {
				clusterName = m
				break
			}
		}
		extraClusters = append(extraClusters, &storage.Cluster{
			Id:   r.ID,
			Name: clusterName,
		})
	}

	if !features.DecommissionedClusterRetention.Enabled() {
		clusters = append(clusters, extraClusters...)
		return &v1.ClustersList{
			Clusters: clusters,
		}, nil
	}

	clusterIDToRetentionInfoMap, err := s.getClusterIDToRetentionInfoMap(ctx, clusters)
	if err != nil {
		return nil, err
	}

	clusters = append(clusters, extraClusters...)
	return &v1.ClustersList{
		Clusters:                 clusters,
		ClusterIdToRetentionInfo: clusterIDToRetentionInfoMap,
	}, nil
}

func (s *serviceImpl) getClusterIDToRetentionInfoMap(
	ctx context.Context,
	clusters []*storage.Cluster,
) (map[string]*v1.DecommissionedClusterRetentionInfo, error) {
	result := make(map[string]*v1.DecommissionedClusterRetentionInfo)

	for _, cluster := range clusters {
		retentionInfo, err := s.getClusterRetentionInfo(ctx, cluster)
		if err != nil {
			return nil, err
		}
		if retentionInfo != nil {
			result[cluster.GetId()] = retentionInfo
		}
	}

	return result, nil
}

// DeleteCluster removes a cluster
func (s *serviceImpl) DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Request must have a id")
	}
	if err := s.datastore.RemoveCluster(ctx, request.GetId(), nil); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// Deprecated: Use GetClusterDefaultValues instead.
func (s *serviceImpl) GetKernelSupportAvailable(ctx context.Context, _ *v1.Empty) (*v1.KernelSupportAvailableResponse, error) {
	anyAvailable, err := s.probeSources.AnyAvailable(ctx)
	if err != nil {
		return nil, err
	}
	result := &v1.KernelSupportAvailableResponse{
		KernelSupportAvailable: anyAvailable,
	}
	return result, nil
}

func (s *serviceImpl) GetClusterDefaultValues(ctx context.Context, _ *v1.Empty) (*v1.ClusterDefaultsResponse, error) {
	kernelSupport, err := s.probeSources.AnyAvailable(ctx)
	if err != nil {
		return nil, err
	}
	flavor := defaults.GetImageFlavorFromEnv()
	defaults := &v1.ClusterDefaultsResponse{
		MainImageRepository:      flavor.MainImageNoTag(),
		CollectorImageRepository: flavor.CollectorFullImageNoTag(),
		KernelSupportAvailable:   kernelSupport,
	}
	return defaults, nil
}
