package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
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
			"/v1.CollectionService/CreateCollection",
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

// CreateCollection creates a new collection from the given request
func (s *serviceImpl) CreateCollection(ctx context.Context, request *v1.CreateCollectionRequest) (*v1.CreateCollectionResponse, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, nil
	}

	if request.GetName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Collection name should not be empty")
	}
	// check if collection with same name doesn't already exist
	nameQuery := search.NewQueryBuilder().AddExactMatches(search.CollectionName, request.GetName()).ProtoQuery()
	c, err := s.datastore.Count(ctx, nameQuery)
	if err != nil {
		return nil, err
	}
	if c != 0 {
		return nil, errors.Wrap(errox.AlreadyExists, "A collection with that name already exists")
	}

	creator := extractUserIdentity(ctx)
	if creator == nil {
		return nil, errors.New("User identity not provided")
	}

	collection := &storage.ResourceCollection{
		Id:                uuid.NewV4().String(),
		Name:              request.GetName(),
		Description:       request.GetDescription(),
		CreatedAt:         protoconv.ConvertTimeToTimestamp(time.Now()),
		CreatedBy:         creator,
		ResourceSelectors: request.GetResourceSelectors(),
	}

	if len(request.GetEmbeddedCollectionIds()) > 0 {
		embeddedCollections := make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(request.GetEmbeddedCollectionIds()))
		for _, id := range request.GetEmbeddedCollectionIds() {
			embeddedCollections = append(embeddedCollections, &storage.ResourceCollection_EmbeddedResourceCollection{Id: id})
		}
		collection.EmbeddedCollections = embeddedCollections
	}

	err = s.datastore.AddCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	return &v1.CreateCollectionResponse{Collection: collection}, nil
}

func extractUserIdentity(ctx context.Context) *storage.SlimUser {
	ctxIdentity := authn.IdentityFromContextOrNil(ctx)
	if ctxIdentity == nil {
		return nil
	}

	return &storage.SlimUser{
		Id:   ctxIdentity.UID(),
		Name: ctxIdentity.FullName(),
	}
}
