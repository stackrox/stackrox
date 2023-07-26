//go:build sql_integration

package datastore

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/central/blob/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/suite"
)

type blobTestSuite struct {
	suite.Suite
	ctx       context.Context
	store     store.Store
	datastore Datastore
	testDB    *pgtest.TestPostgres
}

func TestBlobsStore(t *testing.T) {
	suite.Run(t, new(blobTestSuite))
}

func (s *blobTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	s.store = store.New(s.testDB.DB)
	s.datastore = NewDatastore(s.store, search.New(s.store, postgres.NewIndexer(s.testDB.DB)))
}

func (s *blobTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *blobTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *blobTestSuite) createBlobs(prefix string, size int, n int, modTime *timestamp.Timestamp) []*storage.Blob {
	var blobs []*storage.Blob
	for i := 0; i < n; i++ {
		blob := &storage.Blob{
			Name:         fmt.Sprintf("%s/test/%d", prefix, i),
			ModifiedTime: modTime,
			Length:       int64(size),
		}

		randomData := make([]byte, size)
		_, err := rand.Read(randomData)
		s.NoError(err)

		reader := bytes.NewBuffer(randomData)

		s.Require().NoError(s.store.Upsert(s.ctx, blob, reader))
		blobs = append(blobs, blob)
	}
	return blobs
}

func (s *blobTestSuite) TestSearch() {
	searchTime := protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2020-03-09T12:00:00Z"))
	blobs1 := s.createBlobs("/path1", 10, 2, searchTime)
	blobs2 := s.createBlobs("/path2", 20, 3, timestamp.TimestampNow())
	blobsResults, err := s.datastore.Search(s.ctx, pkgSearch.EmptyQuery())
	s.Require().NoError(err)
	s.Equal(len(blobs1)+len(blobs2), len(blobsResults))

	blobs, err := s.datastore.SearchMetadata(s.ctx, pkgSearch.NewQueryBuilder().AddDocIDs(blobs2[0].GetName()).ProtoQuery())
	s.Require().NoError(err)
	s.Len(blobs, 1)
	s.Equal(blobs2[0].GetName(), blobs[0].GetName())

	blobs, err = s.datastore.SearchMetadata(s.ctx, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.BlobLength, "20").ProtoQuery())
	s.Require().NoError(err)
	s.ElementsMatch(blobs2, blobs)

	blobs, err = s.datastore.SearchMetadata(s.ctx, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.BlobModificationTime, "03/09/2020 UTC").ProtoQuery())
	s.Require().NoError(err)
	s.ElementsMatch(blobs1, blobs)

	blobs, err = s.datastore.SearchMetadata(s.ctx, pkgSearch.NewQueryBuilder().AddRegexes(pkgSearch.BlobName, "/path1/.+").ProtoQuery())
	s.Require().NoError(err)
	s.ElementsMatch(blobs1, blobs)
}
