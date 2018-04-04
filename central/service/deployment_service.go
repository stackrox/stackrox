package service

import (
	"context"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewDeploymentService returns a DeploymentService object.
func NewDeploymentService(storage *datastore.DataStore, enricher *enrichment.Enricher) *DeploymentService {
	return &DeploymentService{
		storage:  storage,
		enricher: enricher,
	}
}

// DeploymentService provides APIs for deployments.
type DeploymentService struct {
	storage  *datastore.DataStore
	enricher *enrichment.Enricher
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *DeploymentService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDeploymentServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *DeploymentService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterDeploymentServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *DeploymentService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(user.Any().Authorized(ctx))
}

// GetDeployment returns the deployment with given id.
func (s *DeploymentService) GetDeployment(ctx context.Context, request *v1.ResourceByID) (*v1.Deployment, error) {
	deployment, exists, err := s.storage.GetDeployment(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "deployment with id '%s' does not exist", request.GetId())
	}

	return deployment, nil
}

// GetDeployments returns deployments according to the request.
func (s *DeploymentService) GetDeployments(ctx context.Context, request *v1.RawQuery) (*v1.GetDeploymentsResponse, error) {
	resp := new(v1.GetDeploymentsResponse)
	if request.GetQuery() == "" {
		deployments, err := s.storage.GetDeployments()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Deployments = deployments
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		deployments, err := s.storage.SearchRawDeployments(parsedQuery)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Deployments = deployments
	}
	return resp, nil
}

// GetLabels returns label keys and values for current deployments.
func (s *DeploymentService) GetLabels(context.Context, *empty.Empty) (*v1.DeploymentLabelsResponse, error) {
	deployments, err := s.storage.GetDeployments()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	labelsMap, values := labelsMapFromDeployments(deployments)

	return &v1.DeploymentLabelsResponse{
		Labels: labelsMap,
		Values: values,
	}, nil
}

func labelsMapFromDeployments(deployments []*v1.Deployment) (keyValuesMap map[string]*v1.DeploymentLabelsResponse_LabelValues, values []string) {
	tempSet := make(map[string]map[string]struct{})
	valSet := make(map[string]struct{})

	for _, d := range deployments {
		for k, v := range d.GetLabels() {
			if valSet := tempSet[k]; valSet == nil {
				tempSet[k] = map[string]struct{}{v: {}}
			} else {
				valSet[v] = struct{}{}
			}

			valSet[v] = struct{}{}
		}
	}

	keyValuesMap = make(map[string]*v1.DeploymentLabelsResponse_LabelValues)
	for k, valSet := range tempSet {
		keyValuesMap[k] = &v1.DeploymentLabelsResponse_LabelValues{
			Values: make([]string, 0, len(valSet)),
		}

		for v := range valSet {
			keyValuesMap[k].Values = append(keyValuesMap[k].Values, v)
		}
		sort.Strings(keyValuesMap[k].Values)
	}

	values = make([]string, 0, len(valSet))
	for v := range valSet {
		values = append(values, v)
	}
	sort.Strings(values)

	return
}

// GetMultipliers returns all multipliers
func (s *DeploymentService) GetMultipliers(ctx context.Context, request *empty.Empty) (*v1.GetMultipliersResponse, error) {
	multipliers, err := s.storage.GetMultipliers()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetMultipliersResponse{
		Multipliers: multipliers,
	}, nil
}

func validateMultiplier(mult *v1.Multiplier) error {
	var errs []string
	if mult.GetName() == "" {
		errs = append(errs, "Multiplier name must be specified")
	}
	if mult.GetValue() < 1 || mult.GetValue() > 2 {
		errs = append(errs, "Multiplier must have a value between 1 and 2 inclusive")
	}
	if len(errs) > 0 {
		return errorhelpers.FormatErrorStrings("Validation", errs)
	}
	return nil

}

// AddMultiplier inserts the specified multiplier
func (s *DeploymentService) AddMultiplier(ctx context.Context, request *v1.Multiplier) (*v1.Multiplier, error) {
	if err := validateMultiplier(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.storage.AddMultiplier(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	s.enricher.UpdateMultiplier(request)
	return request, nil
}

// UpdateMultiplier updates the specified multiplier
func (s *DeploymentService) UpdateMultiplier(ctx context.Context, request *v1.Multiplier) (*empty.Empty, error) {
	if err := s.storage.UpdateMultiplier(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.enricher.UpdateMultiplier(request)
	return &empty.Empty{}, nil
}

// RemoveMultiplier removes the specified multiplier
func (s *DeploymentService) RemoveMultiplier(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID must be specified when removing a multiplier")
	}
	if err := s.storage.RemoveMultiplier(request.GetId()); err != nil {
		if _, ok := err.(db.ErrNotFound); ok {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.enricher.RemoveMultiplier(request.GetId())
	return &empty.Empty{}, nil
}
