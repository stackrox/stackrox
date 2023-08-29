package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifications/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	_ v1.NotificationServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.NotificationService/CountNotifications",
			"/v1.NotificationService/GetNotification",
			"/v1.NotificationService/ListNotifications",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedNotificationServiceServer

	ds datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterNotificationServiceServer(server, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNotificationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CountNotifications returns the number of notifications matching the request query.
func (s *serviceImpl) CountNotifications(ctx context.Context, request *v1.CountNotificationsRequest) (*v1.CountNotificationsResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	count, err := s.ds.CountNotifications(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count notifications")
	}
	return &v1.CountNotificationsResponse{Count: int64(count)}, nil
}

// GetNotification returns a specific notification based on its ID.
func (s *serviceImpl) GetNotification(ctx context.Context, resource *v1.ResourceByID) (*v1.GetNotificationResponse, error) {
	resourceID := resource.GetId()
	notification, err := s.ds.GetNotificationByID(ctx, resourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get notification %q", resourceID)
	}
	return &v1.GetNotificationResponse{Notification: convertToServiceType(notification)}, err
}

// ListNotifications returns all notifications matching the request query.
func (s *serviceImpl) ListNotifications(ctx context.Context, request *v1.ListNotificationsRequest) (*v1.ListNotificationsResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	paginated.FillPagination(query, request.GetPagination(), maxPaginationLimit)
	paginated.FillDefaultSortOption(
		query,
		&v1.QuerySortOption{
			Field:    search.LastUpdatedTime.String(),
			Reversed: true,
		},
	)

	notifications, err := s.ds.ListNotifications(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list notifications")
	}
	var respNotifications []*v1.Notification
	for _, n := range notifications {
		respNotifications = append(respNotifications, convertToServiceType(n))
	}
	return &v1.ListNotificationsResponse{Notifications: respNotifications}, nil
}

func getQueryBuilderFromFilter(filter *v1.NotificationsFilter) *search.QueryBuilder {
	queryBuilder := search.NewQueryBuilder().
		AddTimeRangeField(
			search.CreatedTime,
			protoconv.ConvertTimestampToTimeOrDefault(filter.GetFrom(), time.Unix(0, 0)),
			protoconv.ConvertTimestampToTimeOrNow(filter.GetUntil()),
		)
	if domain := filter.GetDomain(); domain != "" {
		queryBuilder = queryBuilder.AddExactMatches(search.NotificationDomain, domain)
	}
	if level := filter.GetLevel(); level != v1.NotificationLevel_NOTIFICATION_LEVEL_UNKNOWN {
		queryBuilder = queryBuilder.AddExactMatches(search.NotificationLevel, level.String())
	}
	if notificationType := filter.GetNotificationType(); notificationType != v1.NotificationType_NOTIFICATION_TYPE_UNKNOWN {
		queryBuilder = queryBuilder.AddExactMatches(search.NotificationType, notificationType.String())
	}
	if resourceType := filter.GetResourceType(); resourceType != "" {
		queryBuilder = queryBuilder.AddExactMatches(search.ResourceType, resourceType)
	}
	return queryBuilder
}

func convertToServiceType(notification *storage.Notification) *v1.Notification {
	if notification == nil {
		return nil
	}
	return &v1.Notification{
		Id:             notification.GetId(),
		Type:           v1.NotificationType(notification.GetType()),
		Level:          v1.NotificationLevel(notification.GetLevel()),
		Message:        notification.GetMessage(),
		Hint:           notification.GetHint(),
		Domain:         notification.GetDomain(),
		ResourceType:   notification.GetResourceType(),
		ResourceId:     notification.GetResourceId(),
		NumOccurrences: notification.GetNumOccurrences(),
		LastOccurredAt: notification.GetLastOccurredAt(),
		CreatedAt:      notification.GetCreatedAt(),
	}
}
