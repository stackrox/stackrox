//go:build sql_integration

package watcher

import (
	"context"
	"testing"
	"time"

	baseImageDS "github.com/stackrox/rox/central/baseimage/datastore"
	repoDSMocks "github.com/stackrox/rox/central/baseimage/datastore/repository/mocks"
	tagDS "github.com/stackrox/rox/central/baseimage/datastore/tag"
	baseImageStore "github.com/stackrox/rox/central/baseimage/store/postgres"
	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	tagStore "github.com/stackrox/rox/central/baseimage/store/tag/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestUpgradeRefetchesLegacyTagCacheEntry simulates an upgrade from a row
// written before MetadataVersion existed and verifies that a scan rewrites it
// with history and promotes that history to base_images. A second scan must not
// rewrite the now-current row.
func TestUpgradeRefetchesLegacyTagCacheEntry(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := sac.WithAllAccess(context.Background())

	db := pgtest.ForT(t)
	defer db.Close()

	repos := repoStore.New(db.DB)
	tags := tagDS.New(tagStore.New(db.DB))
	baseImages := baseImageDS.New(baseImageStore.New(db.DB), db.DB)

	repo := &storage.BaseImageRepository{
		Id:             uuid.NewV4().String(),
		RepositoryPath: "registry.redhat.io/ubi9/ubi",
		TagPattern:     "9.*",
	}
	require.NoError(t, repos.Upsert(ctx, repo))

	const (
		tag    = "9.8"
		digest = "sha256:same"
		layer  = "sha256:layer1"
	)
	tagID, err := tagUUID(repo.GetId(), tag)
	require.NoError(t, err)

	// Seed the row exactly as a pre-versioning Central would have left it:
	// a digest and layers, no history, version 0.
	require.NoError(t, tags.UpsertMany(ctx, []*storage.BaseImageTag{{
		Id:                    tagID,
		BaseImageRepositoryId: repo.GetId(),
		Tag:                   tag,
		ManifestDigest:        digest,
		Created:               timestamppb.New(time.Now().Add(-time.Hour)),
		LayerDigests:          []string{layer},
	}}))

	seeded, err := tags.ListTagsByRepository(ctx, repo.GetId())
	require.NoError(t, err)
	require.Len(t, seeded, 1)
	require.Equal(t, uint32(0), seeded[0].GetMetadataVersion(), "legacy row must look unstamped")
	require.Empty(t, seeded[0].GetHistory(), "legacy row must have no history")

	history := []*storage.BaseImageHistoryEntry{
		{Instruction: "ADD", Value: "rootfs"},
		{Instruction: "USER", Value: "1001", Empty: true},
	}

	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)
	mockRegistry.EXPECT().Match(gomock.Any()).Return(true).Times(2)
	mockRegistry.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return([]string{tag}, nil).Times(2)
	mockRegistry.EXPECT().Source().Return(&storage.ImageIntegration{Id: "integration-1"}).Times(2)
	mockRegistry.EXPECT().Metadata(gomock.Any()).DoAndReturn(func(_ *storage.Image) (*storage.ImageMetadata, error) {
		return &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: timestamppb.Now(),
				Layers: []*storage.ImageLayer{
					{Instruction: "ADD", Value: "rootfs"},
					{Instruction: "USER", Value: "1001", Empty: true},
				},
			},
			V2:        &storage.V2Metadata{Digest: digest},
			LayerShas: []string{layer},
		}, nil
	}).Times(2)

	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{mockRegistry}).Times(2)

	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(2)

	counted := &countingTagDataStore{DataStore: tags}
	w := &watcherImpl{
		repoDS:       mockRepoDS,
		tagDS:        counted,
		baseImageDS:  baseImages,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		tagLimit:     10,
		batchSize:    10,
	}

	// First scan after upgrade: the legacy row is stale, so the fetched
	// metadata is written back with history and the current version.
	require.NoError(t, w.doScan(ctx, repo))
	require.Equal(t, 1, counted.upserts, "legacy row should be rewritten once")

	stored, err := tags.ListTagsByRepository(ctx, repo.GetId())
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.Equal(t, tagfetcher.CurrentMetadataVersion, stored[0].GetMetadataVersion())
	require.Equal(t, digest, stored[0].GetManifestDigest())
	protoassert.SlicesEqual(t, history, stored[0].GetHistory())

	promoted, err := baseImages.ListByRepository(ctx, repo.GetId())
	require.NoError(t, err)
	require.Len(t, promoted, 1)
	require.Equal(t, digest, promoted[0].GetManifestDigest())
	protoassert.SlicesEqual(t, history, promoted[0].GetHistory())

	// Second scan: the row is current, so it is not rewritten.
	require.NoError(t, w.doScan(ctx, repo))
	require.Equal(t, 1, counted.upserts, "current row should not be rewritten")

	stored, err = tags.ListTagsByRepository(ctx, repo.GetId())
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.Equal(t, tagfetcher.CurrentMetadataVersion, stored[0].GetMetadataVersion())
	protoassert.SlicesEqual(t, history, stored[0].GetHistory())
}

// countingTagDataStore counts writes while delegating to the real store.
type countingTagDataStore struct {
	tagDS.DataStore
	upserts int
}

func (c *countingTagDataStore) UpsertMany(ctx context.Context, tags []*storage.BaseImageTag) error {
	c.upserts++
	return c.DataStore.UpsertMany(ctx, tags)
}
