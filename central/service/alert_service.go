package service

import (
	"context"
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewAlertService returns the AlertService object.
func NewAlertService(storage db.AlertStorage) *AlertService {
	return &AlertService{
		storage: storage,
	}
}

// AlertService provides APIs for alerts.
type AlertService struct {
	storage db.AlertStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *AlertService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAlertServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *AlertService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAlertServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetAlert returns the alert with given id.
func (s *AlertService) GetAlert(ctx context.Context, request *v1.GetAlertRequest) (*v1.Alert, error) {
	alert, exists, err := s.storage.GetAlert(request.GetId())
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		log.Error(err)
		return nil, status.Errorf(codes.NotFound, "alert with id '%s' does not exist", request.GetId())
	}

	return alert, nil
}

// GetAlerts returns alerts according to the request.
func (s *AlertService) GetAlerts(ctx context.Context, request *v1.GetAlertsRequest) (*v1.GetAlertsResponse, error) {
	alerts, err := s.storage.GetAlerts(request)
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GetAlertsResponse{Alerts: alerts}, nil
}

// GetAlertsGroup returns alerts according to the request, grouped by category and policy.
func (s *AlertService) GetAlertsGroup(ctx context.Context, request *v1.GetAlertsRequest) (*v1.GetAlertsGroupResponse, error) {
	alerts, err := s.storage.GetAlerts(request)
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := s.groupAlerts(alerts)
	return response, nil
}

// GetAlertsCounts returns alert counts by severity according to the request.
// Counts can be grouped by policy category or cluster.
func (s *AlertService) GetAlertsCounts(ctx context.Context, request *v1.GetAlertsCountsRequest) (*v1.GetAlertsCountsResponse, error) {
	alerts, err := s.storage.GetAlerts(request.GetRequest())
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	if groupByFunc, ok := groupByFuncs[request.GetGroupBy()]; ok {
		response := s.countAlerts(alerts, groupByFunc)
		return response, nil
	}

	return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unknown group by: %v", request.GetGroupBy()))
}

func (s *AlertService) groupAlerts(alerts []*v1.Alert) (output *v1.GetAlertsGroupResponse) {
	policiesMap := make(map[string]*v1.Policy)
	alertCountsByPolicy := make(map[string]int)

	for _, a := range alerts {
		policiesMap[a.GetPolicy().GetId()] = a.GetPolicy()
		alertCountsByPolicy[a.GetPolicy().GetId()]++
	}

	output = new(v1.GetAlertsGroupResponse)
	output.AlertsByPolicies = make([]*v1.GetAlertsGroupResponse_PolicyGroup, 0, len(policiesMap))

	for id, p := range policiesMap {
		output.AlertsByPolicies = append(output.AlertsByPolicies, &v1.GetAlertsGroupResponse_PolicyGroup{
			Policy:    p,
			NumAlerts: int64(alertCountsByPolicy[id]),
		})
	}

	sort.Slice(output.AlertsByPolicies, func(i, j int) bool {
		return output.AlertsByPolicies[i].GetPolicy().GetName() < output.AlertsByPolicies[j].GetPolicy().GetName()
	})

	return
}

func (s *AlertService) countAlerts(alerts []*v1.Alert, groupByFunc func(*v1.Alert) []string) (output *v1.GetAlertsCountsResponse) {
	groups := s.getMapOfAlertCounts(alerts, groupByFunc)

	output = new(v1.GetAlertsCountsResponse)
	output.Groups = make([]*v1.GetAlertsCountsResponse_AlertGroup, 0, len(groups))

	for group, countsBySeverity := range groups {
		bySeverity := make([]*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts, 0, len(countsBySeverity))

		for severity, count := range countsBySeverity {
			bySeverity = append(bySeverity, &v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
				Severity: severity,
				Count:    int64(count),
			})
		}

		sort.Slice(bySeverity, func(i, j int) bool {
			return bySeverity[i].Severity < bySeverity[j].Severity
		})

		output.Groups = append(output.Groups, &v1.GetAlertsCountsResponse_AlertGroup{
			Group:  group,
			Counts: bySeverity,
		})
	}

	sort.Slice(output.Groups, func(i, j int) bool {
		return output.Groups[i].Group < output.Groups[j].Group
	})

	return
}

func (s *AlertService) getMapOfAlertCounts(alerts []*v1.Alert, groupByFunc func(*v1.Alert) []string) (groups map[string]map[v1.Severity]int) {
	groups = make(map[string]map[v1.Severity]int)

	for _, a := range alerts {
		for _, g := range groupByFunc(a) {
			if groups[g] == nil {
				groups[g] = make(map[v1.Severity]int)
			}

			groups[g][a.GetPolicy().GetSeverity()]++
		}
	}

	return
}

func getEventsFromAlerts(alerts []*v1.Alert) (events []*v1.Event) {
	// Optimization: The final size is guaranteed to be at least len(alerts)
	events = make([]*v1.Event, 0, len(alerts))
	for _, a := range alerts {
		events = append(events, &v1.Event{Time: a.GetTime().GetSeconds() * 1000, Id: a.GetId(), Type: v1.Type_CREATED, Severity: a.GetPolicy().GetSeverity()})
		if a.GetStale() {
			events = append(events, &v1.Event{Time: a.GetMarkedStale().GetSeconds() * 1000, Id: a.GetId(), Type: v1.Type_REMOVED, Severity: a.GetPolicy().GetSeverity()})
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].GetTime() < events[j].GetTime() })
	return
}

// GetAlertTimeseries returns the timeseries format of the events based on the request parameters
func (s *AlertService) GetAlertTimeseries(ctx context.Context, req *v1.GetAlertsRequest) (*v1.GetAlertTimeseriesResponse, error) {
	alerts, err := s.storage.GetAlerts(req)
	if err != nil {
		return nil, err
	}
	response := new(v1.GetAlertTimeseriesResponse)
	response.Events = getEventsFromAlerts(alerts)
	return response, nil
}

var (
	groupByFuncs = map[v1.GetAlertsCountsRequest_RequestGroup]func(*v1.Alert) []string{
		v1.GetAlertsCountsRequest_UNSET: func(*v1.Alert) []string { return []string{""} },
		v1.GetAlertsCountsRequest_CATEGORY: func(a *v1.Alert) (output []string) {
			for _, c := range a.GetPolicy().GetCategories() {
				output = append(output, c.String())
			}
			return
		},
		v1.GetAlertsCountsRequest_CLUSTER: func(a *v1.Alert) []string { return []string{a.GetDeployment().GetClusterId()} },
	}
)
