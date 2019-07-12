package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	processWhitelistStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	processWhitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxDeploymentsReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			"/v1.DeploymentService/GetDeployment",
			"/v1.DeploymentService/ListDeployments",
			"/v1.DeploymentService/GetLabels",
		},
		user.With(permissions.View(resources.Deployment), permissions.View(resources.ProcessWhitelist), permissions.View(resources.Indicator)): {
			"/v1.DeploymentService/ListDeploymentsWithProcessInfo",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	datastore               datastore.DataStore
	processWhitelists       processWhitelistStore.DataStore
	processIndicators       processIndicatorStore.DataStore
	processWhitelistResults processWhitelistResultsStore.DataStore
	manager                 manager.Manager
}

func (s *serviceImpl) whitelistResultsForDeployment(ctx context.Context, deployment *storage.ListDeployment) (*storage.ProcessWhitelistResults, error) {
	whitelistResults, err := s.processWhitelistResults.GetWhitelistResults(ctx, deployment.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return whitelistResults, nil
}

func (s *serviceImpl) ListDeploymentsWithProcessInfo(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListDeploymentsWithProcessInfoResponse, error) {
	deployments, err := s.ListDeployments(ctx, rawQuery)
	if err != nil {
		return nil, err
	}

	resp := &v1.ListDeploymentsWithProcessInfoResponse{}
	for _, deployment := range deployments.Deployments {
		whitelistResults, err := s.whitelistResultsForDeployment(ctx, deployment)
		if err != nil {
			return nil, err
		}

		var whitelistStatuses []*storage.ContainerNameAndWhitelistStatus
		if whitelistResults != nil {
			whitelistStatuses = whitelistResults.WhitelistStatuses
		}
		resp.Deployments = append(resp.Deployments, &v1.ListDeploymentsWithProcessInfoResponse_DeploymentWithProcessInfo{
			Deployment:        deployment,
			WhitelistStatuses: whitelistStatuses,
		})
	}
	return resp, nil
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
	deployment, exists, err := s.datastore.GetDeployment(ctx, request.GetId())
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
	// Fill in Query.
	parsedQuery, err := search.ParseRawQueryOrEmpty(request.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.Pagination, maxDeploymentsReturned)

	deployments, err := s.datastore.SearchListDeployments(ctx, parsedQuery)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.ListDeploymentsResponse{
		Deployments: deployments,
	}, nil
}

// GetLabels returns label keys and values for current deployments.
func (s *serviceImpl) GetLabels(ctx context.Context, _ *v1.Empty) (*v1.DeploymentLabelsResponse, error) {
	deployments, err := s.datastore.GetAllDeployments(ctx)
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
