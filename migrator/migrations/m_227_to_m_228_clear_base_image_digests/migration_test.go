//go:build sql_integration

package m227tom228

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
)

const (
	repoID = "11111111-1111-1111-1111-111111111111"
	tagID1 = "22222222-2222-2222-2222-222222222222"
	tagID2 = "33333333-3333-3333-3333-333333333333"
)

type m227tom228TestSuite struct {
	suite.Suite
	db *types.Databases
}

func TestM227ToM228(t *testing.T) {
	suite.Run(t, new(m227tom228TestSuite))
}

func (s *m227tom228TestSuite) SetupTest() {
	testPG := postgreshelper.ForT(s.T(), false)
	s.T().Cleanup(func() { testPG.Teardown(s.T()) })
	s.db = &types.Databases{
		GormDB:     testPG.GetGormDB(),
		PostgresDB: testPG.DB,
		DBCtx:      s.T().Context(),
	}
	// Inline the table. migrator/migrations cannot import pkg/postgres/schema.
	// Columns match base_image_tags: digest and history live in serialized.
	_, err := s.db.PostgresDB.Exec(s.T().Context(), `
		CREATE TABLE base_image_tags (
			id UUID PRIMARY KEY,
			baseimagerepositoryid UUID,
			tag VARCHAR,
			serialized BYTEA
		)`)
	s.Require().NoError(err)
}

func (s *m227tom228TestSuite) TestMigration() {
	err := migrate(s.db)
	s.NoError(err)

	err = migrate(s.db)
	s.NoError(err)
}

func (s *m227tom228TestSuite) TestClearsManifestDigests() {
	history := []*storage.BaseImageHistoryEntry{
		{Instruction: "ADD", Value: "rootfs"},
		{Instruction: "USER", Value: "1001", Empty: true},
	}
	s.insertTag(tagID1, "latest", "sha256:abc123", history)
	s.insertTag(tagID2, "v1", "", history)

	s.Require().NoError(migrate(s.db))

	cleared := s.readTag(tagID1)
	s.Empty(cleared.GetManifestDigest())
	protoassert.SlicesEqual(s.T(), history, cleared.GetHistory())
	s.Equal("latest", cleared.GetTag())

	untouched := s.readTag(tagID2)
	s.Empty(untouched.GetManifestDigest())
	protoassert.SlicesEqual(s.T(), history, untouched.GetHistory())

	s.Require().NoError(migrate(s.db))
	s.Empty(s.readTag(tagID1).GetManifestDigest())
}

func (s *m227tom228TestSuite) insertTag(id, name, digest string, history []*storage.BaseImageHistoryEntry) {
	tag := &storage.BaseImageTag{
		Id:                    id,
		BaseImageRepositoryId: repoID,
		Tag:                   name,
		ManifestDigest:        digest,
		History:               history,
	}
	serialized, err := tag.MarshalVT()
	s.Require().NoError(err)
	_, err = s.db.PostgresDB.Exec(s.T().Context(), `
		INSERT INTO base_image_tags (id, baseimagerepositoryid, tag, serialized)
		VALUES ($1, $2, $3, $4)`, id, repoID, name, serialized)
	s.Require().NoError(err)
}

func (s *m227tom228TestSuite) readTag(id string) *storage.BaseImageTag {
	var serialized []byte
	err := s.db.PostgresDB.QueryRow(s.T().Context(), `SELECT serialized FROM base_image_tags WHERE id = $1`, id).Scan(&serialized)
	s.Require().NoError(err)
	tag := &storage.BaseImageTag{}
	s.Require().NoError(tag.UnmarshalVT(serialized))
	return tag
}
