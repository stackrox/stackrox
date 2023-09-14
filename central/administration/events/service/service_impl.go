package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	_ v1.AdministrationEventServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.AdministrationEventService/CountAdministrationEvents",
			"/v1.AdministrationEventService/GetAdministrationEvent",
			"/v1.AdministrationEventService/ListAdministrationEvents",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedAdministrationEventServiceServer

	ds datastore.DataStore
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
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	count, err := s.ds.CountEvents(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count administration events")
	}
	return &v1.CountAdministrationEventsResponse{Count: int64(count)}, nil
}

// GetAdministrationEvent returns a specific administration event based on its ID.
func (s *serviceImpl) GetAdministrationEvent(ctx context.Context, resource *v1.ResourceByID) (*v1.GetAdministrationEventResponse, error) {
	resourceID := resource.GetId()
	event, err := s.ds.GetEventByID(ctx, resourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get administration event %q", resourceID)
	}
	return &v1.GetAdministrationEventResponse{Event: toV1Proto(event)}, err
}

// ListAdministrationEvents returns all administration events matching the request query.
func (s *serviceImpl) ListAdministrationEvents(ctx context.Context, request *v1.ListAdministrationEventsRequest) (*v1.ListAdministrationEventsResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	paginated.FillPagination(query, request.GetPagination(), maxPaginationLimit)
	paginated.FillDefaultSortOption(
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

func getQueryBuilderFromFilter(filter *v1.AdministrationEventsFilter) *search.QueryBuilder {
	queryBuilder := search.NewQueryBuilder()
	if filter == nil {
		return queryBuilder
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
	if domain := filter.GetDomain(); domain != "" {
		queryBuilder = queryBuilder.AddExactMatches(search.EventDomain, domain)
	}
	if level := filter.GetLevel(); level != v1.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_UNKNOWN {
		queryBuilder = queryBuilder.AddExactMatches(search.EventLevel, level.String())
	}
	if eventType := filter.GetType(); eventType != v1.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_UNKNOWN {
		queryBuilder = queryBuilder.AddExactMatches(search.EventType, eventType.String())
	}
	if resourceType := filter.GetResourceType(); resourceType != "" {
		queryBuilder = queryBuilder.AddExactMatches(search.ResourceType, resourceType)
	}
	return queryBuilder
}
