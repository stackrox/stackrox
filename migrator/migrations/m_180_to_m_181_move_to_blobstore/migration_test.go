//go:build sql_integration

package m180tom181

import (
	"bytes"
	"crypto/rand"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_move_to_blobstore/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/postgres/gorm/largeobject"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/suite"
)

const (
	validFilePath = "2.4.0/collector-4.9.24-coreos.ko.gz"
)

type blobMigrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(blobMigrationTestSuite))
}

func (s *blobMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
}

func (s *blobMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *blobMigrationTestSuite) TestScannerDefinitionMigration() {
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
	fileInfo, err := file.Stat()
	s.Require().NoError(err)

	// Migrate
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))

	// Verify Blob
	blobModel := &schema.Blobs{Name: scannerDefBlobName}
	s.Require().NoError(s.db.GetGormDB().First(&blobModel).Error)

	blob, err := schema.ConvertBlobToProto(blobModel)
	s.Require().NoError(err)
	s.Equal(scannerDefBlobName, blob.GetName())
	s.EqualValues(size, blob.GetLength())

	modTime := pgutils.NilOrTime(blob.GetModifiedTime())
	s.Equal(fileInfo.ModTime().UTC().Round(time.Microsecond), modTime.UTC().Round(time.Microsecond))

	// Verify Data
	buf := bytes.NewBuffer(nil)

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

func (s *blobMigrationTestSuite) TestUploadProbeMigration() {
	// Nothing to migrate
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))
	rootDir, err := os.MkdirTemp("", "move-blob")
	s.Require().NoError(err)
	defer func() { _ = os.RemoveAll(rootDir) }()
	uploadProbeRoot = rootDir

	// Prepare persistent file
	data := []byte("foobarbaz")
	crc32Sum := crc32.ChecksumIEEE(data)

	dir := path.Join(rootDir, validFilePath)
	s.Require().NoError(os.MkdirAll(dir, 0700))
	s.Require().NoError(os.WriteFile(filepath.Join(dir, dataFileName), data, 0600))
	s.Require().NoError(os.WriteFile(filepath.Join(dir, crc32FileName), binenc.BigEndian.EncodeUint32(crc32Sum), 0600))

	// Migrate
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))

	// Verify Blob
	blobName := path.Join(uploadProbeBlobRoot, validFilePath)
	blobModel := &schema.Blobs{Name: blobName}
	s.Require().NoError(s.db.GetGormDB().First(&blobModel).Error)

	blob, err := schema.ConvertBlobToProto(blobModel)
	s.Require().NoError(err)
	s.Equal(blobName, blob.GetName())
	s.EqualValues(len(data), blob.GetLength())

	// Verify Data
	buf := bytes.NewBuffer(nil)

	tx := s.db.GetGormDB().Begin()
	s.Require().NoError(err)
	los := &largeobject.LargeObjects{DB: tx}
	s.Require().NoError(los.Get(blob.Oid, buf))
	s.Equal(binenc.BigEndian.EncodeUint32(crc32Sum), []byte(blob.Checksum))
	s.Equal(len(data), buf.Len())
	s.Equal(data, buf.Bytes())
	s.NoError(tx.Commit().Error)

	// Test re-entry
	s.Require().NoError(moveToBlobs(s.db.GetGormDB()))
	buf.Reset()
	tx = s.db.GetGormDB().Begin()
	los = &largeobject.LargeObjects{DB: tx}
	s.Require().NoError(err)
	s.Require().NoError(los.Get(blob.Oid, buf))
	s.Equal(binenc.BigEndian.EncodeUint32(crc32Sum), []byte(blob.Checksum))
	s.Equal(len(data), buf.Len())
	s.Equal(data, buf.Bytes())
	s.NoError(tx.Commit().Error)
}
