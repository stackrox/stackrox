//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/baseimage/store/postgres"
	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestBaseImageDataStore(t *testing.T) {
	suite.Run(t, new(BaseImageDataStoreTestSuite))
}

type BaseImageDataStoreTestSuite struct {
	suite.Suite

	pool      *pgtest.TestPostgres
	datastore DataStore
	repoStore repoStore.Store
	ctx       context.Context
}

func (s *BaseImageDataStoreTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.pool = pgtest.ForT(s.T())

	// Initialize the generated stores.
	store := postgres.New(s.pool)
	s.repoStore = repoStore.New(s.pool)

	s.datastore = New(store, s.pool)
}

// createRepository is a helper to create a repository for FK constraints.
func (s *BaseImageDataStoreTestSuite) createRepository(id, path, pattern string) *storage.BaseImageRepository {
	repo := &storage.BaseImageRepository{
		Id:             id,
		RepositoryPath: path,
		TagPattern:     pattern,
	}
	err := s.repoStore.Upsert(s.ctx, repo)
	s.Require().NoError(err)
	return repo
}

func (s *BaseImageDataStoreTestSuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
}
func (s *BaseImageDataStoreTestSuite) TestUpsertImage() {
	imageID := uuid.NewV4().String()
	baseImage := &storage.BaseImage{
		Id:               imageID,
		ManifestDigest:   "sha256:manifest123",
		FirstLayerDigest: "sha256:layer1",
	}

	digests := []string{
		"sha256:layer1", // Index 0
		"sha256:layer2", // Index 1
	}

	err := s.datastore.UpsertImage(s.ctx, baseImage, digests)
	s.Require().NoError(err)

	// Verify Retrieval
	img, found, err := s.datastore.GetBaseImage(s.ctx, "sha256:manifest123")
	s.Require().NoError(err)
	s.True(found)
	s.Equal(imageID, img.GetId())
	s.Len(img.GetLayers(), 2)

	// Verify generated fields
	s.NotEmpty(img.GetLayers()[0].GetId())
	s.Equal(int32(0), img.GetLayers()[0].GetIndex())
	s.Equal(imageID, img.GetLayers()[0].GetBaseImageId())
	s.Equal("sha256:layer1", img.GetLayers()[0].GetLayerDigest())

	s.NotEmpty(img.GetLayers()[1].GetId())
	s.Equal(int32(1), img.GetLayers()[1].GetIndex())
	s.Equal(imageID, img.GetLayers()[1].GetBaseImageId())
	s.Equal("sha256:layer2", img.GetLayers()[1].GetLayerDigest())
}

func (s *BaseImageDataStoreTestSuite) TestUpsertImagesBatch() {
	batch := make(map[*storage.BaseImage][]string)

	// Image 1
	id1 := uuid.NewV4().String()
	img1 := &storage.BaseImage{
		Id:               id1,
		ManifestDigest:   "digestA",
		FirstLayerDigest: "layerA",
	}
	batch[img1] = []string{"layerA"}

	// Image 2
	id2 := uuid.NewV4().String()
	img2 := &storage.BaseImage{
		Id:               id2,
		ManifestDigest:   "digestB",
		FirstLayerDigest: "layerB",
	}
	batch[img2] = []string{"layerB"}

	// Execute Batch Upsert
	err := s.datastore.UpsertImages(s.ctx, batch)
	s.Require().NoError(err)

	fetched1, found, err := s.datastore.GetBaseImage(s.ctx, "digestA")
	s.Require().NoError(err)
	s.True(found)
	s.Equal(id1, fetched1.GetId())
	s.Len(fetched1.GetLayers(), 1)
	s.Equal("layerA", fetched1.GetLayers()[0].GetLayerDigest())

	fetched2, found, err := s.datastore.GetBaseImage(s.ctx, "digestB")
	s.Require().NoError(err)
	s.True(found)
	s.Equal(id2, fetched2.GetId())
	s.Len(fetched2.GetLayers(), 1)
	s.Equal("layerB", fetched2.GetLayers()[0].GetLayerDigest())
}

func (s *BaseImageDataStoreTestSuite) TestListCandidateBaseImages() {
	commonDigest := "sha256:common"
	uniqueDigest := "sha256:unique"

	imgA := &storage.BaseImage{Id: uuid.NewV4().String(), FirstLayerDigest: commonDigest, ManifestDigest: "d1"}
	imgB := &storage.BaseImage{Id: uuid.NewV4().String(), FirstLayerDigest: commonDigest, ManifestDigest: "d2"}
	imgC := &storage.BaseImage{Id: uuid.NewV4().String(), FirstLayerDigest: uniqueDigest, ManifestDigest: "d3"}

	s.Require().NoError(s.datastore.UpsertImage(s.ctx, imgA, []string{commonDigest}))
	s.Require().NoError(s.datastore.UpsertImage(s.ctx, imgB, []string{commonDigest}))
	s.Require().NoError(s.datastore.UpsertImage(s.ctx, imgC, []string{uniqueDigest}))

	candidates, err := s.datastore.ListCandidateBaseImages(s.ctx, commonDigest)
	s.Require().NoError(err)
	s.Len(candidates, 2)

	candidateIDs := []string{candidates[0].GetId(), candidates[1].GetId()}
	s.Contains(candidateIDs, imgA.GetId())
	s.Contains(candidateIDs, imgB.GetId())
	s.NotContains(candidateIDs, imgC.GetId())

	candidatesUnique, err := s.datastore.ListCandidateBaseImages(s.ctx, uniqueDigest)
	s.Require().NoError(err)
	s.Len(candidatesUnique, 1)
	s.Equal(imgC.GetId(), candidatesUnique[0].GetId())

	candidatesNone, err := s.datastore.ListCandidateBaseImages(s.ctx, "sha256:missing")
	s.Require().NoError(err)
	s.Empty(candidatesNone)
}

func (s *BaseImageDataStoreTestSuite) TestGetBaseImageNotFound() {
	img, found, err := s.datastore.GetBaseImage(s.ctx, "sha256:ghost")
	s.NoError(err)
	s.False(found)
	s.Nil(img)
}

func (s *BaseImageDataStoreTestSuite) TestFirstLayerDigestMismatch() {
	id := uuid.NewV4().String()
	img := &storage.BaseImage{
		Id:               id,
		ManifestDigest:   "sha256:mismatch-manifest",
		FirstLayerDigest: "sha256:not-equal",
	}
	digests := []string{"sha256:actual-first"}

	err := s.datastore.UpsertImage(s.ctx, img, digests)
	s.Require().NoError(err, "Upsert should succeed with auto-correction")

	s.Equal("sha256:actual-first", img.GetFirstLayerDigest())
}

func (s *BaseImageDataStoreTestSuite) TestContextCancellation() {
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	err := s.datastore.UpsertImage(ctx, &storage.BaseImage{
		Id:             uuid.NewV4().String(),
		ManifestDigest: "sha256:ctx-cancel",
	}, nil)
	s.Require().Error(err)
}

func (s *BaseImageDataStoreTestSuite) TestReplaceByRepository() {
	repoID := uuid.NewV4().String()
	s.createRepository(repoID, "docker.io/library/nginx", "1.*")

	// Initial images for the repository.
	img1 := &storage.BaseImage{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repoID,
		ManifestDigest:        "sha256:img1",
		Repository:            "docker.io/library/nginx",
		Tag:                   "1.0",
	}
	img2 := &storage.BaseImage{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repoID,
		ManifestDigest:        "sha256:img2",
		Repository:            "docker.io/library/nginx",
		Tag:                   "2.0",
	}

	// Insert initial images.
	initialImages := map[*storage.BaseImage][]string{
		img1: {"sha256:layer1a", "sha256:layer1b"},
		img2: {"sha256:layer2a"},
	}
	err := s.datastore.ReplaceByRepository(s.ctx, repoID, initialImages)
	s.Require().NoError(err)

	// Verify initial images exist.
	images, err := s.datastore.ListByRepository(s.ctx, repoID)
	s.Require().NoError(err)
	s.Len(images, 2)

	// Replace: keep img1, remove img2, add img3.
	img3 := &storage.BaseImage{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repoID,
		ManifestDigest:        "sha256:img3",
		Repository:            "docker.io/library/nginx",
		Tag:                   "3.0",
	}
	replacementImages := map[*storage.BaseImage][]string{
		img1: {"sha256:layer1a", "sha256:layer1b"}, // Kept
		img3: {"sha256:layer3a"},                   // Added
	}
	err = s.datastore.ReplaceByRepository(s.ctx, repoID, replacementImages)
	s.Require().NoError(err)

	// Verify replacement result.
	images, err = s.datastore.ListByRepository(s.ctx, repoID)
	s.Require().NoError(err)
	s.Len(images, 2)

	imageIDs := make(map[string]bool)
	for _, img := range images {
		imageIDs[img.GetId()] = true
	}
	s.True(imageIDs[img1.GetId()], "img1 should be kept")
	s.False(imageIDs[img2.GetId()], "img2 should be deleted")
	s.True(imageIDs[img3.GetId()], "img3 should be added")

	// Verify img2 no longer exists.
	_, found, err := s.datastore.GetBaseImage(s.ctx, "sha256:img2")
	s.Require().NoError(err)
	s.False(found, "img2 should have been deleted")

	// Verify img3 has correct layers.
	retrieved, found, err := s.datastore.GetBaseImage(s.ctx, "sha256:img3")
	s.Require().NoError(err)
	s.True(found)
	s.Len(retrieved.GetLayers(), 1)
	s.Equal("sha256:layer3a", retrieved.GetLayers()[0].GetLayerDigest())
}

func (s *BaseImageDataStoreTestSuite) TestReplaceByRepositoryEmpty() {
	repoID := uuid.NewV4().String()
	s.createRepository(repoID, "docker.io/library/alpine", "latest")

	// Insert initial image.
	img := &storage.BaseImage{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repoID,
		ManifestDigest:        "sha256:to-delete",
		Repository:            "docker.io/library/alpine",
		Tag:                   "latest",
	}
	err := s.datastore.ReplaceByRepository(s.ctx, repoID, map[*storage.BaseImage][]string{
		img: {"sha256:layer"},
	})
	s.Require().NoError(err)

	// Verify image exists.
	images, err := s.datastore.ListByRepository(s.ctx, repoID)
	s.Require().NoError(err)
	s.Len(images, 1)

	// Replace with empty set - should delete all.
	err = s.datastore.ReplaceByRepository(s.ctx, repoID, map[*storage.BaseImage][]string{})
	s.Require().NoError(err)

	// Verify all images deleted.
	images, err = s.datastore.ListByRepository(s.ctx, repoID)
	s.Require().NoError(err)
	s.Empty(images)
}

func (s *BaseImageDataStoreTestSuite) TestReplaceByRepositoryIsolation() {
	repoA := uuid.NewV4().String()
	repoB := uuid.NewV4().String()
	s.createRepository(repoA, "docker.io/library/isolation-nginx-"+repoA[:8], "*")
	s.createRepository(repoB, "docker.io/library/isolation-alpine-"+repoB[:8], "*")

	// Create images in repo A.
	imgA := &storage.BaseImage{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repoA,
		ManifestDigest:        "sha256:imgA",
		Repository:            "docker.io/library/nginx",
		Tag:                   "a",
	}
	err := s.datastore.ReplaceByRepository(s.ctx, repoA, map[*storage.BaseImage][]string{
		imgA: {"sha256:layerA"},
	})
	s.Require().NoError(err)

	// Create images in repo B.
	imgB := &storage.BaseImage{
		Id:                    uuid.NewV4().String(),
		BaseImageRepositoryId: repoB,
		ManifestDigest:        "sha256:imgB",
		Repository:            "docker.io/library/alpine",
		Tag:                   "b",
	}
	err = s.datastore.ReplaceByRepository(s.ctx, repoB, map[*storage.BaseImage][]string{
		imgB: {"sha256:layerB"},
	})
	s.Require().NoError(err)

	// Replace repo A with empty - should not affect repo B.
	err = s.datastore.ReplaceByRepository(s.ctx, repoA, map[*storage.BaseImage][]string{})
	s.Require().NoError(err)

	// Verify repo A is empty.
	imagesA, err := s.datastore.ListByRepository(s.ctx, repoA)
	s.Require().NoError(err)
	s.Empty(imagesA)

	// Verify repo B is untouched.
	imagesB, err := s.datastore.ListByRepository(s.ctx, repoB)
	s.Require().NoError(err)
	s.Len(imagesB, 1)
	s.Equal(imgB.GetId(), imagesB[0].GetId())
}
