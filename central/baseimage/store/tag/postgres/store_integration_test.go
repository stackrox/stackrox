//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type BaseImageTagsIntegrationSuite struct {
	suite.Suite
	tagStore  Store
	repoStore postgres.Store
	testDB    *pgtest.TestPostgres
}

func TestBaseImageTagsIntegration(t *testing.T) {
	suite.Run(t, new(BaseImageTagsIntegrationSuite))
}

func (s *BaseImageTagsIntegrationSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.tagStore = New(s.testDB.DB)
	s.repoStore = postgres.New(s.testDB.DB)
}

func (s *BaseImageTagsIntegrationSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	_, err := s.testDB.Exec(ctx, "TRUNCATE base_image_repositories CASCADE")
	s.NoError(err)
}

func (s *BaseImageTagsIntegrationSuite) TestForeignKeyConstraint() {
	ctx := sac.WithAllAccess(context.Background())

	// Create tag with non-existent repository - should fail
	tag := &storage.BaseImageTag{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: uuid.NewV4().String(), // doesn't exist
		Tag:                   "8.10-1234",
	}
	err := s.tagStore.Upsert(ctx, tag)
	s.Error(err) // FK violation expected
}

func (s *BaseImageTagsIntegrationSuite) TestCompositeUniqueConstraint() {
	ctx := sac.WithAllAccess(context.Background())

	// Create repository first
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.10-*",
	}
	s.NoError(s.repoStore.Upsert(ctx, repo))

	// Create first tag
	tag1 := &storage.BaseImageTag{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repo.GetId(),
		Tag:                   "8.10-1234",
	}
	s.NoError(s.tagStore.Upsert(ctx, tag1))

	// Create second tag with same repo+tag but different ID - should fail
	tag2 := &storage.BaseImageTag{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repo.GetId(),
		Tag:                   "8.10-1234", // duplicate
	}
	err := s.tagStore.Upsert(ctx, tag2)
	s.Error(err) // Unique violation expected
}

func (s *BaseImageTagsIntegrationSuite) TestPlatformsRoundTrip() {
	ctx := sac.WithAllAccess(context.Background())

	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
		TagPattern:     "8.10-*",
	}
	s.NoError(s.repoStore.Upsert(ctx, repo))

	tag := &storage.BaseImageTag{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repo.GetId(),
		Tag:                   "8.10-1234",
		ManifestDigest:        "sha256:list",
		IsManifestList:        true,
		ListDigests: map[string]string{
			"linux/amd64":  "sha256:amd",
			"linux/arm/v7": "sha256:armv7",
		},
		Platforms: []*storage.BaseImagePlatform{
			{
				Os:             "linux",
				Architecture:   "amd64",
				ManifestDigest: "sha256:amd",
				LayerDigests:   []string{"sha256:amd-l1", "sha256:amd-l2"},
				History: []*storage.BaseImageHistoryEntry{
					{Instruction: "ADD", Value: "file"},
					{Instruction: "USER", Value: "1001", Empty: true},
				},
			},
			{
				Os:             "linux",
				Architecture:   "arm",
				Variant:        "v7",
				ManifestDigest: "sha256:armv7",
				LayerDigests:   []string{"sha256:armv7-l1"},
			},
		},
	}
	s.NoError(s.tagStore.Upsert(ctx, tag))

	got, found, err := s.tagStore.Get(ctx, tag.GetId())
	s.NoError(err)
	s.True(found)
	protoassert.Equal(s.T(), tag, got)
}

func (s *BaseImageTagsIntegrationSuite) TestCascadeDelete() {
	ctx := sac.WithAllAccess(context.Background())

	// Create repository
	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi8/ubi",
	}
	s.NoError(s.repoStore.Upsert(ctx, repo))

	// Create multiple tags
	for i := range 3 {
		tag := &storage.BaseImageTag{
			Id:                    uuid.NewV4().String(),
			BaseImageRepositoryId: repo.GetId(),
			Tag:                   fmt.Sprintf("8.10-%d", i),
		}
		s.NoError(s.tagStore.Upsert(ctx, tag))
	}

	// Verify tags exist
	count, err := s.tagStore.Count(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(3, count)

	// Delete repository
	s.NoError(s.repoStore.Delete(ctx, repo.GetId()))

	// Verify tags were cascade deleted
	count, err = s.tagStore.Count(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)
}
