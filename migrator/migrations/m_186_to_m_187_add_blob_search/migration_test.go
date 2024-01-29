//go:build sql_integration

package m186tom187

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	afterSchema "github.com/stackrox/rox/migrator/migrations/m_186_to_m_187_add_blob_search/schema/after"
	beforeSchema "github.com/stackrox/rox/migrator/migrations/m_186_to_m_187_add_blob_search/schema/before"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type blobMigrationTestSuite struct {
	suite.Suite
	ctx context.Context

	db     *pghelper.TestPostgres
	gormDB *gorm.DB
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(blobMigrationTestSuite))
}

func (s *blobMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
	s.ctx = context.Background()
	s.gormDB = s.db.GetGormDB().WithContext(s.ctx)
}

func (s *blobMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *blobMigrationTestSuite) TestMigration() {
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, beforeSchema.CreateTableBlobsStmt)
	batchSize = 9
	blobs := s.createMockBlobs(20)
	var convertedBlobs []beforeSchema.Blobs
	for _, blob := range blobs {
		converted, err := beforeSchema.ConvertBlobFromProto(blob)
		s.Require().NoError(err)
		convertedBlobs = append(convertedBlobs, *converted)
	}
	s.Require().NoError(s.gormDB.Create(convertedBlobs).Error)
	s.Require().NoError(convert(s.gormDB))

	var newBlobs []afterSchema.Blobs
	s.Require().NoError(s.gormDB.Find(&newBlobs).Error)
	s.Equal(len(blobs), len(newBlobs))
	for i, nb := range newBlobs {
		s.Contains(blobs, nb.Name)
		blob := blobs[nb.Name]
		s.Equal(blob.Length, nb.Length)
		s.Equal(pgutils.NilOrTime(blob.ModifiedTime), nb.ModifiedTime)
		converted, err := afterSchema.ConvertBlobToProto(&newBlobs[i])
		s.Require().NoError(err)
		s.Equal(blob, converted)
	}
}

func (s *blobMigrationTestSuite) createMockBlobs(size int) map[string]*storage.Blob {
	blobs := make(map[string]*storage.Blob, size)
	for i := 0; i < size; i++ {
		oid := uint32(80000 + size)
		name := fmt.Sprintf("/path/mock/oid_%d", oid)
		blob := &storage.Blob{
			Name:         name,
			Oid:          oid,
			Length:       int64(rand.Int()),
			ModifiedTime: timestamp.TimestampNow(),
		}
		blobs[name] = blob
	}
	return blobs
}
