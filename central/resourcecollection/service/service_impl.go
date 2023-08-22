package service

import (
	"context"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	defaultPageSize = 1000
)

var (
	authorizer = or.SensorOr(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.CollectionService/GetCollection",
			"/v1.CollectionService/GetCollectionCount",
			"/v1.CollectionService/ListCollections",
			"/v1.CollectionService/ListCollectionSelectors",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			// "/v1.CollectionService/AutoCompleteCollection", TODO ROX-12616
			"/v1.CollectionService/CreateCollection",
			"/v1.CollectionService/DeleteCollection",
			"/v1.CollectionService/UpdateCollection",
			"/v1.CollectionService/DryRunCollection",
		},
	}))
	defaultCollectionSortOption = &v1.QuerySortOption{
		Field:    search.CollectionName.String(),
		Reversed: false,
	}
	defaultDeploymentSortOption = &v1.QuerySortOption{
		Field:    search.DeploymentName.String(),
		Reversed: false,
	}
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

	datastore             datastore.DataStore
	queryResolver         datastore.QueryResolver
	deploymentDS          deploymentDS.DataStore
	reportConfigDatastore reportConfigDS.DataStore
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

// ListCollectionSelectors returns all supported selectors
func (s *serviceImpl) ListCollectionSelectors(_ context.Context, _ *v1.Empty) (*v1.ListCollectionSelectorsResponse, error) {
	selectors := datastore.GetSupportedFieldLabels()
	selectorStrings := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		selectorStrings = append(selectorStrings, selector.String())
	}
	return &v1.ListCollectionSelectorsResponse{
		Selectors: selectorStrings,
	}, nil
}

// GetCollection returns a collection for the given request
func (s *serviceImpl) GetCollection(ctx context.Context, request *v1.GetCollectionRequest) (*v1.GetCollectionResponse, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Id should be set when requesting a collection")
	}

	collection, exists, err := s.datastore.Get(ctx, request.GetId())
	if err != nil {
		return nil, errors.Errorf("Could not get collection: %s", err)
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "collection with id %q does not exist", request.GetId())
	}

	deployments, err := s.tryDeploymentMatching(ctx, collection, request.GetOptions())
	if err != nil {
		return nil, errors.Wrap(err, "failed resolving deployments")
	}

	return &v1.GetCollectionResponse{
		Collection:  collection,
		Deployments: deployments,
	}, nil
}

// GetCollectionCount returns count of collections matching the query in the request
func (s *serviceImpl) GetCollectionCount(ctx context.Context, request *v1.GetCollectionCountRequest) (*v1.GetCollectionCountResponse, error) {
	query, err := resolveQuery(request.GetQuery(), false)
	if err != nil {
		return nil, err
	}

	count, err := s.datastore.Count(ctx, query)
	if err != nil {
		return nil, err
	}
	return &v1.GetCollectionCountResponse{Count: int32(count)}, nil
}

// DeleteCollection deletes the collection with the given ID
func (s *serviceImpl) DeleteCollection(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Non empty collection id must be specified to delete a collection")
	}

	// error out if collection is in use by a report config
	query := search.DisjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.EmbeddedCollectionID, request.GetId()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.CollectionID, request.GetId()).ProtoQuery(),
	)
	reportConfigCount, err := s.reportConfigDatastore.Count(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to check for Report Configuration usages")
	}
	if reportConfigCount != 0 {
		return nil, errors.Wrap(errox.ReferencedByAnotherObject, "Collection is in use by one or more report configurations")
	}

	if err := s.datastore.DeleteCollection(ctx, request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// CreateCollection creates a new collection from the given request
func (s *serviceImpl) CreateCollection(ctx context.Context, request *v1.CreateCollectionRequest) (*v1.CreateCollectionResponse, error) {
	collection, err := collectionRequestToCollection(ctx, request, "")
	if err != nil {
		return nil, err
	}

	_, err = s.datastore.AddCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	return &v1.CreateCollectionResponse{Collection: collection}, nil
}

func (s *serviceImpl) UpdateCollection(ctx context.Context, request *v1.UpdateCollectionRequest) (*v1.UpdateCollectionResponse, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Non empty collection id must be specified to update a collection")
	}

	collection, err := collectionRequestToCollection(ctx, request, request.GetId())
	if err != nil {
		return nil, err
	}

	err = s.datastore.UpdateCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateCollectionResponse{Collection: collection}, nil
}

func collectionRequestToCollection(ctx context.Context, request collectionRequest, id string) (*storage.ResourceCollection, error) {
	collectionName := strings.TrimSpace(request.GetName())
	if collectionName == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Collection name should not be empty")
	}

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	if len(request.GetResourceSelectors())+len(request.GetEmbeddedCollectionIds()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "No resource selectors or embedded collections were provided")
	}

	timeNow := protoconv.ConvertTimeToTimestamp(time.Now())

	collection := &storage.ResourceCollection{
		Id:                id,
		Name:              collectionName,
		Description:       request.GetDescription(),
		LastUpdated:       timeNow,
		UpdatedBy:         slimUser,
		ResourceSelectors: request.GetResourceSelectors(),
	}

	if id == "" {
		// new  collection
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

func resolveQuery(rawQuery *v1.RawQuery, withPagination bool) (*v1.Query, error) {
	query, err := search.ParseQuery(rawQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	if withPagination {
		paginated.FillPagination(query, rawQuery.GetPagination(), defaultPageSize)
		paginated.FillDefaultSortOption(query, defaultCollectionSortOption)
	}
	return query, nil
}

func (s *serviceImpl) ListCollections(ctx context.Context, request *v1.ListCollectionsRequest) (*v1.ListCollectionsResponse, error) {
	query, err := resolveQuery(request.GetQuery(), true)
	if err != nil {
		return nil, err
	}

	collections, err := s.datastore.SearchCollections(ctx, query)
	if err != nil {
		return nil, err
	}

	return &v1.ListCollectionsResponse{
		Collections: collections,
	}, nil
}

func (s *serviceImpl) DryRunCollection(ctx context.Context, request *v1.DryRunCollectionRequest) (*v1.DryRunCollectionResponse, error) {
	collection, err := collectionRequestToCollection(ctx, request, request.GetId())
	if err != nil {
		return nil, err
	}

	if request.GetId() == "" {
		err = s.datastore.DryRunAddCollection(ctx, collection)
	} else {
		err = s.datastore.DryRunUpdateCollection(ctx, collection)
	}
	if err != nil {
		return nil, err
	}

	deployments, err := s.tryDeploymentMatching(ctx, collection, request.GetOptions())
	if err != nil {
		return nil, errors.Wrap(err, "failed resolving deployments")
	}

	return &v1.DryRunCollectionResponse{
		Deployments: deployments,
	}, nil
}

func (s *serviceImpl) tryDeploymentMatching(ctx context.Context, collection *storage.ResourceCollection, matchOptions *v1.CollectionDeploymentMatchOptions) ([]*storage.ListDeployment, error) {
	if matchOptions == nil || !matchOptions.GetWithMatches() {
		return nil, nil
	}

	collectionQuery, err := s.queryResolver.ResolveCollectionQuery(ctx, collection)
	if err != nil {
		return nil, err
	}
	filterQuery, err := search.ParseQuery(matchOptions.GetFilterQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	query := search.ConjunctionQuery(collectionQuery, filterQuery)
	paginated.FillPagination(query, matchOptions.GetFilterQuery().GetPagination(), defaultPageSize)
	paginated.FillDefaultSortOption(query, defaultDeploymentSortOption)
	return s.deploymentDS.SearchListDeployments(ctx, query)
}
