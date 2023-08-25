package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	cluster "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/convert"
	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	deleConnection "github.com/stackrox/rox/central/delegatedregistryconfig/util/connection"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.DelegatedRegistryConfigService/GetConfig",
			"/v1.DelegatedRegistryConfigService/GetClusters",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.DelegatedRegistryConfigService/UpdateConfig",
		},
	})
)

// Service provides the interface to modify the delegated registry config.
type Service interface {
	pkgGRPC.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DelegatedRegistryConfigServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(dataStore datastore.DataStore, clusterDataStore cluster.DataStore, connManager connection.Manager) Service {
	return &serviceImpl{
		dataStore:        dataStore,
		clusterDataStore: clusterDataStore,
		connManager:      connManager,
	}
}

type serviceImpl struct {
	v1.UnimplementedDelegatedRegistryConfigServiceServer

	dataStore        datastore.DataStore
	clusterDataStore cluster.DataStore
	connManager      connection.Manager
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

// GetConfig returns Central's delegated registry config.
func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*v1.DelegatedRegistryConfig, error) {
	config, exists, err := s.dataStore.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving config %w", err)
	}

	if !exists {
		return &v1.DelegatedRegistryConfig{}, nil
	}

	return convert.StorageToPublicAPI(config), nil
}

// GetClusters returns the list of all clusters (id + name + valid flag). The valid flag indicates that
// Central can delegate registry interactions (scanning, signature verification, etc.) to that cluster
// and therefore that cluster is valid for use in the DelegatedRegistryConfig.
func (s *serviceImpl) GetClusters(ctx context.Context, _ *v1.Empty) (*v1.DelegatedRegistryClustersResponse, error) {
	clusters, err := s.getClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving clusters %w", err)
	}

	if len(clusters) == 0 {
		return nil, status.Error(codes.NotFound, "no clusters found")
	}

	return &v1.DelegatedRegistryClustersResponse{
		Clusters: clusters,
	}, nil
}

// UpdateConfig updates Central's delegated registry config.
func (s *serviceImpl) UpdateConfig(ctx context.Context, config *v1.DelegatedRegistryConfig) (*v1.DelegatedRegistryConfig, error) {
	if config == nil {
		return nil, errox.InvalidArgs.CausedBy("config missing")
	}

	// get the clusters ids for validation and broadcast
	clusterIDs, err := s.getValidClusterIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtaining valid cluster %w", err)
	}

	// validate the config
	if err := s.validate(config, clusterIDs); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err.Error())
	}

	// persist the config
	if err := s.dataStore.UpsertConfig(ctx, convert.PublicAPIToStorage(config)); err != nil {
		return nil, fmt.Errorf("upserting config %w", err)
	}

	// broadcast the config
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_DelegatedRegistryConfig{
			DelegatedRegistryConfig: convert.PublicAPIToInternalAPI(config),
		},
	}

	log.Infof("Delegated registry config updated: %q", config)
	for clusterID := range clusterIDs {
		log.Debugf("Sending updated delegated registry config to cluster %q", clusterID)
		if err := s.connManager.SendMessage(clusterID, msg); err != nil {
			log.Errorf("Failed to send updated delegated registry config to cluster %q: %v", clusterID, err)
		}
	}

	return config, nil
}

func (s *serviceImpl) validate(config *v1.DelegatedRegistryConfig, validClusters set.Set[string]) error {
	if config.EnabledFor == v1.DelegatedRegistryConfig_NONE {
		// ignore rest of config, values will not be used
		return nil
	}

	var errorList []error
	if config.DefaultClusterId != "" && !validClusters.Contains(config.DefaultClusterId) {
		errorList = append(errorList, fmt.Errorf("default cluster %q is not valid", config.DefaultClusterId))
	}

	// validate the registries / clusters
	for _, r := range config.Registries {

		// if a cluster id was provided, check if its valid
		if r.ClusterId != "" && !validClusters.Contains(r.ClusterId) {
			errorList = append(errorList, fmt.Errorf("cluster %q is not valid", r.ClusterId))
		}

		if r.Path == "" {
			errorList = append(errorList, errors.New("missing registry path"))
		}
	}

	return errors.Join(errorList...)
}

// getClusters returns all clusters, the clusters with valid set to true can be used as in a
// DelegatedRegistryConfig. All clusters are returned instead of just valid clusters
// so that a consumer (ie: the UI) can show the friendly name of clusters that may no longer
// be valid (but once were).
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
		conn := s.connManager.GetConnection(c.GetId())

		res[i] = &v1.DelegatedRegistryCluster{
			Id:      c.Id,
			Name:    c.Name,
			IsValid: deleConnection.ValidForDelegation(conn),
		}
	}

	return res, nil
}

// getValidClusterIDs returns a set of cluster ids that are valid for use in a DelegatedRegistryConfig.
func (s *serviceImpl) getValidClusterIDs(ctx context.Context) (set.Set[string], error) {
	clusters, err := s.getClusters(ctx)
	if err != nil {
		return nil, err
	}

	validClusterIDs := set.NewStringSet()
	for _, c := range clusters {
		if c.IsValid {
			validClusterIDs.Add(c.Id)
		}
	}

	return validClusterIDs, nil
}
