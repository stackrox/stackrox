package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/clusters"
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewClusterService returns the ClusterService API.
func NewClusterService(storage db.Storage) *ClusterService {
	return &ClusterService{
		storage: storage,
	}
}

// ClusterService is the struct that manages the cluster API
type ClusterService struct {
	storage db.ClusterStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *ClusterService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClustersServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *ClusterService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterClustersServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// PostCluster creates a new cluster.
func (s *ClusterService) PostCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new cluster")
	}
	id, err := s.storage.AddCluster(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	return s.getCluster(request.GetId())
}

// PutCluster creates a new cluster.
func (s *ClusterService) PutCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id must be provided")
	}
	if request.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Name must be provided")
	}
	err := s.storage.UpdateCluster(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return s.getCluster(request.GetId())
}

// GetCluster returns the specified cluster.
func (s *ClusterService) GetCluster(ctx context.Context, request *v1.ResourceByID) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id must be provided")
	}
	return s.getCluster(request.GetId())
}

func (s *ClusterService) getCluster(id string) (*v1.ClusterResponse, error) {
	cluster, ok, err := s.storage.GetCluster(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get cluster: %s", err)
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "Not found")
	}
	dep, err := clusters.Wrap(*cluster).Deployment()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create deployment YAML: %s", err)
	}
	cmd, err := clusters.Wrap(*cluster).Command()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create deployment command: %s", err)
	}
	return &v1.ClusterResponse{
		Cluster:           cluster,
		DeploymentYaml:    dep,
		DeploymentCommand: cmd,
	}, nil
}

// GetClusters returns the currently defined clusters.
func (s *ClusterService) GetClusters(ctx context.Context, _ *empty.Empty) (*v1.ClustersList, error) {
	clusters, err := s.storage.GetClusters()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.ClustersList{
		Clusters: clusters,
	}, nil
}

// DeleteCluster removes a cluster
func (s *ClusterService) DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Request must have a id")
	}
	err := s.storage.RemoveCluster(request.GetId())
	if err != nil {
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}
