package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/deployment/datastore"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			"/v1.DeploymentService/GetDeployment",
			"/v1.DeploymentService/ListDeployments",
			"/v1.DeploymentService/GetLabels",
			"/v1.DeploymentService/GetMultipliers",
		},
		user.With(permissions.Modify(resources.Deployment)): {
			"/v1.DeploymentService/AddMultiplier",
			"/v1.DeploymentService/UpdateMultiplier",
			"/v1.DeploymentService/RemoveMultiplier",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	datastore         datastore.DataStore
	processIndicators processIndicatorStore.DataStore
	multipliers       multiplierStore.Store
	manager           manager.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDeploymentServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDeploymentServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetDeployment returns the deployment with given id.
func (s *serviceImpl) GetDeployment(ctx context.Context, request *v1.ResourceByID) (*storage.Deployment, error) {
	deployment, exists, err := s.datastore.GetDeployment(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "deployment with id '%s' does not exist", request.GetId())
	}
	return deployment, nil
}

// ListDeployments returns ListDeployments according to the request.
func (s *serviceImpl) ListDeployments(ctx context.Context, request *v1.RawQuery) (*v1.ListDeploymentsResponse, error) {
	var deployments []*storage.ListDeployment
	var err error
	if request.GetQuery() == "" {
		deployments, err = s.datastore.ListDeployments()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		deployments, err = s.datastore.SearchListDeployments(parsedQuery)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	sort.SliceStable(deployments, func(i, j int) bool {
		return deployments[i].GetPriority() < deployments[j].GetPriority()
	})
	return &v1.ListDeploymentsResponse{
		Deployments: deployments,
	}, nil
}

// GetLabels returns label keys and values for current deployments.
func (s *serviceImpl) GetLabels(context.Context, *v1.Empty) (*v1.DeploymentLabelsResponse, error) {
	deployments, err := s.datastore.GetDeployments()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	labelsMap, values := labelsMapFromDeployments(deployments)

	return &v1.DeploymentLabelsResponse{
		Labels: labelsMap,
		Values: values,
	}, nil
}

func labelsMapFromDeployments(deployments []*storage.Deployment) (keyValuesMap map[string]*v1.DeploymentLabelsResponse_LabelValues, values []string) {
	tempSet := make(map[string]set.StringSet)
	globalValueSet := set.NewStringSet()

	for _, d := range deployments {
		for k, v := range d.GetLabels() {
			valSet, ok := tempSet[k]
			if !ok {
				valSet = set.NewStringSet()
				tempSet[k] = valSet
			}
			valSet.Add(v)
			globalValueSet.Add(v)
		}
	}

	keyValuesMap = make(map[string]*v1.DeploymentLabelsResponse_LabelValues)
	for k, valSet := range tempSet {
		keyValuesMap[k] = &v1.DeploymentLabelsResponse_LabelValues{
			Values: make([]string, 0, valSet.Cardinality()),
		}

		keyValuesMap[k].Values = append(keyValuesMap[k].Values, valSet.AsSlice()...)
		sort.Strings(keyValuesMap[k].Values)
	}
	values = globalValueSet.AsSlice()
	sort.Strings(values)

	return
}

// GetMultipliers returns all multipliers
func (s *serviceImpl) GetMultipliers(ctx context.Context, request *v1.Empty) (*v1.GetMultipliersResponse, error) {
	multipliers, err := s.multipliers.GetMultipliers()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetMultipliersResponse{
		Multipliers: multipliers,
	}, nil
}

func validateMultiplier(mult *storage.Multiplier) error {
	errorList := errorhelpers.NewErrorList("Validation")
	if mult.GetName() == "" {
		errorList.AddString("multiplier name must be specified")
	}
	if mult.GetValue() < 1 || mult.GetValue() > 2 {
		errorList.AddString("multiplier must have a value between 1 and 2 inclusive")
	}
	return errorList.ToError()
}

// AddMultiplier inserts the specified multiplier
func (s *serviceImpl) AddMultiplier(ctx context.Context, request *storage.Multiplier) (*storage.Multiplier, error) {
	if err := validateMultiplier(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.multipliers.AddMultiplier(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	s.manager.UpdateMultiplier(request)
	return request, nil
}

// UpdateMultiplier updates the specified multiplier
func (s *serviceImpl) UpdateMultiplier(ctx context.Context, request *storage.Multiplier) (*v1.Empty, error) {
	if err := s.multipliers.UpdateMultiplier(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.manager.UpdateMultiplier(request)
	return &v1.Empty{}, nil
}

// RemoveMultiplier removes the specified multiplier
func (s *serviceImpl) RemoveMultiplier(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID must be specified when removing a multiplier")
	}
	if err := s.multipliers.RemoveMultiplier(request.GetId()); err != nil {
		if _, ok := err.(dberrors.ErrNotFound); ok {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.manager.RemoveMultiplier(request.GetId())
	return &v1.Empty{}, nil
}
