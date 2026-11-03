//go:build sql_integration

package m000tom001

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
)

const (
	repoID = "11111111-1111-1111-1111-111111111111"
	tagID1 = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	tagID2 = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	tagID3 = "cccccccc-cccc-cccc-cccc-cccccccccccc"
)

type migrationTestSuite struct {
	suite.Suite
	db *pgtest.TestPostgres
}

func TestClearBaseImageTagDigests(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	s.T().Cleanup(func() {
		batchSize = 500
		afterBatchForTest = nil
	})

	repo, err := (&storage.BaseImageRepository{Id: repoID, RepositoryPath: "library/ubi"}).MarshalVT()
	s.Require().NoError(err)
	_, err = s.db.Exec(s.T().Context(), `
		INSERT INTO base_image_repositories (id, repositorypath, createdby_id, createdby_name, serialized)
		VALUES ($1, $2, '', '', $3)`, repoID, "library/ubi", repo)
	s.Require().NoError(err)
}

func (s *migrationTestSuite) TestClearsDigestOnlyWhenHistoryIsEmpty() {
	history := []*storage.BaseImageHistoryEntry{
		{Instruction: "ADD", Value: "rootfs"},
	}
	s.insertTag(tagID1, "9.6", "sha256:old", nil, []string{"sha256:layer"})
	s.insertTag(tagID2, "9.7", "sha256:fresh", history, nil)
	s.insertTag(tagID3, "9.8", "", nil, nil)

	s.Require().NoError(run(s.T().Context(), s.db))

	cleared := s.readTag(tagID1)
	s.Empty(cleared.GetManifestDigest())
	s.Empty(cleared.GetHistory())
	s.Equal("9.6", cleared.GetTag())
	s.Equal([]string{"sha256:layer"}, cleared.GetLayerDigests())

	refetched := s.readTag(tagID2)
	s.Equal("sha256:fresh", refetched.GetManifestDigest())
	protoassert.SlicesEqual(s.T(), history, refetched.GetHistory())

	s.Empty(s.readTag(tagID3).GetManifestDigest())

	s.Require().NoError(run(s.T().Context(), s.db))
	s.Empty(s.readTag(tagID1).GetManifestDigest())
	s.Equal("sha256:fresh", s.readTag(tagID2).GetManifestDigest())
}

func (s *migrationTestSuite) TestPagesByPrimaryKey() {
	batchSize = 1
	s.insertTag(tagID1, "9.6", "sha256:one", nil, nil)
	s.insertTag(tagID2, "9.7", "sha256:two", nil, nil)
	s.insertTag(tagID3, "9.8", "sha256:three", nil, nil)

	s.Require().NoError(run(s.T().Context(), s.db))

	s.Empty(s.readTag(tagID1).GetManifestDigest())
	s.Empty(s.readTag(tagID2).GetManifestDigest())
	s.Empty(s.readTag(tagID3).GetManifestDigest())
}

func (s *migrationTestSuite) TestCancelBetweenBatchesLeavesCommittedRows() {
	batchSize = 1
	s.insertTag(tagID1, "9.6", "sha256:one", nil, nil)
	s.insertTag(tagID2, "9.7", "sha256:two", nil, nil)
	s.insertTag(tagID3, "9.8", "sha256:three", nil, nil)

	ctx, cancel := context.WithCancel(s.T().Context())
	defer cancel()
	afterBatchForTest = func() {
		cancel()
	}

	err := run(ctx, s.db)
	s.ErrorIs(err, context.Canceled)
	s.Empty(s.readTag(tagID1).GetManifestDigest())
	s.Equal("sha256:two", s.readTag(tagID2).GetManifestDigest())
	s.Equal("sha256:three", s.readTag(tagID3).GetManifestDigest())

	afterBatchForTest = nil
	s.Require().NoError(run(s.T().Context(), s.db))
	s.Empty(s.readTag(tagID2).GetManifestDigest())
	s.Empty(s.readTag(tagID3).GetManifestDigest())
}

func (s *migrationTestSuite) insertTag(id, name, digest string, history []*storage.BaseImageHistoryEntry, layers []string) {
	tag := &storage.BaseImageTag{
		Id:                    id,
		BaseImageRepositoryId: repoID,
		Tag:                   name,
		ManifestDigest:        digest,
		History:               history,
		LayerDigests:          layers,
	}
	serialized, err := tag.MarshalVT()
	s.Require().NoError(err)
	_, err = s.db.Exec(s.T().Context(), `
		INSERT INTO base_image_tags (id, baseimagerepositoryid, tag, serialized)
		VALUES ($1, $2, $3, $4)`, id, repoID, name, serialized)
	s.Require().NoError(err)
}

func (s *migrationTestSuite) readTag(id string) *storage.BaseImageTag {
	var serialized []byte
	err := s.db.QueryRow(s.T().Context(), `SELECT serialized FROM base_image_tags WHERE id = $1`, id).Scan(&serialized)
	s.Require().NoError(err)
	tag := &storage.BaseImageTag{}
	s.Require().NoError(tag.UnmarshalVT(serialized))
	return tag
}
