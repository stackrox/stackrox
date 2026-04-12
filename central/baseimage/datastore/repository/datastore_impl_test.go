//go:build sql_integration

package repository

import (
	"context"
	"testing"
	"time"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestBaseImageRepositoryDatastore(t *testing.T) {
	suite.Run(t, new(BaseImageRepositoryDatastoreTestSuite))
}

type BaseImageRepositoryDatastoreTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	datastore     DataStore
	storage       repoStore.Store
	pool          postgres.DB
	imgAdminCtx   context.Context
	normalUserCtx context.Context
}

var _ repoStore.Store = (*blockingRepositoryStore)(nil)

var _ interface {
	suite.SetupAllSuite
	suite.TearDownAllSuite
} = (*BaseImageRepositoryDatastoreTestSuite)(nil)

func (s *BaseImageRepositoryDatastoreTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.pool = pgtest.ForT(s.T())
	s.storage = repoStore.New(s.pool)
	s.datastore = New(s.storage, concurrency.NewKeyFence())

	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return("uid").AnyTimes()
	mockID.EXPECT().FullName().Return("name").AnyTimes()
	mockID.EXPECT().FriendlyName().Return("name").AnyTimes()

	ctx := sac.WithGlobalAccessScopeChecker(s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ImageAdministration),
		))
	s.imgAdminCtx = authn.ContextWithIdentity(ctx, mockID)

	ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.normalUserCtx = authn.ContextWithIdentity(ctx, mockID)
}

func (s *BaseImageRepositoryDatastoreTestSuite) SetupTest() {
	_, err := s.pool.Exec(s.imgAdminCtx, "TRUNCATE base_image_repositories CASCADE")
	s.NoError(err)
}

func (s *BaseImageRepositoryDatastoreTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
	s.pool.Close()
}

func (s *BaseImageRepositoryDatastoreTestSuite) mustGetRepository(ctx context.Context, id string) (*storage.BaseImageRepository, bool) {
	repo, found, err := s.datastore.GetRepository(ctx, id)
	s.Require().NoError(err)
	return repo, found
}

func newRepository(path, tagPattern string) *storage.BaseImageRepository {
	return &storage.BaseImageRepository{
		RepositoryPath: path,
		TagPattern:     tagPattern,
	}
}

func ptr(s string) *string {
	return &s
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestBaseImageRepositoryDatastore() {
	ctx := s.imgAdminCtx

	// Test repository lifecycle: Create, Read, Update, Delete.
	repo := newRepository("registry.example.com/test-repo", "v*")

	// Test Create.
	created, err := s.datastore.UpsertRepository(ctx, repo)
	s.NoError(err)
	s.NotEmpty(created.GetId())
	s.Equal(repo.GetRepositoryPath(), created.GetRepositoryPath())
	s.Equal(repo.GetTagPattern(), created.GetTagPattern())
	s.Equal(storage.BaseImageRepository_CREATED, created.GetStatus())
	s.Zero(created.GetFailureCount())
	s.Empty(created.GetLastFailureMessage())

	// Verify it exists.
	retrieved, found := s.mustGetRepository(ctx, created.GetId())
	s.True(found)
	protoassert.Equal(s.T(), created, retrieved)

	// Get all and verify the repository is returned.
	allRepos, err := s.datastore.ListRepositories(ctx)
	s.NoError(err)
	protoassert.SliceContains(s.T(), allRepos, created)

	// Update the tag pattern.
	updated := &storage.BaseImageRepository{
		Id:             created.GetId(),
		RepositoryPath: created.GetRepositoryPath(),
		TagPattern:     "latest",
	}
	updatedResult, err := s.datastore.UpsertRepository(ctx, updated)
	s.NoError(err)
	s.Equal(created.GetId(), updatedResult.GetId())
	s.Equal("latest", updatedResult.GetTagPattern())
	s.Equal(storage.BaseImageRepository_CREATED, updatedResult.GetStatus())
	s.Zero(updatedResult.GetFailureCount())
	s.Empty(updatedResult.GetLastFailureMessage())

	// Delete the repository.
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
	s.NoError(s.datastore.DeleteRepository(s.imgAdminCtx, uuid.NewV4().String()))
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUniqueRepositoryPathConstraint() {
	ctx := s.imgAdminCtx

	// Create the first repository.
	repo1 := newRepository("registry.example.com/unique-repo", "v1.*")

	createdRepo1, err := s.datastore.UpsertRepository(ctx, repo1)
	s.NoError(err)
	s.NotEmpty(createdRepo1.GetId())

	// Creating a second repository with the same path should fail.
	repo2 := newRepository("registry.example.com/unique-repo", "v2.*")

	_, err = s.datastore.UpsertRepository(ctx, repo2)
	s.Error(err, "Creating second repository with same path should fail due to unique constraint")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestListRepositoriesAccessDenied() {
	// Create a context with a user that has no image administration access.
	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)

	noAccessCtx := sac.WithNoAccess(basic.ContextWithNoAccessIdentity(authProvider))

	repos, err := s.datastore.ListRepositories(noAccessCtx)
	s.Error(err, "ListRepositories should fail for user with no access")
	s.Nil(repos, "No repositories should be returned when access is denied")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUserCtxOperationsDenied() {
	// Exercise all datastore methods with a user context that lacks permissions.
	repo := newRepository("registry.example.com/test-repo", "v*")

	repos, err := s.datastore.ListRepositories(s.normalUserCtx)
	s.Error(err, "ListRepositories should fail for user context without permissions")
	s.Nil(repos, "No repositories should be returned when access is denied due to permissions")

	_, err = s.datastore.UpsertRepository(s.normalUserCtx, repo)
	s.Error(err, "UpsertRepository should fail for user context without permissions")

	_, _, err = s.datastore.GetRepository(s.normalUserCtx, "non-existent-id")
	s.Error(err, "GetRepository should fail for user context without permissions")

	err = s.datastore.DeleteRepository(s.normalUserCtx, "non-existent-id")
	s.Error(err, "DeleteRepository should fail for user context without permissions")

	_, err = s.datastore.UpdateStatus(s.normalUserCtx, "non-existent-id", StatusUpdate{Status: storage.BaseImageRepository_READY})
	s.Error(err, "UpdateStatus should fail for user context without permissions")

	_, err = s.datastore.UpdateConfiguration(s.normalUserCtx, "non-existent-id", ConfigUpdate{TagPattern: ptr("v*")})
	s.Error(err, "UpdateConfiguration should fail for user context without permissions")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpdateStatus_UpdatesLastPolledAt() {
	ctx := s.imgAdminCtx

	repo := newRepository("registry.example.com/test-repo", "v*")
	created, err := s.datastore.UpsertRepository(ctx, repo)
	s.Require().NoError(err)
	s.Nil(created.GetLastPolledAt(), "LastPolledAt should be nil initially")

	pollTime := time.Now().Truncate(time.Microsecond)
	_, err = s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:       storage.BaseImageRepository_IN_PROGRESS,
		LastPolledAt: &pollTime,
	})
	s.NoError(err)

	retrieved, found := s.mustGetRepository(ctx, created.GetId())
	s.True(found)
	s.NotNil(retrieved.GetLastPolledAt())
	s.Equal(pollTime.UTC(), retrieved.GetLastPolledAt().AsTime().UTC())
	s.Equal(storage.BaseImageRepository_IN_PROGRESS, retrieved.GetStatus())
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpdateStatus_DoesNotChangeUpdatedAt() {
	ctx := s.imgAdminCtx

	repo := newRepository("registry.example.com/test-repo", "v*")
	created, err := s.datastore.UpsertRepository(ctx, repo)
	s.Require().NoError(err)
	originalUpdatedAt := created.GetUpdatedAt()
	s.NotNil(originalUpdatedAt)

	time.Sleep(10 * time.Millisecond)

	pollTime := time.Now()
	_, err = s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:       storage.BaseImageRepository_READY,
		LastPolledAt: &pollTime,
	})
	s.NoError(err)

	retrieved, found := s.mustGetRepository(ctx, created.GetId())
	s.True(found)
	s.Equal(originalUpdatedAt.AsTime(), retrieved.GetUpdatedAt().AsTime(),
		"UpdatedAt should not change when updating repository status")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpdateStatus_FailureFields() {
	ctx := s.imgAdminCtx

	repo := newRepository("registry.example.com/test-repo", "v1.*")
	created, err := s.datastore.UpsertRepository(ctx, repo)
	s.Require().NoError(err)

	failureMessage := "polling failure"
	pollTime := time.Now().Truncate(time.Microsecond)
	_, err = s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:             storage.BaseImageRepository_FAILED,
		LastPolledAt:       &pollTime,
		LastFailureMessage: &failureMessage,
		FailureCountOp:     FailureCountIncrement,
	})
	s.Require().NoError(err)

	retrieved, found := s.mustGetRepository(ctx, created.GetId())
	s.True(found)
	s.Equal(storage.BaseImageRepository_FAILED, retrieved.GetStatus())
	s.Equal(int32(1), retrieved.GetFailureCount())
	s.Equal(failureMessage, retrieved.GetLastFailureMessage())
	s.Equal(pollTime.UTC(), retrieved.GetLastPolledAt().AsTime().UTC())

	clearedFailureMessage := ""
	_, err = s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:             storage.BaseImageRepository_READY,
		LastFailureMessage: &clearedFailureMessage,
		FailureCountOp:     FailureCountReset,
	})
	s.Require().NoError(err)

	retrieved, found = s.mustGetRepository(ctx, created.GetId())
	s.True(found)
	s.Equal(storage.BaseImageRepository_READY, retrieved.GetStatus())
	s.Zero(retrieved.GetFailureCount())
	s.Empty(retrieved.GetLastFailureMessage())
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpdateStatus_NonExistentRepo() {
	ctx := s.imgAdminCtx

	nonExistentID := uuid.NewV4().String()
	updated, err := s.datastore.UpdateStatus(ctx, nonExistentID, StatusUpdate{Status: storage.BaseImageRepository_READY})

	s.Nil(updated)
	s.NoError(err, "UpdateStatus on non-existent repo should return nil")
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpsertRepository_InsertsProvidedIDWhenRepositoryDoesNotExist() {
	repo := newRepository("registry.example.com/test-repo", "v*")
	repo.Id = uuid.NewV4().String()

	updated, err := s.datastore.UpsertRepository(s.imgAdminCtx, repo)

	s.NoError(err)
	s.NotNil(updated)
	s.Equal(repo.GetId(), updated.GetId())
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpdateConfiguration_NotFound() {
	updated, err := s.datastore.UpdateConfiguration(s.imgAdminCtx, uuid.NewV4().String(), ConfigUpdate{TagPattern: ptr("v*")})

	s.Nil(updated)
	s.ErrorIs(err, errox.NotFound)
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUpdateStatus_FailureCountOperations() {
	ctx := s.imgAdminCtx

	repo := newRepository("registry.example.com/test-repo", "v1.*")
	created, err := s.datastore.UpsertRepository(ctx, repo)
	s.Require().NoError(err)

	// Increment failure count.
	failureMessage := "polling failure"
	updated, err := s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:             storage.BaseImageRepository_FAILED,
		LastFailureMessage: &failureMessage,
		FailureCountOp:     FailureCountIncrement,
	})
	s.Require().NoError(err)
	s.Equal(int32(1), updated.GetFailureCount())

	// Increment again.
	updated, err = s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:         storage.BaseImageRepository_FAILED,
		FailureCountOp: FailureCountIncrement,
	})
	s.Require().NoError(err)
	s.Equal(int32(2), updated.GetFailureCount())

	// Reset failure count.
	updated, err = s.datastore.UpdateStatus(ctx, created.GetId(), StatusUpdate{
		Status:         storage.BaseImageRepository_READY,
		FailureCountOp: FailureCountReset,
	})
	s.Require().NoError(err)
	s.Zero(updated.GetFailureCount())
	s.Equal(storage.BaseImageRepository_READY, updated.GetStatus())
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestConcurrency_OperationsSerialize() {
	ctx := s.imgAdminCtx
	repo := &storage.BaseImageRepository{Id: "repo-id", Status: storage.BaseImageRepository_CREATED}
	store := &blockingRepositoryStore{
		repo:          repo,
		getEntered:    make(chan struct{}, 1),
		deleteEntered: make(chan struct{}, 1),
		unblockGet:    make(chan struct{}),
	}
	ds := New(store, concurrency.NewKeyFence())

	statusUpdateDone := make(chan struct{})
	go func() {
		_, _ = ds.UpdateStatus(ctx, repo.GetId(), StatusUpdate{Status: storage.BaseImageRepository_QUEUED})
		close(statusUpdateDone)
	}()

	<-store.getEntered

	deleteDone := make(chan struct{})
	go func() {
		_ = ds.DeleteRepository(ctx, repo.GetId())
		close(deleteDone)
	}()

	select {
	case <-store.deleteEntered:
		s.Fail("DeleteRepository reached store.Delete while UpdateRepositoryStatus held the fence")
	default:
	}

	close(store.unblockGet)
	<-statusUpdateDone
	<-store.deleteEntered
	<-deleteDone
}

type blockingRepositoryStore struct {
	repo *storage.BaseImageRepository

	getEntered    chan struct{}
	deleteEntered chan struct{}
	unblockGet    chan struct{}
}

func (s *blockingRepositoryStore) Upsert(ctx context.Context, obj *storage.BaseImageRepository) error {
	s.repo = proto.Clone(obj).(*storage.BaseImageRepository)
	return nil
}

func (s *blockingRepositoryStore) UpsertMany(context.Context, []*storage.BaseImageRepository) error {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) Delete(context.Context, string) error {
	select {
	case s.deleteEntered <- struct{}{}:
	default:
	}
	s.repo = nil
	return nil
}

func (s *blockingRepositoryStore) DeleteByQuery(context.Context, *v1.Query) error {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) DeleteByQueryWithIDs(context.Context, *v1.Query) ([]string, error) {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) DeleteMany(context.Context, []string) error {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) PruneMany(context.Context, []string) error {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) Count(context.Context, *v1.Query) (int, error) {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) Exists(context.Context, string) (bool, error) {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) Search(context.Context, *v1.Query) ([]search.Result, error) {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) Get(context.Context, string) (*storage.BaseImageRepository, bool, error) {
	select {
	case s.getEntered <- struct{}{}:
	default:
	}
	<-s.unblockGet
	if s.repo == nil {
		return nil, false, nil
	}
	return proto.Clone(s.repo).(*storage.BaseImageRepository), true, nil
}

func (s *blockingRepositoryStore) GetMany(context.Context, []string) ([]*storage.BaseImageRepository, []int, error) {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) GetIDs(context.Context) ([]string, error) {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) Walk(context.Context, func(*storage.BaseImageRepository) error) error {
	panic("unexpected call")
}

func (s *blockingRepositoryStore) WalkByQuery(context.Context, *v1.Query, func(*storage.BaseImageRepository) error) error {
	panic("unexpected call")
}
