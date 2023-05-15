package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	cluster "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/convert"
	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		or.SensorOrAuthorizer(user.With(permissions.View(resources.Administration))): {
			"/v1.DelegatedRegistryConfigService/GetConfig",
		},
		user.With(permissions.View(resources.Administration)): {
			"/v1.DelegatedRegistryConfigService/GetClusters",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.DelegatedRegistryConfigService/PutConfig",
		},
	})
)

// Service provides the interface to modify the delegated registry config
type Service interface {
	pkgGRPC.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DelegatedRegistryConfigServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(dataStore datastore.DataStore, clusterDataStore cluster.DataStore) Service {
	return &serviceImpl{
		dataStore:        dataStore,
		clusterDataStore: clusterDataStore,
	}
}

type serviceImpl struct {
	v1.UnimplementedDelegatedRegistryConfigServiceServer

	dataStore        datastore.DataStore
	clusterDataStore cluster.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDelegatedRegistryConfigServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDelegatedRegistryConfigServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetConfig returns Central's delegated registry config
func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*v1.DelegatedRegistryConfig, error) {
	config, exists, err := s.dataStore.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving config %w", err)
	}

	if !exists {
		return &v1.DelegatedRegistryConfig{}, nil
	}

	return convert.StorageToAPI(config), nil
}

// GetClusters returns the list of clusters (id + name) that are eligible for delegating scanning
// requests (ie: only clusters with scanners that understand the delegated registry config)
func (s *serviceImpl) GetClusters(ctx context.Context, _ *v1.Empty) (*v1.DelegatedRegistryClustersResponse, error) {
	clusters, err := s.getClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving clusters %w", err)
	}

	if len(clusters) == 0 {
		return nil, status.Error(codes.NotFound, "no valid clusters found")
	}

	return &v1.DelegatedRegistryClustersResponse{
		Clusters: clusters,
	}, nil
}

// PutConfig updates Central's delegated registry config
func (s *serviceImpl) PutConfig(ctx context.Context, config *v1.DelegatedRegistryConfig) (*v1.DelegatedRegistryConfig, error) {
	if err := s.validate(ctx, config); err != nil {
		return nil, fmt.Errorf("%w: %v", errox.InvalidArgs, err.Error())
	}

	if err := s.dataStore.UpsertConfig(ctx, convert.APIToStorage(config)); err != nil {
		return nil, fmt.Errorf("upserting config %w", err)
	}

	return config, nil
}

func (s *serviceImpl) validate(ctx context.Context, config *v1.DelegatedRegistryConfig) error {
	if config == nil {
		// this block not reachable via GRPC-gateway invocations
		return errors.New("config missing")
	}

	if config.EnabledFor == v1.DelegatedRegistryConfig_NONE {
		// ignore rest of config
		return nil
	}

	// validate the default cluster id
	if config.DefaultClusterId == "" {
		return errors.New("default cluster id required if enabled is not NONE")
	}

	// get the valid clusters
	validClusters, err := s.getValidClusterIds(ctx)
	if err != nil {
		return fmt.Errorf("unable to validate config %w", err)
	}

	var errorList []error
	if !validClusters.Contains(config.DefaultClusterId) {
		errorList = append(errorList, fmt.Errorf("default cluster %q is not valid", config.DefaultClusterId))
	}

	// validate the registries / clusters
	for _, r := range config.Registries {

		// if a cluster id was provided, check if its valid
		if r.ClusterId != "" && !validClusters.Contains(r.ClusterId) {
			errorList = append(errorList, fmt.Errorf("cluster %q is not valid", r.ClusterId))
		}

		if r.RegistryPath == "" {
			errorList = append(errorList, errors.New("missing registry path"))
		}
	}

	return errors.Join(errorList...)
}

// getClusters returns all clusters annotated with a flag indicating if cluster is valid
// for use as a delegation target. All clusters are returned instead of just valid clusters
// so that a consumer (ie: the UI) can show the friendly name of clusters that may no longer
// be valid but once were
func (s *serviceImpl) getClusters(ctx context.Context) ([]*v1.DelegatedRegistryCluster, error) {
	clusters, err := s.clusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return nil, nil
	}

	res := make([]*v1.DelegatedRegistryCluster, len(clusters))
	for i, c := range clusters {

		// TODO (ROX-16970): change this to use sensor capability instead
		valid := c.GetHealthStatus().GetScannerHealthStatus() == storage.ClusterHealthStatus_HEALTHY

		res[i] = &v1.DelegatedRegistryCluster{
			Id:      c.Id,
			Name:    c.Name,
			IsValid: valid,
		}
	}

	return res, nil
}

// getValidClusterIds returns a set cluster ids that are valid for delegation
func (s *serviceImpl) getValidClusterIds(ctx context.Context) (set.Set[string], error) {
	clusters, err := s.getClusters(ctx)
	if err != nil {
		return nil, err
	}

	validClusterIds := set.NewStringSet()
	for _, c := range clusters {
		if c.IsValid {
			validClusterIds.Add(c.Id)
		}
	}

	return validClusterIds, nil
}
