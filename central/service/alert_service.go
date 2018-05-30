package service

import (
	"context"
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewAlertService returns the AlertService object.
func NewAlertService(datastore datastore.AlertDataStore) *AlertService {
	return &AlertService{
		datastore: datastore,
	}
}

// AlertService provides APIs for alerts.
type AlertService struct {
	datastore datastore.AlertDataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *AlertService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAlertServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *AlertService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAlertServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *AlertService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(user.Any().Authorized(ctx))
}

// GetAlert returns the alert with given id.
func (s *AlertService) GetAlert(ctx context.Context, request *v1.ResourceByID) (*v1.Alert, error) {
	alert, exists, err := s.datastore.GetAlert(request.GetId())
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "alert with id '%s' does not exist", request.GetId())
	}

	return alert, nil
}

func convertAlertsToListAlerts(alerts []*v1.Alert) []*v1.ListAlert {
	listAlerts := make([]*v1.ListAlert, 0, len(alerts))
	for _, a := range alerts {
		listAlerts = append(listAlerts, &v1.ListAlert{
			Id:   a.GetId(),
			Time: a.GetTime(),
			Policy: &v1.ListAlert_Policy{
				Id:          a.GetPolicy().GetId(),
				Name:        a.GetPolicy().GetName(),
				Severity:    a.GetPolicy().GetSeverity(),
				Description: a.GetPolicy().GetDescription(),
				Categories:  a.GetPolicy().GetCategories(),
			},
			Deployment: &v1.ListAlert_Deployment{
				Id:          a.GetDeployment().GetId(),
				Name:        a.GetDeployment().GetName(),
				UpdatedAt:   a.GetDeployment().GetUpdatedAt(),
				ClusterName: a.GetDeployment().GetClusterName(),
			},
		})
	}
	return listAlerts
}

// ListAlerts returns ListAlerts according to the request.
func (s *AlertService) ListAlerts(ctx context.Context, request *v1.ListAlertsRequest) (*v1.ListAlertsResponse, error) {
	alerts, err := s.datastore.GetAlerts(request)
	if err != nil {
		return nil, err
	}
	return &v1.ListAlertsResponse{Alerts: convertAlertsToListAlerts(alerts)}, nil
}

// GetAlertsGroup returns alerts according to the request, grouped by category and policy.
func (s *AlertService) GetAlertsGroup(ctx context.Context, request *v1.ListAlertsRequest) (*v1.GetAlertsGroupResponse, error) {
	alerts, err := s.datastore.GetAlerts(request)
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
	alerts, err := s.datastore.GetAlerts(request.GetRequest())
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
		if _, ok := policiesMap[a.GetPolicy().GetId()]; !ok {
			policiesMap[a.GetPolicy().GetId()] = a.GetPolicy()
		}
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

// GetAlertTimeseries returns the timeseries format of the events based on the request parameters
func (s *AlertService) GetAlertTimeseries(ctx context.Context, req *v1.ListAlertsRequest) (*v1.GetAlertTimeseriesResponse, error) {
	alerts, err := s.datastore.GetAlerts(req)
	if err != nil {
		return nil, err
	}

	response := new(v1.GetAlertTimeseriesResponse)
	for cluster, severityMap := range getGroupToAlertEvents(alerts) {
		alertCluster := &v1.GetAlertTimeseriesResponse_ClusterAlerts{Cluster: cluster}
		for severity, alertEvents := range severityMap {
			// Sort the alert events so they are chronological
			sort.SliceStable(alertEvents, func(i, j int) bool { return alertEvents[i].GetTime() < alertEvents[j].GetTime() })
			alertCluster.Severities = append(alertCluster.Severities, &v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
				Severity: severity,
				Events:   alertEvents,
			})
		}
		sort.Slice(alertCluster.Severities, func(i, j int) bool { return alertCluster.Severities[i].Severity < alertCluster.Severities[j].Severity })
		response.Clusters = append(response.Clusters, alertCluster)
	}
	sort.SliceStable(response.Clusters, func(i, j int) bool { return response.Clusters[i].Cluster < response.Clusters[j].Cluster })
	return response, nil
}

func getGroupToAlertEvents(alerts []*v1.Alert) (clusters map[string]map[v1.Severity][]*v1.AlertEvent) {
	clusters = make(map[string]map[v1.Severity][]*v1.AlertEvent)
	for _, a := range alerts {
		alertCluster := a.GetDeployment().GetClusterName()
		if clusters[alertCluster] == nil {
			clusters[alertCluster] = make(map[v1.Severity][]*v1.AlertEvent)
		}
		eventList := clusters[alertCluster][a.GetPolicy().GetSeverity()]
		eventList = append(eventList, &v1.AlertEvent{Time: a.GetTime().GetSeconds() * 1000, Id: a.GetId(), Type: v1.Type_CREATED})
		if a.GetStale() {
			eventList = append(eventList, &v1.AlertEvent{Time: a.GetMarkedStale().GetSeconds() * 1000, Id: a.GetId(), Type: v1.Type_REMOVED})
		}
		clusters[alertCluster][a.GetPolicy().GetSeverity()] = eventList
	}

	for _, v1 := range clusters {
		for k2, v2 := range v1 {
			sort.SliceStable(v2, func(i, j int) bool { return v2[i].GetTime() < v2[j].GetTime() })
			v1[k2] = v2
		}
	}
	return
}

var (
	groupByFuncs = map[v1.GetAlertsCountsRequest_RequestGroup]func(*v1.Alert) []string{
		v1.GetAlertsCountsRequest_UNSET: func(*v1.Alert) []string { return []string{""} },
		v1.GetAlertsCountsRequest_CATEGORY: func(a *v1.Alert) (output []string) {
			for _, c := range a.GetPolicy().GetCategories() {
				output = append(output, c)
			}
			return
		},
		v1.GetAlertsCountsRequest_CLUSTER: func(a *v1.Alert) []string { return []string{a.GetDeployment().GetClusterName()} },
	}
)
