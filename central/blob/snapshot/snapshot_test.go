//go:build sql_integration

package snapshot

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type snapshotTestSuite struct {
	suite.Suite
	ctx       context.Context
	store     store.Store
	datastore datastore.Datastore
	testDB    *pgtest.TestPostgres
}

func TestBlobsStoreSnapshot(t *testing.T) {
	suite.Run(t, new(snapshotTestSuite))
}

func (s *snapshotTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	s.store = store.New(s.testDB.DB)
	s.datastore = datastore.NewDatastore(s.store, nil)
}

func (s *snapshotTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *snapshotTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *snapshotTestSuite) TestSnapshot() {
	ctx := sac.WithAllAccess(context.Background())
	size := 1024*1024 + 16
	insertBlob := &storage.Blob{
		Name:         "test",
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
		Length:       int64(size),
	}

	randomData := make([]byte, size)
	_, err := rand.Read(randomData)
	s.NoError(err)

	reader := bytes.NewBuffer(randomData)

	s.Require().NoError(s.store.Upsert(ctx, insertBlob, reader))

	snap, err := TakeBlobSnapshot(ctx, s.datastore, insertBlob.GetName())
	s.NoError(err)
	defer func() {
		s.NoError(snap.Close())
		s.NoFileExists(snap.Name())
	}()
	bytes, err := io.ReadAll(snap)
	s.Require().NoError(err)
	s.Equal(randomData, bytes)
	s.Equal(insertBlob, snap.GetBlob())
	s.FileExists(snap.Name())
}
