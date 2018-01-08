package service

import (
	"context"
	"sort"

	"bitbucket.org/stack-rox/apollo/apollo/db"
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
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "alert with id '%s' does not exist", request.GetId())
	}

	return alert, nil
}

// GetAlerts returns alerts according to the request.
func (s *AlertService) GetAlerts(ctx context.Context, request *v1.GetAlertsRequest) (*v1.GetAlertsResponse, error) {
	alerts, err := s.storage.GetAlerts(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GetAlertsResponse{Alerts: alerts}, nil
}

// GetAlertsGroup returns alerts according to the request, grouped by category and policy.
func (s *AlertService) GetAlertsGroup(ctx context.Context, request *v1.GetAlertsRequest) (*v1.GetAlertsGroupResponse, error) {
	alerts, err := s.storage.GetAlerts(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := s.groupAlerts(alerts)
	return response, nil
}

func (s *AlertService) groupAlerts(alerts []*v1.Alert) (output *v1.GetAlertsGroupResponse) {
	group := make(map[v1.Policy_Category]map[string]*v1.GetAlertsGroupResponse_PolicyGroup)

	for _, a := range alerts {
		pol := a.GetPolicy()
		cats := a.GetPolicy().GetCategories()

		for _, cat := range cats {
			if group[cat] == nil {
				group[cat] = make(map[string]*v1.GetAlertsGroupResponse_PolicyGroup)
			}

			if existing := group[cat][pol.GetName()]; existing == nil {
				group[cat][pol.GetName()] = &v1.GetAlertsGroupResponse_PolicyGroup{
					Policy:    pol,
					NumAlerts: 1,
				}
			} else {
				existing.NumAlerts++
			}
		}

	}

	output = new(v1.GetAlertsGroupResponse)
	output.ByCategory = make([]*v1.GetAlertsGroupResponse_CategoryGroup, 0, len(group))

	for cat, byPolicy := range group {
		policyGroups := make([]*v1.GetAlertsGroupResponse_PolicyGroup, 0, len(byPolicy))

		for _, policyGroup := range byPolicy {
			policyGroups = append(policyGroups, policyGroup)
		}

		sort.Slice(policyGroups, func(i, j int) bool { return policyGroups[i].Policy.GetName() < policyGroups[j].Policy.GetName() })

		output.ByCategory = append(output.ByCategory, &v1.GetAlertsGroupResponse_CategoryGroup{
			Category: cat,
			ByPolicy: policyGroups,
		})
	}

	sort.Slice(output.ByCategory, func(i, j int) bool { return output.ByCategory[i].Category < output.ByCategory[j].Category })

	return
}
