package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/vulnerabilityrequest/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	defaultPageSize = 1000
)

var (
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.CollectionService/GetCollection",
			// "/v1.CollectionService/GetCollectionCount", TODO ROX-12625
			"/v1.CollectionService/ListCollections",
			// "/v1.CollectionService/ListCollectionSelectors", TODO ROX-12612
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			// "/v1.CollectionService/AutoCompleteCollection", TODO ROX-12616
			"/v1.CollectionService/CreateCollection",
			"/v1.CollectionService/DeleteCollection",
			// "/v1.CollectionService/DryRunCollection", TODO ROX-13031
			// "/v1.CollectionService/UpdateCollection", TODO ROX-13032
		},
	}))
)

type collectionRequest interface {
	GetName() string
	GetDescription() string
	GetResourceSelectors() []*storage.ResourceSelector
	GetEmbeddedCollectionIds() []string
}

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
		return nil, errors.New("Resource collections is not enabled")
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

// GetCollectionCount returns count of collections matching the query in the request
func (s *serviceImpl) GetCollectionCount(ctx context.Context, request *v1.GetCollectionCountRequest) (*v1.GetCollectionCountResponse, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, errors.New("Resource collections is not enabled")
	}

	// parse query
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	count, err := s.datastore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.GetCollectionCountResponse{Count: int32(count)}, nil
}

// DeleteCollection deletes the collection with the given ID
func (s *serviceImpl) DeleteCollection(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, errors.New("Resource collections is not enabled")
	}
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Non empty collection id must be specified to delete a collection")
	}
	if err := s.datastore.DeleteCollection(ctx, request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// CreateCollection creates a new collection from the given request
func (s *serviceImpl) CreateCollection(ctx context.Context, request *v1.CreateCollectionRequest) (*v1.CreateCollectionResponse, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, errors.New("Resource collections is not enabled")
	}

	collection, err := collectionRequestToCollection(ctx, request, true)
	if err != nil {
		return nil, err
	}

	err = s.datastore.AddCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	return &v1.CreateCollectionResponse{Collection: collection}, nil
}

func (s *serviceImpl) UpdateCollection(ctx context.Context, request *v1.UpdateCollectionRequest) (*v1.UpdateCollectionResponse, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, errors.New("Resource collections is not enabled")
	}

	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Non empty collection id must be specified to delete a collection")
	}

	collection, err := collectionRequestToCollection(ctx, request, false)
	if err != nil {
		return nil, err
	}
	collection.Id = request.GetId()

	err = s.datastore.UpdateCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateCollectionResponse{Collection: collection}, nil
}

func collectionRequestToCollection(ctx context.Context, request collectionRequest, isCreate bool) (*storage.ResourceCollection, error) {
	if request.GetName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Collection Id should not be empty")
	}

	slimUser := utils.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	if len(request.GetResourceSelectors()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "No resource selectors were provided")
	}

	timeNow := protoconv.ConvertTimeToTimestamp(time.Now())

	collection := &storage.ResourceCollection{
		Name:              request.GetName(),
		Description:       request.GetDescription(),
		LastUpdated:       timeNow,
		UpdatedBy:         slimUser,
		ResourceSelectors: request.GetResourceSelectors(),
	}

	if isCreate {
		collection.CreatedBy = slimUser
		collection.CreatedAt = timeNow
	}

	if len(request.GetEmbeddedCollectionIds()) > 0 {
		embeddedCollections := make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(request.GetEmbeddedCollectionIds()))
		for _, id := range request.GetEmbeddedCollectionIds() {
			embeddedCollections = append(embeddedCollections, &storage.ResourceCollection_EmbeddedResourceCollection{Id: id})
		}
		collection.EmbeddedCollections = embeddedCollections
	}

	return collection, nil
}

func (s *serviceImpl) ListCollections(ctx context.Context, request *v1.ListCollectionsRequest) (*v1.ListCollectionsResponse, error) {
	if !features.ObjectCollections.Enabled() {
		return nil, errors.New("Resource collections is not enabled")
	}

	// parse query
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// pagination
	paginated.FillPagination(parsedQuery, request.GetQuery().GetPagination(), defaultPageSize)

	collections, err := s.datastore.SearchCollections(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}

	return &v1.ListCollectionsResponse{
		Collections: collections,
	}, nil
}
