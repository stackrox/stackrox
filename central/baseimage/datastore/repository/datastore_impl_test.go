//go:build sql_integration

package repository

import (
	"context"
	"testing"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	allAllowedCtx = sac.WithAllAccess(context.Background())
)

func TestBaseImageRepositoryDatastore(t *testing.T) {
	suite.Run(t, new(BaseImageRepositoryDatastoreTestSuite))
}

type BaseImageRepositoryDatastoreTestSuite struct {
	suite.Suite

	datastore DataStore
	storage   repoStore.Store
	pool      postgres.DB
}

var _ interface {
	suite.SetupAllSuite
	suite.TearDownAllSuite
} = (*BaseImageRepositoryDatastoreTestSuite)(nil)

func (s *BaseImageRepositoryDatastoreTestSuite) SetupSuite() {
	s.pool = pgtest.ForT(s.T())
	s.storage = repoStore.New(s.pool)
	s.datastore = New(s.storage)
}

func (s *BaseImageRepositoryDatastoreTestSuite) SetupTest() {
	ctx := allAllowedCtx
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

// TODO(ROX-32170): Add RBAC tests for BaseImageRepository datastore

func (s *BaseImageRepositoryDatastoreTestSuite) TestBaseImageRepositoryDatastore() {
	ctx := allAllowedCtx

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
	repo, found, err := s.datastore.GetRepository(allAllowedCtx, uuid.NewV4().String())
	s.NoError(err)
	s.False(found)
	s.Nil(repo)
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestDeleteRepositoryNotFound() {
	// No error when deleting non-existent repository
	s.NoError(s.datastore.DeleteRepository(allAllowedCtx, uuid.NewV4().String()))
}

func (s *BaseImageRepositoryDatastoreTestSuite) TestUniqueRepositoryPathConstraint() {
	ctx := allAllowedCtx

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
