package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/stringutils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/validation"
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
	datastore datastore.DataStore
	enricher  enrichment.Enricher
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

func normalizeCluster(cluster *v1.Cluster) {
	cluster.CentralApiEndpoint = strings.TrimPrefix(cluster.GetCentralApiEndpoint(), "https://")
	cluster.CentralApiEndpoint = strings.TrimPrefix(cluster.GetCentralApiEndpoint(), "http://")
}

// Validate a field that should adhere to DNS1123 standards,
// and format a helpful error message so the end user
// knows which field to fix.
func validateDNS1123Field(fieldName, value string) error {
	errors := validation.IsDNS1123Label(value)
	if len(errors) == 0 {
		return nil
	}
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("%s validation failed", fieldName))
	errorList.AddStrings(errors...)
	return errorList.ToError()
}

func validateInput(cluster *v1.Cluster) error {
	errorList := errorhelpers.NewErrorList("Cluster Validation")
	if cluster.GetName() == "" {
		errorList.AddString("Cluster name is required")
	}
	if _, err := reference.ParseAnyReference(cluster.GetPreventImage()); err != nil {
		errorList.AddError(fmt.Errorf("invalid prevent image '%s': %s", cluster.GetPreventImage(), err))
	}
	if cluster.GetCentralApiEndpoint() == "" {
		errorList.AddString("Central API Endpoint is required")
	} else if !strings.Contains(cluster.GetCentralApiEndpoint(), ":") {
		errorList.AddString("Central API Endpoint must have port specified")
	}

	if stringutils.ContainsWhitespace(cluster.GetCentralApiEndpoint()) {
		errorList.AddString("Central API endpoint cannot contain whitespace")
	}
	switch orchSpecific := cluster.GetOrchestratorParams().(type) {
	case *v1.Cluster_Kubernetes:
		// Kube validates namespaces and secret names using the DNS1123 Label validator.
		errorList.AddError(validateDNS1123Field("namespace", orchSpecific.Kubernetes.GetParams().GetNamespace()))
	case *v1.Cluster_Openshift:
		errorList.AddError(validateDNS1123Field("namespace", orchSpecific.Openshift.GetParams().GetNamespace()))
	case *v1.Cluster_Swarm:
		if cluster.GetRuntimeSupport() {
			errorList.AddError(errors.New("runtime is not supported with Swarm"))
		}
	}

	return errorList.ToError()
}

// PostCluster creates a new cluster.
func (s *serviceImpl) PostCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new cluster")
	}
	normalizeCluster(request)
	if err := validateInput(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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
	s.enricher.ReprocessRiskAsync()

	return &v1.Empty{}, nil
}
