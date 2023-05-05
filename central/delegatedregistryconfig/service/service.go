package service

import (
	"context"

	cluster "github.com/stackrox/rox/central/cluster/datastore"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log        = logging.LoggerForModule()
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
func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*storage.DelegatedRegistryConfig, error) {
	if s.dataStore == nil {
		return nil, status.Errorf(codes.Unimplemented, "datastore not initialized, is postgres enabled?")
	}

	config, err := s.dataStore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return &storage.DelegatedRegistryConfig{}, nil
	}

	return config, nil
}

// GetClusters returns the list of clusters (id + name) that are eligible for delegating scanning
// requests (ie: only clusters with scanners that understand the delegated registry config)
func (s *serviceImpl) GetClusters(ctx context.Context, _ *v1.Empty) (*v1.DelegatedRegistryClustersResponse, error) {
	clusters, err := s.getClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving clusters")
	}

	if len(clusters) == 0 {
		return nil, status.Errorf(codes.NotFound, "no valid clusters found")
	}

	return &v1.DelegatedRegistryClustersResponse{
		Clusters: clusters,
	}, nil
}

// PutConfig updates Central's delegated registry config
func (s *serviceImpl) PutConfig(ctx context.Context, config *storage.DelegatedRegistryConfig) (*storage.DelegatedRegistryConfig, error) {
	if s.dataStore == nil {
		return nil, status.Errorf(codes.Unimplemented, "datastore not initialized, is postgres enabled?")
	}

	log.Debugf("PutConfig %T [%+v]", config, *config)

	if err := s.validate(config); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	if err := s.dataStore.UpsertConfig(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

func (s *serviceImpl) validate(config *storage.DelegatedRegistryConfig) error {
	if config == nil {
		// this block not reachable via GRPC-gateway invocations FYI
		return errors.New("config missing")
	}

	errorList := errorhelpers.NewErrorList("Validation")
	if config.EnabledFor != storage.DelegatedRegistryConfig_NONE {
		// a default cluster is required if EnabledFor is set to something other than NONE
		if config.DefaultClusterId == "" {
			errorList.AddStrings("defaultClusterId required if enabledFor != NONE")
		}
	}

	// TODO: also check if cluster IDs are valid
	// TODO: also check if only clusters that have scanner and delegated scanning capability are used

	return errorList.ToError()
}

func (s *serviceImpl) getClusters(ctx context.Context) ([]*v1.DelegatedRegistryCluster, error) {
	clusters, err := s.clusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving clusters")
	}

	if len(clusters) == 0 {
		return nil, nil
	}

	var res []*v1.DelegatedRegistryCluster
	for _, c := range clusters {
		res = append(res, &v1.DelegatedRegistryCluster{
			Id:      c.Id,
			Name:    c.Name,
			IsValid: true,
		})
	}

	return res, nil
}
