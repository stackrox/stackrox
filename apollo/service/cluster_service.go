package service

import (
	"bytes"
	"text/template"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
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

type clusterWrap v1.Cluster

func (c clusterWrap) asDeployment() (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)

	if _, ok := clusterTypeTemplates[c.Type]; !ok {
		return "", status.Errorf(codes.Unimplemented, "Cluster type %s is not currently implemented", c.Type.String())
	}

	t := clusterTypeTemplates[c.Type]
	fields := c.commonFields()

	switch c.Type {
	case v1.ClusterType_KUBERNETES_CLUSTER:
		namespace := "default"
		if len(c.Namespace) != 0 {
			namespace = c.Namespace
		}
		fields["Namespace"] = namespace
		fields["ImagePullSecretEnv"] = env.ImagePullSecrets.EnvVar()
		fields["ImagePullSecret"] = c.ImagePullSecret
	}

	err := t.Execute(buf, fields)
	if err != nil {
		log.Errorf("%s deployment template execution: %s", c.Type.String(), err)
		return "", err
	}

	return buf.String(), nil
}

func (c clusterWrap) commonFields() map[string]string {
	return map[string]string{
		"ImageEnv":              env.Image.EnvVar(),
		"Image":                 c.ApolloImage,
		"PublicEndpointEnv":     env.ApolloEndpoint.EnvVar(),
		"PublicEndpoint":        c.CentralApiEndpoint,
		"ClusterNameEnv":        env.ClusterID.EnvVar(),
		"ClusterName":           c.Name,
		"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
		"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),
	}
}

var (
	clusterTypeTemplates = map[v1.ClusterType]*template.Template{}
)

func init() {
	var err error
	clusterTypeTemplates[v1.ClusterType_DOCKER_EE_CLUSTER], err = template.New("base").Parse(`version: "3.2"
services:
  sensor:
    image: {{.Image}}
    entrypoint:
      - swarm-sensor
    networks:
      net:
    deploy:
      placement:
        constraints:
          - node.role==manager
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
    environment:
      - "{{.PublicEndpointEnv}}={{.PublicEndpoint}}"
      - "{{.ClusterNameEnv}}={{.ClusterName}}"
      - "{{.AdvertisedEndpointEnv}}={{.AdvertisedEndpoint}}"
      - "{{.ImageEnv}}={{.Image}}"
    secrets:
      - source: sensor_certificate
        target: stackrox.io/cert.pem
        mode: 400
      - source: sensor_private_key
        target: stackrox.io/key.pem
        mode: 400
      - source: central_certificate
        target: stackrox.io/ca.pem
        mode: 400
networks:
  net:
    driver: overlay
    attachable: true
secrets:
  sensor_private_key:
    file: sensor-{{.ClusterName}}-key.pem
  sensor_certificate:
    file: sensor-{{.ClusterName}}-cert.pem
  central_certificate:
    file: central-ca.pem
`)
	// TODO(cg): Do we need to include DOCKER_HOST, DOCKER_CERT_PATH, DOCKER_TLS_VERIFY?
	if err != nil {
		panic(err)
	}

	// Swarm is an alias of Docker EE for the purposes of cluster configuration.
	clusterTypeTemplates[v1.ClusterType_SWARM_CLUSTER] = clusterTypeTemplates[v1.ClusterType_DOCKER_EE_CLUSTER]

	clusterTypeTemplates[v1.ClusterType_KUBERNETES_CLUSTER], err = template.New("base").Parse(`apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: sensor
  namespace: {{.Namespace}}
  labels:
    app: sensor
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sensor
  template:
    metadata:
      namespace: {{.Namespace}}
      labels:
        app: sensor
    spec:
      containers:
      - image: {{.Image}}
        env:
        - name: {{.PublicEndpointEnv}}
          value: {{.PublicEndpoint}}
        - name: {{.ClusterNameEnv}}
          value: {{.ClusterName}}
        - name: {{.ImageEnv}}
          value: {{.Image}}
        - name: {{.AdvertisedEndpointEnv}}
          value: {{.AdvertisedEndpoint}}
        - name: {{.ImagePullSecretEnv}}
          value: {{.ImagePullSecret}}
        - name: ROX_APOLLO_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: ROX_APOLLO_SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        imagePullPolicy: Always
        name: sensor
        command:
        - kubernetes-sensor
        volumeMounts:
        - name: certs
          mountPath: /run/secrets/stackrox.io/
          readOnly: true
      imagePullSecrets:
      - name: {{.ImagePullSecret}}
      volumes:
      - name: certs
        secret:
          secretName: sensor-tls
          items:
          - key: sensor-{{.ClusterName}}-cert.pem
            path: cert.pem
          - key: sensor-{{.ClusterName}}-key.pem
            path: key.pem
          - key: central-ca.pem
            path: ca.pem
---
apiVersion: v1
kind: Service
metadata:
  name: sensor
  namespace: {{.Namespace}}
spec:
  ports:
  - name: https
    port: 443
    targetPort: 443
  selector:
    app: sensor
  type: ClusterIP`)

	if err != nil {
		panic(err)
	}
}
