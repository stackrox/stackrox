package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	_ v1.AdministrationEventServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			v1.AdministrationEventService_CountAdministrationEvents_FullMethodName,
			v1.AdministrationEventService_GetAdministrationEvent_FullMethodName,
			v1.AdministrationEventService_ListAdministrationEvents_FullMethodName,
		},
	})
)

var errNoMatchingImages = errors.New("no images match the workload filter")

type serviceImpl struct {
	v1.UnimplementedAdministrationEventServiceServer

	ds          datastore.DataStore
	deployments deploymentDatastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterAdministrationEventServiceServer(server, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAdministrationEventServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CountAdministrationEvents returns the number of events matching the request query.
func (s *serviceImpl) CountAdministrationEvents(ctx context.Context, request *v1.CountAdministrationEventsRequest) (*v1.CountAdministrationEventsResponse, error) {
	qb, err := s.getQueryBuilderFromFilter(ctx, request.GetFilter())
	if err != nil {
		if errors.Is(err, errNoMatchingImages) {
			return &v1.CountAdministrationEventsResponse{Count: 0}, nil
		}
		return nil, errors.Wrap(err, "building query from filter")
	}
	query := qb.ProtoQuery()
	count, err := s.ds.CountEvents(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count administration events")
	}
	return &v1.CountAdministrationEventsResponse{Count: int32(count)}, nil
}

// GetAdministrationEvent returns a specific administration event based on its ID.
func (s *serviceImpl) GetAdministrationEvent(ctx context.Context, resource *v1.ResourceByID) (*v1.GetAdministrationEventResponse, error) {
	resourceID := resource.GetId()
	event, err := s.ds.GetEvent(ctx, resourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get administration event %q", resourceID)
	}
	return &v1.GetAdministrationEventResponse{Event: toV1Proto(event)}, err
}

// ListAdministrationEvents returns all administration events matching the request query.
func (s *serviceImpl) ListAdministrationEvents(ctx context.Context, request *v1.ListAdministrationEventsRequest) (*v1.ListAdministrationEventsResponse, error) {
	qb, err := s.getQueryBuilderFromFilter(ctx, request.GetFilter())
	if err != nil {
		if errors.Is(err, errNoMatchingImages) {
			return &v1.ListAdministrationEventsResponse{}, nil
		}
		return nil, errors.Wrap(err, "building query from filter")
	}
	query := qb.ProtoQuery()
	paginated.FillPagination(query, request.GetPagination(), maxPaginationLimit)
	query = paginated.FillDefaultSortOption(
		query,
		&v1.QuerySortOption{
			Field:    search.LastUpdatedTime.String(),
			Reversed: true,
		},
	)

	events, err := s.ds.ListEvents(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list administration events")
	}
	respEvents := make([]*v1.AdministrationEvent, 0, len(events))
	for _, n := range events {
		respEvents = append(respEvents, toV1Proto(n))
	}
	return &v1.ListAdministrationEventsResponse{Events: respEvents}, nil
}

func (s *serviceImpl) getQueryBuilderFromFilter(ctx context.Context, filter *v1.AdministrationEventsFilter) (*search.QueryBuilder, error) {
	queryBuilder := search.NewQueryBuilder()
	if filter == nil {
		return queryBuilder, nil
	}

	queryBuilder = queryBuilder.
		AddTimeRangeField(
			search.CreatedTime,
			protoconv.ConvertTimestampToTimeOrDefault(filter.GetFrom(), time.Unix(0, 0)),
			// We could potentially miss events that were _just_ created, so create a jitter which allows to also
			// include those events in the list response. This was discovered in tests, where the time from upserting
			// events to listing them is relatively short.
			protoconv.ConvertTimestampToTimeOrDefault(filter.GetUntil(), time.Now().Add(time.Second)),
		)
	if domains := filter.GetDomain(); len(domains) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.EventDomain, sliceutils.Unique(domains)...)
	}
	if levels := filter.GetLevel(); len(levels) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.EventLevel,
			sliceutils.Unique(sliceutils.StringSlice(levels...))...)
	}
	if eventTypes := filter.GetType(); len(eventTypes) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.EventType,
			sliceutils.Unique(sliceutils.StringSlice(eventTypes...))...)
	}
	if resourceTypes := filter.GetResourceType(); len(resourceTypes) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.ResourceType, sliceutils.Unique(resourceTypes)...)
	}

	imageIDs, err := s.resolveWorkloadFilterToImageIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	if imageIDs != nil {
		if len(imageIDs) == 0 {
			return nil, errNoMatchingImages
		}
		queryBuilder = queryBuilder.AddExactMatches(search.ResourceType, "Image")
		queryBuilder = queryBuilder.AddExactMatches(search.ResourceID, imageIDs...)
	}

	return queryBuilder, nil
}

// resolveWorkloadFilterToImageIDs returns image IDs for deployments matching
// the cluster/namespace/deployment filter. Returns nil if no workload filter
// is set. Returns empty slice if the filter matched zero deployments.
func (s *serviceImpl) resolveWorkloadFilterToImageIDs(ctx context.Context, filter *v1.AdministrationEventsFilter) ([]string, error) {
	clusters := filter.GetCluster()
	namespaces := filter.GetNamespace()
	deploymentNames := filter.GetDeployment()

	if len(clusters) == 0 && len(namespaces) == 0 && len(deploymentNames) == 0 {
		return nil, nil
	}

	if resourceTypes := filter.GetResourceType(); len(resourceTypes) != 0 {
		hasImage := false
		for _, rt := range resourceTypes {
			if rt == "Image" {
				hasImage = true
				break
			}
		}
		if !hasImage {
			return nil, nil
		}
	}

	depQuery := search.NewQueryBuilder()
	if len(clusters) != 0 {
		depQuery = depQuery.AddExactMatches(search.Cluster, clusters...)
	}
	if len(namespaces) != 0 {
		depQuery = depQuery.AddExactMatches(search.Namespace, namespaces...)
	}
	if len(deploymentNames) != 0 {
		depQuery = depQuery.AddExactMatches(search.DeploymentName, deploymentNames...)
	}

	deployments, err := s.deployments.SearchRawDeployments(ctx, depQuery.ProtoQuery())
	if err != nil {
		return nil, errors.Wrap(err, "searching deployments for workload filter")
	}

	imageIDSet := set.NewStringSet()
	for _, dep := range deployments {
		for _, container := range dep.GetContainers() {
			if id := container.GetImage().GetId(); id != "" { //nolint:staticcheck // SA1019: GetId is deprecated but still used for legacy images.
				imageIDSet.Add(id)
			}
			if id := container.GetImage().GetIdV2(); id != "" {
				imageIDSet.Add(id)
			}
		}
	}
	return imageIDSet.AsSlice(), nil
}
