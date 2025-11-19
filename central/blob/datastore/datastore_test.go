//go:build sql_integration

package datastore

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/central/reports/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
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
	s.datastore = NewDatastore(s.store)
}

func (s *blobTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
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

		s.Require().NoError(s.datastore.Upsert(s.ctx, blob, reader))
		blobs = append(blobs, blob)
	}
	return blobs
}

func (s *blobTestSuite) TestSearch() {
	searchTime := timeutil.MustParse(time.RFC3339, "2020-03-09T12:00:00Z")
	blobs1 := s.createBlobs(common.ReportBlobPathPrefix, 10, 2, searchTime)
	blobs2 := s.createBlobs(common.ComplianceReportBlobPathPrefix, 20, 3, time.Now())

	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddDocIDs(blobs2[0].GetName()).ProtoQuery(), []*storage.Blob{blobs2[0]}, nil)
	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.BlobLength, "20").ProtoQuery(), blobs2, nil)
	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.BlobModificationTime, "03/09/2020 UTC").ProtoQuery(), blobs1, nil)
	s.testQuery(s.ctx, pkgSearch.NewQueryBuilder().AddRegexes(pkgSearch.BlobName, common.ReportBlobRegex).ProtoQuery(), append(blobs1, blobs2...), nil)

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
	expectedLength := len(expected)
	s.Len(blobs, expectedLength)
	protoassert.ElementsMatch(s.T(), expected, blobs)

	results, err := s.datastore.Search(ctx, q)
	s.Require().Equal(err, errExpected)
	s.Len(results, expectedLength)
	idSet := pkgSearch.ResultsToIDSet(results)
	for _, e := range expected {
		s.Contains(idSet, e.GetName())
	}

	ids, err := s.datastore.SearchIDs(ctx, q)
	s.Require().Equal(err, errExpected)
	s.Len(ids, expectedLength)
	s.ElementsMatch(idSet.AsSlice(), ids)
}

func (s *blobTestSuite) TestUpsert() {
	// Test initial upsert (create)
	blob := &storage.Blob{
		Name:         "test/upsert/blob",
		ModifiedTime: protoconv.MustConvertTimeToTimestamp(time.Now()),
		Length:       100,
	}

	originalData := make([]byte, 100)
	_, err := rand.Read(originalData)
	s.NoError(err)
	reader := bytes.NewBuffer(originalData)

	err = s.datastore.Upsert(s.ctx, blob, reader)
	s.NoError(err)

	// Verify blob was created
	metadata, exists, err := s.datastore.GetMetadata(s.ctx, blob.GetName())
	s.NoError(err)
	s.True(exists)
	s.Equal(blob.GetName(), metadata.GetName())
	s.Equal(blob.GetLength(), metadata.GetLength())

	// Test update via upsert
	updatedTime := time.Now().Add(time.Hour)
	blob.ModifiedTime = protoconv.MustConvertTimeToTimestamp(updatedTime)
	blob.Length = 150

	updatedData := make([]byte, 150)
	_, err = rand.Read(updatedData)
	s.NoError(err)
	reader = bytes.NewBuffer(updatedData)

	err = s.datastore.Upsert(s.ctx, blob, reader)
	s.NoError(err)

	// Verify blob was updated
	metadata, exists, err = s.datastore.GetMetadata(s.ctx, blob.GetName())
	s.NoError(err)
	s.True(exists)
	s.Equal(blob.GetName(), metadata.GetName())
	s.Equal(blob.GetLength(), metadata.GetLength())

	// Test upsert with unauthorized context
	unauthorizedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))

	err = s.datastore.Upsert(unauthorizedCtx, blob, bytes.NewBuffer(updatedData))
	s.Error(err)
}

func (s *blobTestSuite) TestGetBlobWithDataInBuffer() {
	// Create a blob with small data (within buffer limit)
	blob := &storage.Blob{
		Name:         "test/buffer/small",
		ModifiedTime: protoconv.MustConvertTimeToTimestamp(time.Now()),
		Length:       1024, // 1KB - well within 5MB limit
	}

	originalData := make([]byte, 1024)
	_, err := rand.Read(originalData)
	s.NoError(err)
	reader := bytes.NewBuffer(originalData)

	err = s.datastore.Upsert(s.ctx, blob, reader)
	s.NoError(err)

	// Test GetBlobWithDataInBuffer
	buffer, metadata, exists, err := s.datastore.GetBlobWithDataInBuffer(s.ctx, blob.GetName())
	s.NoError(err)
	s.True(exists)
	s.NotNil(buffer)
	s.NotNil(metadata)
	s.Equal(blob.GetName(), metadata.GetName())
	s.Equal(blob.GetLength(), metadata.GetLength())
	s.Equal(originalData, buffer.Bytes())

	// Test with non-existent blob
	_, metadata, exists, err = s.datastore.GetBlobWithDataInBuffer(s.ctx, "nonexistent/blob")
	s.NoError(err)
	s.False(exists)
	s.Nil(metadata)
}

func (s *blobTestSuite) TestGetMetadata() {
	// Create a test blob
	blob := &storage.Blob{
		Name:         "test/metadata/blob",
		ModifiedTime: protoconv.MustConvertTimeToTimestamp(time.Now()),
		Length:       512,
	}

	data := make([]byte, 512)
	_, err := rand.Read(data)
	s.NoError(err)
	reader := bytes.NewBuffer(data)

	err = s.datastore.Upsert(s.ctx, blob, reader)
	s.NoError(err)

	// Test GetMetadata for existing blob
	metadata, exists, err := s.datastore.GetMetadata(s.ctx, blob.GetName())
	s.NoError(err)
	s.True(exists)
	s.NotNil(metadata)
	s.Equal(blob.GetName(), metadata.GetName())
	s.Equal(blob.GetLength(), metadata.GetLength())

	// Test GetMetadata for non-existent blob
	metadata, exists, err = s.datastore.GetMetadata(s.ctx, "nonexistent/blob")
	s.NoError(err)
	s.False(exists)
	s.Nil(metadata)
}

func (s *blobTestSuite) TestGetIDs() {
	// Test with no blobs
	ids, err := s.datastore.GetIDs(s.ctx)
	s.NoError(err)
	s.Empty(ids)

	// Create multiple blobs
	blobs := s.createBlobs("test/ids", 100, 3, time.Now())

	// Test GetIDs with blobs present
	ids, err = s.datastore.GetIDs(s.ctx)
	s.NoError(err)
	s.Len(ids, 3)

	expectedIDs := make([]string, len(blobs))
	for i, blob := range blobs {
		expectedIDs[i] = blob.GetName()
	}
	s.ElementsMatch(expectedIDs, ids)
}

func (s *blobTestSuite) TestDelete() {
	// Create a test blob
	blob := &storage.Blob{
		Name:         "test/delete/blob",
		ModifiedTime: protoconv.MustConvertTimeToTimestamp(time.Now()),
		Length:       256,
	}

	data := make([]byte, 256)
	_, err := rand.Read(data)
	s.NoError(err)
	reader := bytes.NewBuffer(data)

	err = s.datastore.Upsert(s.ctx, blob, reader)
	s.NoError(err)

	// Verify blob exists
	metadata, exists, err := s.datastore.GetMetadata(s.ctx, blob.GetName())
	s.NoError(err)
	s.True(exists)
	s.NotNil(metadata)

	// Test Delete
	err = s.datastore.Delete(s.ctx, blob.GetName())
	s.NoError(err)

	// Verify blob no longer exists
	metadata, exists, err = s.datastore.GetMetadata(s.ctx, blob.GetName())
	s.NoError(err)
	s.False(exists)
	s.Nil(metadata)

	// Test Delete non-existent blob (should not error)
	err = s.datastore.Delete(s.ctx, "nonexistent/blob")
	s.NoError(err)

	// Test with unauthorized context
	unauthorizedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))

	err = s.datastore.Delete(unauthorizedCtx, "some/blob")
	s.Error(err)
}
