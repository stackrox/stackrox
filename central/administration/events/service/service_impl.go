package service

import (
	"context"
	"slices"
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
	"github.com/stackrox/rox/pkg/logging"
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

var log = logging.LoggerForModule()

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
	query, err := s.getQueryFromFilter(ctx, request.GetFilter())
	if err != nil {
		return nil, errors.Wrap(err, "building query from filter")
	}
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
	query, err := s.getQueryFromFilter(ctx, request.GetFilter())
	if err != nil {
		return nil, errors.Wrap(err, "building query from filter")
	}
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

func (s *serviceImpl) getQueryFromFilter(ctx context.Context, filter *v1.AdministrationEventsFilter) (*v1.Query, error) {
	baseQuery := search.NewQueryBuilder()
	if filter == nil {
		return baseQuery.ProtoQuery(), nil
	}

	baseQuery = baseQuery.
		AddTimeRangeField(
			search.CreatedTime,
			protoconv.ConvertTimestampToTimeOrDefault(filter.GetFrom(), time.Unix(0, 0)),
			// We could potentially miss events that were _just_ created, so create a jitter which allows to also
			// include those events in the list response. This was discovered in tests, where the time from upserting
			// events to listing them is relatively short.
			protoconv.ConvertTimestampToTimeOrDefault(filter.GetUntil(), time.Now().Add(time.Second)),
		)
	if domains := filter.GetDomain(); len(domains) != 0 {
		baseQuery = baseQuery.AddExactMatches(search.EventDomain, sliceutils.Unique(domains)...)
	}
	if levels := filter.GetLevel(); len(levels) != 0 {
		baseQuery = baseQuery.AddExactMatches(search.EventLevel,
			sliceutils.Unique(sliceutils.StringSlice(levels...))...)
	}
	if eventTypes := filter.GetType(); len(eventTypes) != 0 {
		baseQuery = baseQuery.AddExactMatches(search.EventType,
			sliceutils.Unique(sliceutils.StringSlice(eventTypes...))...)
	}

	resourceTypes := sliceutils.Unique(filter.GetResourceType())
	imageIDs := s.resolveWorkloadImageIDs(ctx, filter)

	if imageIDs == nil {
		if len(resourceTypes) != 0 {
			baseQuery = baseQuery.AddExactMatches(search.ResourceType, resourceTypes...)
		}
		return baseQuery.ProtoQuery(), nil
	}

	imageQuery := search.NewQueryBuilder().
		AddExactMatches(search.ResourceType, "Image").
		AddExactMatches(search.ResourceID, imageIDs...)

	otherTypes := sliceutils.Without(resourceTypes, []string{"Image"})

	var otherQuery *v1.Query
	switch {
	case len(resourceTypes) == 0:
		notImageType := search.NewQueryBuilder().AddExactMatches(search.ResourceType, "Image").ProtoQuery()
		otherQuery = search.NewBooleanQuery(nil, &v1.DisjunctionQuery{Queries: []*v1.Query{notImageType}})
	case len(otherTypes) == 0:
		return search.ConjunctionQuery(baseQuery.ProtoQuery(), imageQuery.ProtoQuery()), nil
	default:
		otherQuery = search.NewQueryBuilder().AddExactMatches(search.ResourceType, otherTypes...).ProtoQuery()
	}

	resourceFilter := search.DisjunctionQuery(imageQuery.ProtoQuery(), otherQuery)
	return search.ConjunctionQuery(baseQuery.ProtoQuery(), resourceFilter), nil
}

// resolveWorkloadImageIDs returns image IDs for deployments matching the workload
// query, or nil if the workload filter does not apply.
func (s *serviceImpl) resolveWorkloadImageIDs(ctx context.Context, filter *v1.AdministrationEventsFilter) []string {
	rawQuery := filter.GetWorkloadQuery()
	if rawQuery == "" {
		return nil
	}
	if resourceTypes := filter.GetResourceType(); len(resourceTypes) != 0 && !slices.Contains(resourceTypes, "Image") {
		return nil
	}

	parsedQuery, err := search.ParseQuery(rawQuery)
	if err != nil {
		log.Warnw("Ignoring workload filter due to query parse error", logging.Err(err))
		return nil
	}

	imageViews, err := s.deployments.GetContainerImageViews(ctx, parsedQuery)
	if err != nil {
		log.Warnw("Ignoring workload filter due to deployment search error", logging.Err(err))
		return nil
	}

	imageIDSet := set.NewStringSet()
	for _, view := range imageViews {
		if id := view.GetImageID(); id != "" {
			imageIDSet.Add(id)
		}
	}
	if imageIDSet.Cardinality() == 0 {
		return []string{""}
	}
	return imageIDSet.AsSlice()
}
