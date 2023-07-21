//go:build sql_integration

package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

type BlobsStoreSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
}

func TestBlobsStore(t *testing.T) {
	suite.Run(t, new(BlobsStoreSuite))
}

func (s *BlobsStoreSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
}

func (s *BlobsStoreSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	tag, err := s.testDB.Exec(ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *BlobsStoreSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *BlobsStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())
	size := 1024*1024 + 16

	insertBlob := &storage.Blob{
		Name:         "test",
		Length:       int64(size),
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
	}

	buf := &bytes.Buffer{}
	_, exists, err := s.store.Get(ctx, insertBlob.GetName(), buf)
	s.Require().NoError(err)
	s.Require().False(exists)

	randomData := make([]byte, size)
	_, err = rand.Read(randomData)
	s.NoError(err)

	reader := bytes.NewBuffer(randomData)

	s.Require().NoError(s.store.Upsert(ctx, insertBlob, reader))

	buf = &bytes.Buffer{}
	blob, exists, err := s.store.Get(ctx, insertBlob.GetName(), buf)
	s.Require().NoError(err)
	s.Require().True(exists)
	s.NotZero(blob.GetOid())
	s.verifyLargeObjectCounts(1)
	s.Equal(insertBlob, blob)
	s.Equal(randomData, buf.Bytes())

	s.NoError(s.store.Delete(ctx, insertBlob.GetName()))

	buf.Truncate(0)
	blob, exists, err = s.store.Get(ctx, insertBlob.GetName(), buf)
	s.Require().NoError(err)
	s.Require().False(exists)
	s.Zero(blob.GetOid())
	s.Nil(blob)
	s.Zero(buf.Len())
	s.verifyLargeObjectCounts(0)
}

func (s *BlobsStoreSuite) TestSacForUpsertAndDelete() {
	rwCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	rCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	wCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	size := 1024

	insertBlob := &storage.Blob{
		Name:         "test",
		Length:       int64(size),
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
	}

	randomData := make([]byte, size)
	_, err := rand.Read(randomData)
	s.NoError(err)

	reader := bytes.NewBuffer(randomData)

	// Upsert fails with read permission
	s.Require().Error(s.store.Upsert(rCtx, insertBlob, reader))
	s.verifyLargeObjectCounts(0)
	// Upsert succeeds with write permission
	s.Require().NoError(s.store.Upsert(wCtx, insertBlob, reader))
	s.verifyLargeObjectCounts(1)
	// Upsert succeeds updating blob
	reader = bytes.NewBuffer(randomData)
	s.Require().NoError(s.store.Upsert(wCtx, insertBlob, reader))
	s.verifyLargeObjectCounts(1)
	// Upsert succeeds with read and write permission
	reader = bytes.NewBuffer(randomData)
	s.Require().NoError(s.store.Upsert(rwCtx, insertBlob, reader))
	s.verifyLargeObjectCounts(1)

	// Delete fails with read permission
	s.Require().Error(s.store.Delete(rCtx, insertBlob.GetName()))
	s.verifyLargeObjectCounts(1)
	// Delete succeeds with write permission
	s.Require().NoError(s.store.Delete(wCtx, insertBlob.GetName()))
	s.verifyLargeObjectCounts(0)
	// Delete succeeds with read and write permission
	reader = bytes.NewBuffer(randomData)
	s.Require().NoError(s.store.Upsert(rwCtx, insertBlob, reader))
	s.verifyLargeObjectCounts(1)
	s.Require().NoError(s.store.Delete(rwCtx, insertBlob.GetName()))
	s.verifyLargeObjectCounts(0)
}

func (s *BlobsStoreSuite) verifyLargeObjectCounts(expected int) {
	ctx := context.Background()
	tx, err := s.testDB.DB.Begin(context.Background())
	s.Require().NoError(err)

	defer func() { _ = tx.Rollback(ctx) }()

	var n int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM pg_largeobject_metadata;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(expected, n)
}
