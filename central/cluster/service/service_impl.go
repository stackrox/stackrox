package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/probesources"
	"github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/timeutil"
	"google.golang.org/grpc"
)

var (
	authorizer = or.SensorOr(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Cluster)): {
			v1.ClustersService_GetClusters_FullMethodName,
			v1.ClustersService_GetCluster_FullMethodName,
			v1.ClustersService_GetKernelSupportAvailable_FullMethodName,
			v1.ClustersService_GetClusterDefaultValues_FullMethodName,
		},
		user.With(permissions.Modify(resources.Cluster)): {
			v1.ClustersService_PostCluster_FullMethodName,
			v1.ClustersService_PutCluster_FullMethodName,
			v1.ClustersService_DeleteCluster_FullMethodName,
		},
	}))

	// skewOptionsMap contains only the SensorVersionCompatibility field. It is
	// computed at runtime in the datastore layer and has no DB column
	// (storage/cluster.proto sql:"-"), so it can't be pushed down to the DB.
	// It is used to split an incoming query into a DB-safe part and a part
	// that must be evaluated in memory against the already-populated field.
	skewOptionsMap = search.NewOptionsMap(v1.SearchCategory_CLUSTERS).Add(
		search.SensorVersionCompatibility,
		schema.ClustersSchema.OptionsMap.MustGet(search.SensorVersionCompatibility.String()),
	)

	clusterPredicateFactory = predicate.NewFactory("cluster", (*storage.Cluster)(nil))
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
		return nil, errox.InvalidArgs.New("Id field should be empty when posting a new cluster")
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
		return nil, errox.InvalidArgs.New("Id must be provided")
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
		return nil, errox.InvalidArgs.New("Id must be provided")
	}
	return s.getCluster(ctx, request.GetId())
}

func (s *serviceImpl) getCluster(ctx context.Context, id string) (*v1.ClusterResponse, error) {
	cluster, ok, err := s.datastore.GetCluster(ctx, id)
	if err != nil {
		return nil, errors.Errorf("Could not get cluster: %s", err)
	}
	if !ok {
		return nil, errox.NotFound.New("Not found")
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

	systemPrivateConfig, err := s.sysConfigDatastore.GetPrivateConfig(ctx)
	if err != nil {
		return nil, err
	}

	clusterRetentionConfig := systemPrivateConfig.GetDecommissionedClusterRetention()
	if clusterRetentionConfig == nil || clusterRetentionConfig.GetRetentionDurationDays() == 0 {
		// If retention is disabled, there is no "days remaining" calculation to be done.
		return nil, nil
	}

	if maputil.MapsIntersect(clusterRetentionConfig.GetIgnoreClusterLabels(), cluster.GetLabels()) {
		return &v1.DecommissionedClusterRetentionInfo{
			RetentionInfo: &v1.DecommissionedClusterRetentionInfo_IsExcluded{
				IsExcluded: true,
			},
		}, nil
	}

	timeNow := time.Now()
	retentionDays := clusterRetentionConfig.GetRetentionDurationDays()

	configCreateTime, err := protocompat.ConvertTimestampToTimeOrError(clusterRetentionConfig.GetCreatedAt())
	if err != nil {
		return nil, err
	}

	lastContactTime, err := protocompat.ConvertTimestampToTimeOrError(cluster.GetHealthStatus().GetLastContact())
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
	fullQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errox.InvalidArgs.CausedByf("invalid query %q: %v", req.GetQuery(), err)
	}

	// Split the query: dbQuery contains everything the DB can filter on;
	// skewQuery contains only the SensorVersionCompatibility field, which
	// is computed at runtime and has no DB column.
	dbQuery, _ := search.InverseFilterQueryWithMap(fullQuery, skewOptionsMap)
	skewQuery, _ := search.FilterQueryWithMap(fullQuery, skewOptionsMap)

	clusters, err := s.datastore.SearchRawClusters(ctx, dbQuery)
	if err != nil {
		return nil, err
	}

	if skewQuery != nil {
		pred, err := clusterPredicateFactory.GeneratePredicate(skewQuery)
		if err != nil {
			return nil, errox.InvalidArgs.CausedByf("building predicate for version compatibility filter %q: %v", req.GetQuery(), err)
		}
		var filtered []*storage.Cluster
		for _, cluster := range clusters {
			// TODO(ROX-36353): Try to move this to the db layer instead here
			// Clusters where sensor has never connected have nil Status.
			// Initialize it so the predicate can match UNKNOWN.
			if cluster.GetStatus() == nil {
				cluster.Status = &storage.ClusterStatus{
					SensorVersionCompatibility: storage.SensorVersionCompatibility_SENSOR_VERSION_COMPATIBILITY_UNKNOWN,
				}
			}
			if pred.Matches(cluster) {
				filtered = append(filtered, cluster)
			}
		}
		clusters = filtered
	}

	// If we want to add pagination in the future it will have to be done
	// in memory at this point, after both DB-side and in-memory filters
	// have been applied.

	clusterIDToRetentionInfoMap, err := s.getClusterIDToRetentionInfoMap(ctx, clusters)
	if err != nil {
		return nil, err
	}

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
		return nil, errox.InvalidArgs.New("Request must have a id")
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
		CollectorImageRepository: flavor.CollectorImageNoTag(),
		KernelSupportAvailable:   kernelSupport,
	}
	return defaults, nil
}
