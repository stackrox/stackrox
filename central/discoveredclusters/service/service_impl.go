package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/storagetov1"
	"github.com/stackrox/rox/central/discoveredclusters/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.Administration)): {
		v1.DiscoveredClustersService_CountDiscoveredClusters_FullMethodName,
		v1.DiscoveredClustersService_GetDiscoveredCluster_FullMethodName,
		v1.DiscoveredClustersService_ListDiscoveredClusters_FullMethodName,
	},
})

type serviceImpl struct {
	v1.UnimplementedDiscoveredClustersServiceServer

	ds datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterDiscoveredClustersServiceServer(server, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDiscoveredClustersServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CountDiscoveredClusters returns the number of discovered clusters matching the request query.
func (s *serviceImpl) CountDiscoveredClusters(ctx context.Context, request *v1.CountDiscoveredClustersRequest,
) (*v1.CountDiscoveredClustersResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	count, err := s.ds.CountDiscoveredClusters(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count discovered clusters")
	}
	return &v1.CountDiscoveredClustersResponse{Count: int32(count)}, nil
}

// GetDiscoveredCluster returns a specific discovered cluster based on its ID.
func (s *serviceImpl) GetDiscoveredCluster(ctx context.Context, request *v1.GetDiscoveredClusterRequest,
) (*v1.GetDiscoveredClusterResponse, error) {
	resourceID := request.GetId()
	if resourceID == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be provided")
	}
	discoveredCluster, err := s.ds.GetDiscoveredCluster(ctx, resourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get discovered cluster %q", resourceID)
	}
	return &v1.GetDiscoveredClusterResponse{Cluster: storagetov1.DiscoveredCluster(discoveredCluster)}, nil
}

// ListDiscoveredClusters returns all discovered clusters matching the request query.
func (s *serviceImpl) ListDiscoveredClusters(ctx context.Context, request *v1.ListDiscoveredClustersRequest,
) (*v1.ListDiscoveredClustersResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	paginated.FillPagination(query, request.GetPagination(), maxPaginationLimit)
	query = paginated.FillDefaultSortOption(
		query,
		&v1.QuerySortOption{
			Field: search.Cluster.String(),
		},
	)

	discoveredClusters, err := s.ds.ListDiscoveredClusters(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list discovered clusters")
	}
	return &v1.ListDiscoveredClustersResponse{
		Clusters: storagetov1.DiscoveredClusterList(discoveredClusters...),
	}, nil
}

func getQueryBuilderFromFilter(filter *v1.DiscoveredClustersFilter) *search.QueryBuilder {
	queryBuilder := search.NewQueryBuilder()
	if names := filter.GetNames(); len(names) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.Cluster,
			sliceutils.Unique(names)...,
		)
	}
	if types := filter.GetTypes(); len(types) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.ClusterType,
			sliceutils.Unique(sliceutils.StringSlice(types...))...,
		)
	}
	if statuses := filter.GetStatuses(); len(statuses) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.ClusterStatus,
			sliceutils.Unique(sliceutils.StringSlice(statuses...))...,
		)
	}
	if sources := filter.GetSourceIds(); len(sources) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.IntegrationID,
			sliceutils.Unique(sources)...,
		)
	}
	return queryBuilder
}
