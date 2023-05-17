//go:build sql_integration

package m179tom180

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"testing"

	"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_move_to_blobstore/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/gorm/largeobject"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/suite"
)

type categoriesMigrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(categoriesMigrationTestSuite))
}

func (s *categoriesMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
}

func (s *categoriesMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *categoriesMigrationTestSuite) TestMigration() {
	// Nothing to migrate
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))

	// Prepare persistent file
	size := 90000
	randomData := make([]byte, size)
	_, err := rand.Read(randomData)
	s.Require().NoError(err)
	reader := bytes.NewBuffer(randomData)

	file, err := os.CreateTemp("", "move-blob")
	s.Require().NoError(err)
	defer func() {
		s.NoError(file.Close())
		s.NoError(os.Remove(file.Name()))
	}()
	scannerDefPath = file.Name()
	n, err := io.Copy(file, reader)
	s.Require().NoError(err)

	s.Require().EqualValues(size, n)

	// Migrate
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))

	// Verify Blob
	blobModel := &schema.Blobs{Name: scannerDefBlobName}
	s.Require().NoError(s.db.GetGormDB().First(&blobModel).Error)

	blob, err := schema.ConvertBlobToProto(blobModel)
	s.Require().NoError(err)
	s.Equal(scannerDefBlobName, blob.GetName())
	s.EqualValues(size, blob.GetLength())

	fileInfo, err := file.Stat()
	s.Require().NoError(err)
	modTime := pgutils.NilOrTime(blob.GetModifiedTime())
	s.Equal(fileInfo.ModTime().UTC(), modTime.UTC())

	// Verify Data
	buf := bytes.NewBuffer([]byte{})

	tx := s.db.GetGormDB().Begin()
	s.Require().NoError(err)
	los := &largeobject.LargeObjects{DB: tx}
	s.Require().NoError(los.Get(blob.Oid, buf))
	s.Equal(len(randomData), buf.Len())
	s.Equal(randomData, buf.Bytes())
	s.NoError(tx.Commit().Error)

	// Test re-entry
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))
	buf.Reset()
	tx = s.db.GetGormDB().Begin()
	los = &largeobject.LargeObjects{DB: tx}
	s.Require().NoError(err)
	s.Require().NoError(los.Get(blob.Oid, buf))
	s.Equal(len(randomData), buf.Len())
	s.Equal(randomData, buf.Bytes())
	s.NoError(tx.Commit().Error)
}
