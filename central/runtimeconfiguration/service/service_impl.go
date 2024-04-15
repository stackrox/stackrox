package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	datastore "github.com/stackrox/rox/central/runtimeconfiguration/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.CollectorRuntimeConfigurationService/GetCollectorRuntimeConfiguration",
			"/v1.CollectorRuntimeConfigurationService/PostCollectorRuntimeConfiguration",
		},
	})
)

type serviceImpl struct {
	dataStore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCollectorRuntimeConfigurationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCollectorRuntimeConfigurationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetCollectorRuntimeConfiguration returns the runtime configuration for collector
func (s *serviceImpl) GetCollectorRuntimeConfiguration(
	ctx context.Context, _ *v1.Empty,
) (*v1.GetCollectorRuntimeConfigurationResponse, error) {

	runtimeFilterRule := storage.RuntimeFilter_RuntimeFilterRule{
		ResourceCollectionId: "abcd",
		Status:               "off",
	}

	rules := []*storage.RuntimeFilter_RuntimeFilterRule{&runtimeFilterRule}

	runtimeFilter := storage.RuntimeFilter{
		Feature:       storage.RuntimeFilter_PROCESSES,
		DefaultStatus: "on",
		Rules:         rules,
	}

	resourceSelector := storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Namespace",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "webapp",
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
		},
	}

	resourceSelectors := []*storage.ResourceSelector{&resourceSelector}

	resourceCollection := storage.ResourceCollection{
		Id:                "abcd",
		Name:              "Fake collection",
		ResourceSelectors: resourceSelectors,
	}

	runtimeFilters := []*storage.RuntimeFilter{&runtimeFilter}
	resourceCollections := []*storage.ResourceCollection{&resourceCollection}

	runtimeFilteringConfiguration := &storage.RuntimeFilteringConfiguration{
		RuntimeFilters:      runtimeFilters,
		ResourceCollections: resourceCollections,
	}

	getCollectorRuntimeConfigurationResponse := v1.GetCollectorRuntimeConfigurationResponse{
		CollectorRuntimeConfiguration: runtimeFilteringConfiguration,
	}

	return &getCollectorRuntimeConfigurationResponse, nil
}

func (s *serviceImpl) PostCollectorRuntimeConfiguration(
	ctx context.Context,
	_ *v1.Empty,
) (*v1.Empty, error) {
	return nil, nil
}
