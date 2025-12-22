//go:build sql_integration

package tag

import (
	"context"
	"testing"
	"time"

	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	tagStore "github.com/stackrox/rox/central/baseimage/store/tag/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TagDataStoreIntegrationSuite struct {
	suite.Suite
	dataStore DataStore
	repoDS    repoDS.DataStore
	testDB    *pgtest.TestPostgres
	ctx       context.Context
}

func TestTagDataStoreIntegration(t *testing.T) {
	suite.Run(t, new(TagDataStoreIntegrationSuite))
}

func (s *TagDataStoreIntegrationSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.dataStore = New(tagStore.New(s.testDB.DB))
	s.repoDS = repoDS.New(repoStore.New(s.testDB.DB))
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *TagDataStoreIntegrationSuite) SetupTest() {
	_, err := s.testDB.Exec(s.ctx, "TRUNCATE base_image_repositories CASCADE")
	s.NoError(err)
}

// TestListTagsByRepository_Sorting verifies that tags are returned sorted by
// created timestamp descending (newest first).
func (s *TagDataStoreIntegrationSuite) TestListTagsByRepository_Sorting() {
	// Create repository
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo)
	s.NoError(err)

	// Create tags with different timestamps
	now := time.Now()
	tags := []*storage.BaseImageTag{
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   "8.10-1234",
			ManifestDigest:        "sha256:abc123",
			Created:               timestamppb.New(now.Add(-3 * time.Hour)), // Oldest
		},
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   "8.10-1235",
			ManifestDigest:        "sha256:def456",
			Created:               timestamppb.New(now.Add(-1 * time.Hour)), // Newest
		},
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   "8.10-1236",
			ManifestDigest:        "sha256:ghi789",
			Created:               timestamppb.New(now.Add(-2 * time.Hour)), // Middle
		},
	}

	err = s.dataStore.UpsertMany(s.ctx, tags)
	s.NoError(err)

	// List tags
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo.GetId())
	s.NoError(err)
	s.Require().Len(result, 3)

	// Verify sorting: newest first
	s.Equal("8.10-1235", result[0].GetTag(), "Newest tag should be first")
	s.Equal("8.10-1236", result[1].GetTag(), "Middle tag should be second")
	s.Equal("8.10-1234", result[2].GetTag(), "Oldest tag should be last")
}

// TestListTagsByRepository_FiltersByRepository verifies database-level filtering
// by repository ID.
func (s *TagDataStoreIntegrationSuite) TestListTagsByRepository_FiltersByRepository() {
	// Create two repositories
	repo1 := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo1)
	s.NoError(err)

	repo2 := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi9/ubi",
		TagPattern:     "9.*",
	}
	_, err = s.repoDS.UpsertRepository(s.ctx, repo2)
	s.NoError(err)

	now := time.Now()

	// Create tags for repo1
	repo1Tags := []*storage.BaseImageTag{
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo1.GetId(),
			Tag:                   "8.10-1234",
			Created:               timestamppb.New(now),
		},
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo1.GetId(),
			Tag:                   "8.10-1235",
			Created:               timestamppb.New(now.Add(-1 * time.Hour)),
		},
	}
	err = s.dataStore.UpsertMany(s.ctx, repo1Tags)
	s.NoError(err)

	// Create tags for repo2
	repo2Tags := []*storage.BaseImageTag{
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo2.GetId(),
			Tag:                   "9.1-5678",
			Created:               timestamppb.New(now),
		},
	}
	err = s.dataStore.UpsertMany(s.ctx, repo2Tags)
	s.NoError(err)

	// List tags for repo1 - should only return repo1 tags
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo1.GetId())
	s.NoError(err)
	s.Require().Len(result, 2)
	s.Equal("8.10-1234", result[0].GetTag())
	s.Equal("8.10-1235", result[1].GetTag())

	// List tags for repo2 - should only return repo2 tags
	result, err = s.dataStore.ListTagsByRepository(s.ctx, repo2.GetId())
	s.NoError(err)
	s.Require().Len(result, 1)
	s.Equal("9.1-5678", result[0].GetTag())
}

// TestUpsertMany_BatchOperation verifies batch upsert functionality.
func (s *TagDataStoreIntegrationSuite) TestUpsertMany_BatchOperation() {
	// Create repository
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo)
	s.NoError(err)

	now := time.Now()

	// Create 100 tags
	tags := make([]*storage.BaseImageTag, 100)
	for i := 0; i < 100; i++ {
		tags[i] = &storage.BaseImageTag{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   uuid.NewV4().String(), // Unique tag
			ManifestDigest:        "sha256:digest" + uuid.NewV4().String(),
			Created:               timestamppb.New(now.Add(-time.Duration(i) * time.Minute)),
		}
	}

	// Batch insert
	err = s.dataStore.UpsertMany(s.ctx, tags)
	s.NoError(err)

	// Verify all tags were inserted
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo.GetId())
	s.NoError(err)
	s.Len(result, 100)
}

// TestUpsertMany_UpdatesExistingTags verifies that UpsertMany updates existing tags.
func (s *TagDataStoreIntegrationSuite) TestUpsertMany_UpdatesExistingTags() {
	// Create repository
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo)
	s.NoError(err)

	now := time.Now()
	tagID := uuid.NewV4().String()

	// Insert initial tag
	initialTag := &storage.BaseImageTag{
		Id:                    tagID,
		BaseImageRepositoryId: repo.GetId(),
		Tag:                   "8.10-1234",
		ManifestDigest:        "sha256:old-digest",
		Created:               timestamppb.New(now),
	}
	err = s.dataStore.UpsertMany(s.ctx, []*storage.BaseImageTag{initialTag})
	s.NoError(err)

	// Update tag with new digest
	updatedTag := &storage.BaseImageTag{
		Id:                    tagID, // Same ID
		BaseImageRepositoryId: repo.GetId(),
		Tag:                   "8.10-1234",
		ManifestDigest:        "sha256:new-digest", // Updated
		Created:               timestamppb.New(now.Add(1 * time.Hour)),
	}
	err = s.dataStore.UpsertMany(s.ctx, []*storage.BaseImageTag{updatedTag})
	s.NoError(err)

	// Verify update
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo.GetId())
	s.NoError(err)
	s.Require().Len(result, 1)
	s.Equal("sha256:new-digest", result[0].GetManifestDigest())
}

// TestDeleteMany_BatchOperation verifies batch delete functionality.
func (s *TagDataStoreIntegrationSuite) TestDeleteMany_BatchOperation() {
	// Create repository
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo)
	s.NoError(err)

	now := time.Now()

	// Create 10 tags
	tags := make([]*storage.BaseImageTag, 10)
	tagIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		tagID := uuid.NewV4().String()
		tags[i] = &storage.BaseImageTag{
			Id:                    tagID,
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   uuid.NewV4().String(),
			Created:               timestamppb.New(now),
		}
		tagIDs[i] = tagID
	}
	err = s.dataStore.UpsertMany(s.ctx, tags)
	s.NoError(err)

	// Delete first 5 tags
	err = s.dataStore.DeleteMany(s.ctx, tagIDs[:5])
	s.NoError(err)

	// Verify 5 tags remain
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo.GetId())
	s.NoError(err)
	s.Len(result, 5)
}

// TestListTagsByRepository_EmptyResult verifies behavior with no tags.
func (s *TagDataStoreIntegrationSuite) TestListTagsByRepository_EmptyResult() {
	// Create repository with no tags
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo)
	s.NoError(err)

	// List tags for empty repository
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo.GetId())
	s.NoError(err)
	s.Empty(result)
}

// TestListTagsByRepository_NonexistentRepository verifies behavior with invalid repo ID.
func (s *TagDataStoreIntegrationSuite) TestListTagsByRepository_NonexistentRepository() {
	// Query with non-existent repository ID
	result, err := s.dataStore.ListTagsByRepository(s.ctx, uuid.NewV4().String())
	s.NoError(err)
	s.Empty(result)
}

// TestSorting_NilTimestamps verifies handling of nil created timestamps.
// Tags with nil timestamps should sort as oldest.
func (s *TagDataStoreIntegrationSuite) TestSorting_NilTimestamps() {
	// Create repository
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.*",
	}
	_, err := s.repoDS.UpsertRepository(s.ctx, repo)
	s.NoError(err)

	now := time.Now()

	// Create tags with mix of nil and non-nil timestamps
	tags := []*storage.BaseImageTag{
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   "with-timestamp",
			Created:               timestamppb.New(now),
		},
		{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   "nil-timestamp",
			Created:               nil, // Nil timestamp
		},
	}
	err = s.dataStore.UpsertMany(s.ctx, tags)
	s.NoError(err)

	// List tags
	result, err := s.dataStore.ListTagsByRepository(s.ctx, repo.GetId())
	s.NoError(err)
	s.Require().Len(result, 2)

	// Tag with timestamp should be first (newest)
	s.Equal("with-timestamp", result[0].GetTag())
	// Tag with nil timestamp should be last (oldest/epoch)
	s.Equal("nil-timestamp", result[1].GetTag())
}
