package service

import (
	"context"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// NewDashboardService returns the DashboardService object.
func NewDashboardService(storage db.AlertStorage) *DashboardService {
	return &DashboardService{
		storage: storage,
	}
}

// DashboardService provides APIs for the dashboard.
type DashboardService struct {
	storage db.AlertStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *DashboardService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDashboardServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *DashboardService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterDashboardServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func getEventsFromAlerts(alerts []*v1.Alert) (events []*v1.Event) {
	// Optimization: The final size is guaranteed to be at least len(alerts)
	events = make([]*v1.Event, 0, len(alerts))
	for _, a := range alerts {
		events = append(events, &v1.Event{Time: a.GetTime().GetSeconds() * 1000, Id: a.GetId(), Action: v1.Action_CREATED})
		if a.GetStale() {
			events = append(events, &v1.Event{Time: a.GetMarkedStale().GetSeconds() * 1000, Id: a.GetId(), Action: v1.Action_REMOVED})
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].GetTime() < events[j].GetTime() })
	return
}

// GetAlertTimeseries returns the timeseries format of the events based on the request parameters
func (s *DashboardService) GetAlertTimeseries(ctx context.Context, req *v1.GetAlertsRequest) (*v1.GetAlertTimeseriesResponse, error) {
	alerts, err := s.storage.GetAlerts(req)
	if err != nil {
		return nil, err
	}
	response := new(v1.GetAlertTimeseriesResponse)
	response.Events = getEventsFromAlerts(alerts)
	return response, nil
}
