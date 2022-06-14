package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/probesources"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Cluster)): {
			"/v1.ClustersService/GetClusters",
			"/v1.ClustersService/GetCluster",
			"/v1.ClustersService/GetKernelSupportAvailable",
			"/v1.ClustersService/GetClusterDefaultValues",
		},
		user.With(permissions.Modify(resources.Cluster)): {
			"/v1.ClustersService/PostCluster",
			"/v1.ClustersService/PutCluster",
			"/v1.ClustersService/DeleteCluster",
		},
	}))
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	datastore    datastore.DataStore
	riskManager  manager.Manager
	probeSources probesources.ProbeSources
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClustersServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterClustersServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// PostCluster creates a new cluster.
func (s *serviceImpl) PostCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id field should be empty when posting a new cluster")
	}
	id, err := s.datastore.AddCluster(ctx, request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	return s.getCluster(ctx, request.GetId())
}

// PutCluster updates an existing cluster.
func (s *serviceImpl) PutCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id must be provided")
	}
	err := s.datastore.UpdateCluster(ctx, request)
	if err != nil {
		return nil, err
	}
	return s.getCluster(ctx, request.GetId())
}

// GetCluster returns the specified cluster.
func (s *serviceImpl) GetCluster(ctx context.Context, request *v1.ResourceByID) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id must be provided")
	}
	return s.getCluster(ctx, request.GetId())
}

func (s *serviceImpl) getCluster(ctx context.Context, id string) (*v1.ClusterResponse, error) {
	cluster, ok, err := s.datastore.GetCluster(ctx, id)
	if err != nil {
		return nil, errors.Errorf("Could not get cluster: %s", err)
	}
	if !ok {
		return nil, errors.Wrap(errox.NotFound, "Not found")
	}

	return &v1.ClusterResponse{
		Cluster: cluster,
	}, nil
}

// GetClusters returns the currently defined clusters.
func (s *serviceImpl) GetClusters(ctx context.Context, req *v1.GetClustersRequest) (*v1.ClustersList, error) {
	q, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "invalid query %q: %v", req.GetQuery(), err)
	}

	clusters, err := s.datastore.SearchRawClusters(ctx, q)
	if err != nil {
		return nil, err
	}
	return &v1.ClustersList{
		Clusters: clusters,
	}, nil
}

// DeleteCluster removes a cluster
func (s *serviceImpl) DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Request must have a id")
	}
	if err := s.datastore.RemoveCluster(ctx, request.GetId(), nil); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// Deprecated: Use GetClusterDefaultValues instead.
func (s *serviceImpl) GetKernelSupportAvailable(ctx context.Context, _ *v1.Empty) (*v1.KernelSupportAvailableResponse, error) {
	anyAvailable, err := s.probeSources.AnyAvailable(ctx)
	if err != nil {
		return nil, err
	}
	result := &v1.KernelSupportAvailableResponse{
		KernelSupportAvailable: anyAvailable,
	}
	return result, nil
}

func (s *serviceImpl) GetClusterDefaultValues(ctx context.Context, _ *v1.Empty) (*v1.ClusterDefaultsResponse, error) {
	kernelSupport, err := s.probeSources.AnyAvailable(ctx)
	if err != nil {
		return nil, err
	}
	flavor := defaults.GetImageFlavorFromEnv()
	defaults := &v1.ClusterDefaultsResponse{
		MainImageRepository:      flavor.MainImageNoTag(),
		CollectorImageRepository: flavor.CollectorFullImageNoTag(),
		KernelSupportAvailable:   kernelSupport,
	}
	return defaults, nil
}
