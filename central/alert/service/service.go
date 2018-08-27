package service

import (
	"context"

	"github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is a thin facade over a domain layer that handles CRUD use cases on Alert objects from API clients.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetAlert(ctx context.Context, request *v1.ResourceByID) (*v1.Alert, error)
	ListAlerts(ctx context.Context, request *v1.ListAlertsRequest) (*v1.ListAlertsResponse, error)
	GetAlertsGroup(ctx context.Context, request *v1.ListAlertsRequest) (*v1.GetAlertsGroupResponse, error)
	GetAlertsCounts(ctx context.Context, request *v1.GetAlertsCountsRequest) (*v1.GetAlertsCountsResponse, error)
	GetAlertTimeseries(ctx context.Context, req *v1.ListAlertsRequest) (*v1.GetAlertTimeseriesResponse, error)
}

// New returns a new Service soleInstance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		dataStore: datastore,
	}
}
