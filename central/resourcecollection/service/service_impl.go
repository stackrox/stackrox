package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.CollectionService/GetCollection",
			// "/v1.CollectionService/GetCollectionCount", TODO ROX-12625
			// "/v1.CollectionService/ListCollections", TODO ROX-12623
			// "/v1.CollectionService/ListCollectionSelectors", TODO ROX-12612
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			// "/v1.CollectionService/AutoCompleteCollection", TODO ROX-12616
			// "/v1.CollectionService/CreateCollection", TODO ROX-12622
			// "/v1.CollectionService/DeleteCollection", TODO ROX-13030
			// "/v1.CollectionService/DryRunCollection", TODO ROX-13031
			// "/v1.CollectionService/UpdateCollection", TODO ROX-13032
		},
	}))
)

// serviceImpl is the struct that manages the collection API
type serviceImpl struct {
	v1.UnimplementedCollectionServiceServer

	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCollectionServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCollectionServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetCollection returns a collection for the given request
func (s *serviceImpl) GetCollection(ctx context.Context, request *v1.GetCollectionRequest) (*v1.GetCollectionResponse, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, nil
	}
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id field should be set when requesting a collection")
	}
	return s.getCollection(ctx, request.Id)
}

func (s *serviceImpl) getCollection(ctx context.Context, id string) (*v1.GetCollectionResponse, error) {
	collection, ok, err := s.datastore.Get(ctx, id)
	if err != nil {
		return nil, errors.Errorf("Could not get collection: %s", err)
	}
	if !ok {
		return nil, errors.Wrap(errox.NotFound, "Not found")
	}

	return &v1.GetCollectionResponse{
		Collection:  collection,
		Deployments: nil,
	}, nil
}
