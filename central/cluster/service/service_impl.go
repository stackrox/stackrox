package service

import (
	"bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/central/clusters"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	datastore datastore.DataStore
	enricher  *enrichment.Enricher
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClustersServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterClustersServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// PostCluster creates a new cluster.
func (s *serviceImpl) PostCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new cluster")
	}
	id, err := s.datastore.AddCluster(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	return s.getCluster(request.GetId())
}

// PutCluster creates a new cluster.
func (s *serviceImpl) PutCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id must be provided")
	}
	if request.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Name must be provided")
	}
	err := s.datastore.UpdateCluster(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return s.getCluster(request.GetId())
}

// GetCluster returns the specified cluster.
func (s *serviceImpl) GetCluster(ctx context.Context, request *v1.ResourceByID) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id must be provided")
	}
	return s.getCluster(request.GetId())
}

func (s *serviceImpl) getCluster(id string) (*v1.ClusterResponse, error) {
	cluster, ok, err := s.datastore.GetCluster(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get cluster: %s", err)
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "Not found")
	}

	deployer, err := clusters.NewDeployer(cluster)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	files, err := deployer.Render(clusters.Wrap(*cluster))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not render all files: %s", err)
	}

	return &v1.ClusterResponse{
		Cluster: cluster,
		Files:   files,
	}, nil
}

// GetClusters returns the currently defined clusters.
func (s *serviceImpl) GetClusters(ctx context.Context, _ *empty.Empty) (*v1.ClustersList, error) {
	clusters, err := s.datastore.GetClusters()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.ClustersList{
		Clusters: clusters,
	}, nil
}

// DeleteCluster removes a cluster
func (s *serviceImpl) DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Request must have a id")
	}
	if err := s.datastore.RemoveCluster(request.GetId()); err != nil {
		return nil, service.ReturnErrorCode(err)
	}

	go func() {
		if err := s.enricher.ReprocessRisk(); err != nil {
			log.Errorf("Error reprocessing risk during cluster removal %#v: %s", request, err)
		}
	}()

	return &empty.Empty{}, nil
}
