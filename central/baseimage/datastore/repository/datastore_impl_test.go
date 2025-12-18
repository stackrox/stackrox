//go:build sql_integration

package repository

import (
	"context"
	"testing"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestBaseImageRepositoryDatastore(t *testing.T) {
	suite.Run(t, new(BaseImageRepositoryDatastoreTestSuite))
}

type BaseImageRepositoryDatastoreTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	mockID   *mockIdentity.MockIdentity

	datastore     DataStore
	storage       repoStore.Store
	pool          postgres.DB
	imgAdminCtx   context.Context
	normalUserCtx context.Context
}

var _ interface {
	suite.SetupAllSuite
	suite.TearDownAllSuite
} = (*BaseImageRepositoryDatastoreTestSuite)(nil)

func (s *BaseImageRepositoryDatastoreTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.pool = pgtest.ForT(s.T())
	s.storage = repoStore.New(s.pool)
	s.datastore = New(s.storage)

	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return("uid").AnyTimes()
	mockID.EXPECT().FullName().Return("name").AnyTimes()
	mockID.EXPECT().FriendlyName().Return("name").AnyTimes()
	s.mockID = mockID

	ctx := sac.WithGlobalAccessScopeChecker(s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ImageAdministration),
		))
	s.imgAdminCtx = authn.ContextWithIdentity(ctx, mockID, s.T())

	ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.normalUserCtx = authn.ContextWithIdentity(ctx, mockID, s.T())
}

func (s *BaseImageRepositoryDatastoreTestSuite) SetupTest() {
	ctx := s.imgAdminCtx
	_, err := s.pool.Exec(ctx, "TRUNCATE base_image_repositories CASCADE")
	s.NoError(err)
}

func (s *BaseImageRepositoryDatastoreTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *BaseImageRepositoryDatastoreTestSuite) mustGetRepository(ctx context.Context, id string) (*storage.BaseImageRepository, bool) {
	repo, found, err := s.datastore.GetRepository(ctx, id)
	s.Require().NoError(err)
	return repo, found
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestBaseImageRepositoryDatastore() {
	ctx := s.imgAdminCtx

	// Test repository lifecycle: Create, Read, Update, Delete
	repo := &storage.BaseImageRepository{
		RepositoryPath: "registry.example.com/test-repo",
		TagPattern:     "v*",
	}

	// Test Create
	created, err := s.datastore.UpsertRepository(ctx, repo)
	s.NoError(err)
	s.NotEmpty(created.GetId())
	s.Equal(repo.GetRepositoryPath(), created.GetRepositoryPath())
	s.Equal(repo.GetTagPattern(), created.GetTagPattern())

	// Verify it exists
	retrieved, found := s.mustGetRepository(ctx, created.GetId())
	s.True(found)
	protoassert.Equal(s.T(), created, retrieved)

	// Get All - verify it's in the list
	allRepos, err := s.datastore.ListRepositories(ctx)
	s.NoError(err)
	protoassert.SliceContains(s.T(), allRepos, created)

	// Update - change tag pattern
	updated := &storage.BaseImageRepository{
		Id:             created.GetId(),
		RepositoryPath: created.GetRepositoryPath(),
		TagPattern:     "latest",
	}
	updatedResult, err := s.datastore.UpsertRepository(ctx, updated)
	s.NoError(err)
	s.Equal(created.GetId(), updatedResult.GetId())
	s.Equal("latest", updatedResult.GetTagPattern())

	// Delete - remove the repository
	s.NoError(s.datastore.DeleteRepository(ctx, created.GetId()))
	_, found = s.mustGetRepository(ctx, created.GetId())
	s.False(found)
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestGetRepositoryNotFound() {
	repo, found, err := s.datastore.GetRepository(s.imgAdminCtx, uuid.NewV4().String())
	s.NoError(err)
	s.False(found)
	s.Nil(repo)
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestDeleteRepositoryNotFound() {
	// No error when deleting non-existent repository
	s.NoError(s.datastore.DeleteRepository(s.imgAdminCtx, uuid.NewV4().String()))
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUniqueRepositoryPathConstraint() {
	ctx := s.imgAdminCtx

	// Create first repository
	repo1 := &storage.BaseImageRepository{
		RepositoryPath: "registry.example.com/unique-repo",
		TagPattern:     "v1.*",
	}

	createdRepo1, err := s.datastore.UpsertRepository(ctx, repo1)
	s.NoError(err)
	s.NotEmpty(createdRepo1.GetId())

	// Try to create a second repository with the same repository_path but different tag_pattern
	repo2 := &storage.BaseImageRepository{
		RepositoryPath: "registry.example.com/unique-repo", // Same path
		TagPattern:     "v2.*",                             // Different pattern
	}

	// This should fail due to unique constraint on repository_path
	_, err = s.datastore.UpsertRepository(ctx, repo2)
	s.Error(err, "Creating second repository with same path should fail due to unique constraint")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestListRepositoriesAccessDenied() {
	// Create mock auth provider
	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)

	// Create context with a normal user that has no access
	noAccessCtx := sac.WithNoAccess(basic.ContextWithNoAccessIdentity(s.T(), authProvider))

	// Attempt to list repositories with no access user should be denied
	repos, err := s.datastore.ListRepositories(noAccessCtx)
	s.Error(err, "ListRepositories should fail for user with no access")
	s.Nil(repos, "No repositories should be returned when access is denied")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUserCtxOperationsDenied() {
	// Test all datastore operations with userCtx (has GlobalAccessScopeChecker but mock identity has no permissions)
	// Operations should fail due to lack of permissions, not missing scope checker

	repo := &storage.BaseImageRepository{
		RepositoryPath: "registry.example.com/test-repo",
		TagPattern:     "v*",
	}

	// Test ListRepositories
	repos, err := s.datastore.ListRepositories(s.normalUserCtx)
	s.Error(err, "ListRepositories should fail for user context without permissions")
	s.Nil(repos, "No repositories should be returned when access is denied due to permissions")

	// Test UpsertRepository
	_, err = s.datastore.UpsertRepository(s.normalUserCtx, repo)
	s.Error(err, "UpsertRepository should fail for user context without permissions")

	// Test GetRepository
	_, _, err = s.datastore.GetRepository(s.normalUserCtx, "non-existent-id")
	s.Error(err, "GetRepository should fail for user context without permissions")

	// Test DeleteRepository
	err = s.datastore.DeleteRepository(s.normalUserCtx, "non-existent-id")
	s.Error(err, "DeleteRepository should fail for user context without permissions")
}
