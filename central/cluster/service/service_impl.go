package service

import (
	"io/ioutil"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/monitoring"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Cluster)): {
			"/v1.ClustersService/GetClusters",
			"/v1.ClustersService/GetCluster",
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
	datastore   datastore.DataStore
	riskManager manager.Manager
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

func normalizeCluster(cluster *storage.Cluster) {
	cluster.CentralApiEndpoint = strings.TrimPrefix(cluster.GetCentralApiEndpoint(), "https://")
	cluster.CentralApiEndpoint = strings.TrimPrefix(cluster.GetCentralApiEndpoint(), "http://")
}

func validateInput(cluster *storage.Cluster) error {
	errorList := clusterValidation.Validate(cluster)
	if cluster.GetMonitoringEndpoint() != "" {
		// Purposefully not checking the CAPath because only one is needed for an indication of monitoring not being enabled
		if _, err := ioutil.ReadFile(monitoring.CAPath); err != nil {
			errorList.AddString("Could not read monitoring CA. Continue by removing the monitoring endpoint")
		}
	}

	return errorList.ToError()
}

// PostCluster creates a new cluster.
func (s *serviceImpl) PostCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new cluster")
	}
	normalizeCluster(request)
	if err := validateInput(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := s.datastore.AddCluster(request)
	if err != nil {
		if errWrap, ok := err.(*errorhelpers.ErrorWrap); ok && errWrap.Type == errorhelpers.ErrAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	return s.getCluster(request.GetId())
}

// PutCluster creates a new cluster.
func (s *serviceImpl) PutCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id must be provided")
	}
	normalizeCluster(request)
	if err := validateInput(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	return &v1.ClusterResponse{
		Cluster: cluster,
	}, nil
}

// GetClusters returns the currently defined clusters.
func (s *serviceImpl) GetClusters(ctx context.Context, _ *v1.Empty) (*v1.ClustersList, error) {
	clusters, err := s.datastore.GetClusters()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.ClustersList{
		Clusters: clusters,
	}, nil
}

// DeleteCluster removes a cluster
func (s *serviceImpl) DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Request must have a id")
	}
	if err := s.datastore.RemoveCluster(request.GetId()); err != nil {
		return nil, err
	}
	s.riskManager.ReprocessRisk()
	return &v1.Empty{}, nil
}
