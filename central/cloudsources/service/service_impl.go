package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cloudsources/datastore"
	"github.com/stackrox/rox/central/convert/storagetov1"
	"github.com/stackrox/rox/central/convert/v1tostorage"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/endpoints"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			"/v1.CloudSourcesService/CountCloudSources",
			"/v1.CloudSourcesService/GetCloudSource",
			"/v1.CloudSourcesService/ListCloudSources",
		},
		user.With(permissions.Modify(resources.Integration)): {
			"/v1.CloudSourcesService/CreateCloudSource",
			"/v1.CloudSourcesService/DeleteCloudSource",
			"/v1.CloudSourcesService/TestCloudSource",
			"/v1.CloudSourcesService/UpdateCloudSource",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedCloudSourcesServiceServer

	ds datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterCloudSourcesServiceServer(server, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCloudSourcesServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CountCloudSources returns the number of cloud sources matching the request query.
func (s *serviceImpl) CountCloudSources(ctx context.Context, request *v1.CountCloudSourcesRequest,
) (*v1.CountCloudSourcesResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	count, err := s.ds.CountCloudSources(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count cloud sources")
	}
	return &v1.CountCloudSourcesResponse{Count: int32(count)}, nil
}

// GetCloudSource returns a specific cloud source based on its ID.
func (s *serviceImpl) GetCloudSource(ctx context.Context, request *v1.GetCloudSourceRequest,
) (*v1.GetCloudSourceResponse, error) {
	resourceID := request.GetId()
	if resourceID == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be provided")
	}
	cloudSource, err := s.ds.GetCloudSource(ctx, resourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cloud source %q", resourceID)
	}
	return &v1.GetCloudSourceResponse{CloudSource: storagetov1.CloudSource(cloudSource)}, err
}

// ListCloudSources returns all cloud sources matching the request query.
func (s *serviceImpl) ListCloudSources(ctx context.Context, request *v1.ListCloudSourcesRequest,
) (*v1.ListCloudSourcesResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	paginated.FillPagination(query, request.GetPagination(), maxPaginationLimit)
	paginated.FillDefaultSortOption(
		query,
		&v1.QuerySortOption{
			Field: search.IntegrationName.String(),
		},
	)

	cloudSources, err := s.ds.ListCloudSources(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list cloud sources")
	}
	v1CloudSources := make([]*v1.CloudSource, 0, len(cloudSources))
	for _, cs := range cloudSources {
		v1CloudSources = append(v1CloudSources, storagetov1.CloudSource(cs))
	}
	return &v1.ListCloudSourcesResponse{CloudSources: v1CloudSources}, nil
}

// CreateCloudSource creates a new cloud source.
func (s *serviceImpl) CreateCloudSource(ctx context.Context, request *v1.CreateCloudSourceRequest,
) (*v1.CreateCloudSourceResponse, error) {
	v1CloudSource := request.GetCloudSource()
	if v1CloudSource.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id field must be empty when posting a new cloud source")
	}
	if err := s.validateCloudSource(ctx, v1CloudSource, true); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	v1CloudSource.Id = uuid.NewV4().String()
	storageCloudSource := v1tostorage.CloudSource(v1CloudSource)
	if err := s.ds.UpsertCloudSource(ctx, storageCloudSource); err != nil {
		_ = s.ds.DeleteCloudSource(ctx, storageCloudSource.GetId())
		return nil, errors.Wrapf(err, "failed to post cloud source %q", v1CloudSource.GetName())
	}
	return &v1.CreateCloudSourceResponse{CloudSource: storagetov1.CloudSource(storageCloudSource)}, nil
}

// UpdateCloudSource creates or updates a cloud source.
func (s *serviceImpl) UpdateCloudSource(ctx context.Context, request *v1.UpdateCloudSourceRequest,
) (*v1.Empty, error) {
	v1CloudSource := request.GetCloudSource()
	if err := s.validateCloudSource(ctx, v1CloudSource, request.GetUpdateCredentials()); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}
	updatedCloudSource := v1tostorage.CloudSource(v1CloudSource)

	if !request.GetUpdateCredentials() {
		if err := s.enrichWithStoredCredentials(ctx, updatedCloudSource); err != nil {
			if errors.Is(err, errox.NotFound) {
				return nil, errox.InvalidArgs.CausedByf(
					"cannot fetch existing credentials: %q does not exist", v1CloudSource.GetId(),
				)
			}
			return nil, errors.Wrapf(err, "cannot fetch existing credentials for %q", v1CloudSource.GetId())
		}
	}

	if err := s.ds.UpsertCloudSource(ctx, updatedCloudSource); err != nil {
		return nil, errors.Wrapf(err, "failed to put cloud source %q", v1CloudSource.GetId())
	}
	return &v1.Empty{}, nil
}

// DeleteCloudSource deletes a cloud source.
func (s *serviceImpl) DeleteCloudSource(ctx context.Context, request *v1.DeleteCloudSourceRequest,
) (*v1.Empty, error) {
	resourceID := request.GetId()
	if resourceID == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be provided")
	}
	if err := s.ds.DeleteCloudSource(ctx, resourceID); err != nil {
		return nil, errors.Wrapf(err, "failed to delete cloud source %q", resourceID)
	}
	return &v1.Empty{}, nil
}

// TestCloudSource tests a cloud source.
func (s *serviceImpl) TestCloudSource(_ context.Context, _ *v1.TestCloudSourceRequest,
) (*v1.Empty, error) {
	return nil, errox.NotImplemented.New("TestCloudSource is not implemented yet")
}

func (s *serviceImpl) validateCloudSource(ctx context.Context,
	cloudSource *v1.CloudSource, updateCredentials bool,
) error {
	if cloudSource == nil {
		return errors.New("empty cloud source")
	}

	errorList := errorhelpers.NewErrorList("Validation")
	if err := validateType(cloudSource); err != nil {
		errorList.AddError(err)
	}
	if updateCredentials && cloudSource.GetCredentials().GetSecret() == "" {
		errorList.AddString("cloud source credentials must be defined")
	}
	if err := endpoints.ValidateEndpoints(cloudSource.GetConfig()); err != nil {
		errorList.AddWrap(err, "invalid endpoint")
	}
	cloudSourceName := cloudSource.GetName()
	if cloudSourceName == "" {
		errorList.AddString("cloud source name must be defined")
		// Don't test for duplicated names if no name is set.
		return errorList.ToError()
	}
	if err := s.validateUniqueName(ctx, cloudSource); err != nil {
		errorList.AddError(err)
	}
	return errorList.ToError()
}

func (s *serviceImpl) enrichWithStoredCredentials(ctx context.Context,
	cloudSource *storage.CloudSource,
) error {
	id := cloudSource.GetId()
	storedCloudSource, err := s.ds.GetCloudSource(ctx, id)
	if err != nil {
		return err
	}
	return secrets.ReconcileScrubbedStructWithExisting(cloudSource, storedCloudSource)
}

func (s *serviceImpl) validateUniqueName(ctx context.Context, cloudSource *v1.CloudSource) error {
	query := getQueryBuilderFromFilter(&v1.CloudSourcesFilter{
		Names: []string{cloudSource.GetName()},
	}).ProtoQuery()
	integrations, err := s.ds.ListCloudSources(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to list cloud sources")
	}
	for _, cs := range integrations {
		if cs.GetId() != cloudSource.GetId() {
			return errors.Errorf("integration with name %q already exists", cloudSource.GetName())
		}
	}
	return nil
}

func validateType(cloudSource *v1.CloudSource) error {
	cloudSourceType := cloudSource.GetType()
	if cloudSourceType == v1.CloudSource_TYPE_UNSPECIFIED {
		return errors.New("cloud source type must be specified")
	}
	switch cloudSource.GetConfig().(type) {
	case *v1.CloudSource_PaladinCloud:
		if cloudSourceType != v1.CloudSource_TYPE_PALADIN_CLOUD {
			return errors.Errorf("invalid cloud source type %q", cloudSourceType.String())
		}
		return nil
	case *v1.CloudSource_Ocm:
		if cloudSourceType != v1.CloudSource_TYPE_OCM {
			return errors.Errorf("invalid cloud source type %q", cloudSourceType.String())
		}
		return nil
	}
	return errors.New("invalid cloud source config")
}

func getQueryBuilderFromFilter(filter *v1.CloudSourcesFilter) *search.QueryBuilder {
	queryBuilder := search.NewQueryBuilder()
	if names := filter.GetNames(); len(names) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.IntegrationName,
			sliceutils.Unique(names)...,
		)
	}
	if types := filter.GetTypes(); len(types) != 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.IntegrationType,
			sliceutils.Unique(sliceutils.StringSlice(types...))...,
		)
	}
	return queryBuilder
}
