package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cloudsources/datastore"
	"github.com/stackrox/rox/central/cloudsources/manager"
	"github.com/stackrox/rox/central/convert/storagetov1"
	"github.com/stackrox/rox/central/convert/v1tostorage"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/cloudsources"
	"github.com/stackrox/rox/pkg/cloudsources/opts"
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

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.Integration)): {
		v1.CloudSourcesService_CountCloudSources_FullMethodName,
		v1.CloudSourcesService_GetCloudSource_FullMethodName,
		v1.CloudSourcesService_ListCloudSources_FullMethodName,
	},
	user.With(permissions.Modify(resources.Integration)): {
		v1.CloudSourcesService_CreateCloudSource_FullMethodName,
		v1.CloudSourcesService_DeleteCloudSource_FullMethodName,
		v1.CloudSourcesService_TestCloudSource_FullMethodName,
		v1.CloudSourcesService_UpdateCloudSource_FullMethodName,
	},
})

type cloudSourceClientFactory = func(ctx context.Context,
	source *storage.CloudSource, opts ...opts.ClientOpts) (cloudsources.Client, error)

type serviceImpl struct {
	v1.UnimplementedCloudSourcesServiceServer

	ds            datastore.DataStore
	mgr           manager.Manager
	clientFactory cloudSourceClientFactory
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
	return &v1.GetCloudSourceResponse{CloudSource: storagetov1.CloudSource(cloudSource)}, nil
}

// ListCloudSources returns all cloud sources matching the request query.
func (s *serviceImpl) ListCloudSources(ctx context.Context, request *v1.ListCloudSourcesRequest,
) (*v1.ListCloudSourcesResponse, error) {
	query := getQueryBuilderFromFilter(request.GetFilter()).ProtoQuery()
	paginated.FillPagination(query, request.GetPagination(), maxPaginationLimit)
	query = paginated.FillDefaultSortOption(
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
	if v1CloudSource == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "empty cloud source")
	}
	if v1CloudSource.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id field must be empty when creating a new cloud source")
	}
	v1CloudSource.Id = uuid.NewV4().String()
	storageCloudSource := v1tostorage.CloudSource(v1CloudSource)

	if !v1CloudSource.GetSkipTestIntegration() {
		if err := s.testCloudSource(ctx, storageCloudSource); err != nil {
			return nil, errox.InvalidArgs.
				Newf("failed to test cloud source %q", v1CloudSource.GetName()).CausedBy(err)
		}
	}

	if err := s.ds.UpsertCloudSource(ctx, storageCloudSource); err != nil {
		return nil, errors.Wrapf(err, "failed to create cloud source %q", v1CloudSource.GetName())
	}
	// Short-circuit the cloud sources manager to ensure the latest changes are propagated.
	s.mgr.ShortCircuit()
	return &v1.CreateCloudSourceResponse{CloudSource: storagetov1.CloudSource(storageCloudSource)}, nil
}

// UpdateCloudSource creates or updates a cloud source.
func (s *serviceImpl) UpdateCloudSource(ctx context.Context, request *v1.UpdateCloudSourceRequest,
) (*v1.Empty, error) {
	v1CloudSource := request.GetCloudSource()
	if v1CloudSource == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "empty cloud source")
	}
	storageCloudSource := v1tostorage.CloudSource(v1CloudSource)

	if !request.GetUpdateCredentials() {
		if err := s.enrichWithStoredCredentials(ctx, storageCloudSource); err != nil {
			if errors.Is(err, errox.NotFound) {
				return nil, errox.InvalidArgs.CausedByf(
					"cannot fetch existing credentials: cloud source %q does not exist", v1CloudSource.GetId(),
				)
			}
			return nil, errors.Wrapf(err, "cannot fetch existing credentials for cloud source %q", v1CloudSource.GetId())
		}
	}

	if !v1CloudSource.GetSkipTestIntegration() {
		if err := s.testCloudSource(ctx, storageCloudSource); err != nil {
			return nil, errox.InvalidArgs.
				Newf("failed to test cloud source %q", v1CloudSource.GetName()).CausedBy(err)
		}
	}

	if err := s.ds.UpsertCloudSource(ctx, storageCloudSource); err != nil {
		return nil, errors.Wrapf(err, "failed to update cloud source %q", v1CloudSource.GetId())
	}
	// Short-circuit the cloud sources manager to ensure the latest changes are propagated.
	s.mgr.ShortCircuit()
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
func (s *serviceImpl) TestCloudSource(ctx context.Context, req *v1.TestCloudSourceRequest) (*v1.Empty, error) {
	v1CloudSource := req.GetCloudSource()
	if v1CloudSource == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "empty cloud source")
	}
	storageCloudSource := v1tostorage.CloudSource(v1CloudSource)

	if !req.GetUpdateCredentials() {
		if err := s.enrichWithStoredCredentials(ctx, storageCloudSource); err != nil {
			if errors.Is(err, errox.NotFound) {
				return nil, errox.InvalidArgs.CausedByf(
					"cannot fetch existing credentials: cloud source %q does not exist", v1CloudSource.GetId(),
				)
			}
			return nil, errors.Wrapf(err, "cannot fetch existing credentials for cloud source %q", v1CloudSource.GetId())
		}
	}
	if err := s.testCloudSource(ctx, storageCloudSource); err != nil {
		return nil, errox.InvalidArgs.
			Newf("test for cloud source %q failed", storageCloudSource.GetName()).CausedBy(err)
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) testCloudSource(ctx context.Context, storageCloudSource *storage.CloudSource) error {
	// Use a lower timeout as well as no retries for the test call. This is required to ensure that the UI request
	// does not time out, which has a default timeout of 10 seconds.
	client, err := s.clientFactory(ctx, storageCloudSource, opts.WithTimeout(8*time.Second), opts.WithRetries(0))
	if err != nil {
		return err
	}
	return client.Ping(ctx)
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
