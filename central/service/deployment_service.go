package service

import (
	"context"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewDeploymentService returns a DeploymentService object.
func NewDeploymentService(storage db.DeploymentStorage) *DeploymentService {
	return &DeploymentService{
		storage: storage,
	}
}

// DeploymentService provides APIs for deployments.
type DeploymentService struct {
	storage db.DeploymentStorage
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
func (s *DeploymentService) GetDeployments(ctx context.Context, request *v1.GetDeploymentsRequest) (*v1.GetDeploymentsResponse, error) {
	deployments, err := s.storage.GetDeployments(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GetDeploymentsResponse{Deployments: deployments}, nil
}

// GetLabels returns label keys and values for current deployments.
func (s *DeploymentService) GetLabels(context.Context, *empty.Empty) (*v1.DeploymentLabelsResponse, error) {
	deployments, err := s.storage.GetDeployments(&v1.GetDeploymentsRequest{})
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
