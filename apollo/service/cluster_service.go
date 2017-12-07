package service

import (
	"bytes"
	"text/template"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/env"
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
func (s *ClusterService) PostCluster(ctx context.Context, request *v1.Cluster) (*empty.Empty, error) {
	err := s.storage.AddCluster(request)
	return &empty.Empty{}, err
}

// PutCluster creates a new cluster.
func (s *ClusterService) PutCluster(ctx context.Context, request *v1.Cluster) (*empty.Empty, error) {
	err := s.storage.UpdateCluster(request)
	return &empty.Empty{}, err
}

// GetCluster returns the specified cluster.
func (s *ClusterService) GetCluster(ctx context.Context, request *v1.ClusterByName) (*v1.ClusterResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "Name must be provided")
	}
	cluster, ok, err := s.storage.GetCluster(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get cluster: %s", err)
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "Not found")
	}
	dep, err := clusterWrap(*cluster).asDeployment()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create deployment YAML: %s", err)
	}
	return &v1.ClusterResponse{
		Cluster:        cluster,
		DeploymentYaml: dep,
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
func (s *ClusterService) DeleteCluster(ctx context.Context, request *v1.ClusterByName) (*empty.Empty, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "Request must have a name")
	}
	err := s.storage.RemoveCluster(request.Name)
	return &empty.Empty{}, status.Error(codes.Internal, err.Error())
}

type clusterWrap v1.Cluster

var swarmTemplate *template.Template

func init() {
	var err error
	swarmTemplate, err = template.New("base").Parse(`version: "3.2"
services:
  agent:
    image: {{.Image}}
    entrypoint:
      - swarm-agent
    networks:
      net:
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
    deploy:
      placement:
        constraints:
          - node.role==manager
    environment:
      - "{{.PublicEndpointEnv}}={{.PublicEndpoint}}"
      - "{{.ClusterNameEnv}}={{.ClusterName}}"
      - "{{.AdvertisedEndpointEnv}}={{.AdvertisedEndpoint}}"
      - "{{.ImageEnv}}={{.Image}}"
networks:
  net:
    driver: overlay
    attachable: true
`)
	if err != nil {
		panic(err)
	}
}

func (c clusterWrap) asDeployment() (string, error) {
	switch c.Type {
	case v1.ClusterType_DOCKER_EE_CLUSTER:
		var b []byte
		buf := bytes.NewBuffer(b)
		err := swarmTemplate.Execute(buf, map[string]string{
			"ImageEnv":              env.Image.EnvVar(),
			"Image":                 c.ApolloImage,
			"PublicEndpointEnv":     env.ApolloEndpoint.EnvVar(),
			"PublicEndpoint":        c.CentralApiEndpoint,
			"ClusterNameEnv":        env.ClusterID.EnvVar(),
			"ClusterName":           c.Name,
			"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
			"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),
		})
		if err != nil {
			return "", err
		}
		return buf.String(), nil
	default:
		return "", status.Error(codes.Unimplemented, "Only Swarm deployments can be described currently")
	}
}
