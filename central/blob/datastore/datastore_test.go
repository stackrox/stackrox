//go:build sql_integration

package datastore

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
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
	s.datastore = NewDatastore(s.store, search.New(s.store))
}

func (s *blobTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *blobTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *blobTestSuite) createBlobs(prefix string, size int, n int, modTime time.Time) []*storage.Blob {
	var blobs []*storage.Blob
	for i := 0; i < n; i++ {
		blob := &storage.Blob{
			Name:         fmt.Sprintf("%s/test/%d", prefix, i),
			ModifiedTime: protoconv.MustConvertTimeToTimestamp(modTime),
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
	searchTime := timeutil.MustParse(time.RFC3339, "2020-03-09T12:00:00Z")
	blobs1 := s.createBlobs("/path1", 10, 2, searchTime)
	blobs2 := s.createBlobs("/path2", 20, 3, time.Now())

	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddDocIDs(blobs2[0].GetName()).ProtoQuery(), []*storage.Blob{blobs2[0]}, nil)
	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.BlobLength, "20").ProtoQuery(), blobs2, nil)
	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.BlobModificationTime, "03/09/2020 UTC").ProtoQuery(), blobs1, nil)
	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddRegexes(pkgSearch.BlobName, "/path1/.+").ProtoQuery(), blobs1, nil)

	// Global access context without access to Blob
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	s.testQuery(ctx, pkgSearch.NewQueryBuilder().AddDocIDs(blobs2[0].GetName()).ProtoQuery(), nil, errox.NotAuthorized)
}

func (s *blobTestSuite) testQuery(ctx context.Context, q *v1.Query, expected []*storage.Blob, errExpected error) {
	blobs, err := s.datastore.SearchMetadata(ctx, q)
	s.Require().Equal(err, errExpected)
	s.Len(blobs, len(expected))
	s.ElementsMatch(expected, blobs)

	results, err := s.datastore.Search(ctx, q)
	s.Require().Equal(err, errExpected)
	s.Len(results, len(expected))
	idSet := pkgSearch.ResultsToIDSet(results)
	for _, e := range expected {
		s.Contains(idSet, e.GetName())
	}

	ids, err := s.datastore.SearchIDs(ctx, q)
	s.Require().Equal(err, errExpected)
	s.Len(ids, len(expected))
	s.ElementsMatch(idSet.AsSlice(), ids)
}
