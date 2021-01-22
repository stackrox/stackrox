package datastore

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	imageComponentEdgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	imageComponentEdgeStore "github.com/stackrox/rox/central/imagecomponentedge/store/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	edgeID "github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestImageComponentEdgeDataStore(t *testing.T) {
	suite.Run(t, new(ImageComponentEdgeDataStoreTestSuite))
}

type ImageComponentEdgeDataStoreTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	blevePath string
	indexQ    queue.WaitableQueue
	datastore DataStore
}

func (suite *ImageComponentEdgeDataStoreTestSuite) SetupSuite() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())

	suite.indexQ = queue.NewWaitableQueue()

	dacky, err := dackbox.NewRocksDBDackBox(suite.db, suite.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNow("failed to create dackbox", err.Error())
	}

	suite.blevePath, err = ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("failed to create dir for bleve", err.Error())
	}
	blevePath := filepath.Join(suite.blevePath, "scorch.bleve")
	bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex, "")
	if err != nil {
		suite.FailNow("failed to create bleve index", err.Error())
	}

	reg := indexer.NewWrapperRegistry()
	indexer.NewLazy(suite.indexQ, reg, bleveIndex, dacky.AckIndexed).Start()
	reg.RegisterWrapper(imageComponentEdgeDackBox.Bucket, imageComponentEdgeIndex.Wrapper{})

	edgeStore, _ := imageComponentEdgeStore.New(dacky)
	index := imageComponentEdgeIndex.New(bleveIndex)
	suite.datastore, _ = New(dacky, edgeStore, index, search.New(edgeStore, index))
}

func (suite *ImageComponentEdgeDataStoreTestSuite) TearDownSuite() {
	_ = os.RemoveAll(suite.blevePath)
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageComponentEdgeDataStoreTestSuite) TestBasicOps() {
	id := edgeID.EdgeID{
		ParentID: "id1",
		ChildID:  "comp1",
	}.ToString()
	edge := getTestImageComponentEdge(id)

	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		))
	suite.Error(suite.datastore.Upsert(readCtx, edge), "permission denied")

	// No permission to write edges.
	imgCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node),
		))
	suite.Error(suite.datastore.Upsert(imgCtx, edge), "permission denied")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Upsert edge.
	suite.NoError(suite.datastore.Upsert(ctx, edge))

	// Get edge.
	storedEdge, exists, err := suite.datastore.Get(ctx, id)
	suite.True(exists)
	suite.NoError(err)
	suite.Equal(edge, storedEdge)

	// Exists tests.
	exists, err = suite.datastore.Exists(ctx, id)
	suite.NoError(err)
	suite.True(exists)

	id2 := edgeID.EdgeID{
		ParentID: "id2",
		ChildID:  "comp1",
	}.ToString()

	exists, err = suite.datastore.Exists(ctx, id2)
	suite.NoError(err)
	suite.False(exists)

	newEdge := getTestImageComponentEdge(id2)

	// Upsert new edge.
	suite.NoError(suite.datastore.Upsert(ctx, newEdge))

	// Exists test.
	exists, err = suite.datastore.Exists(ctx, id2)
	suite.NoError(err)
	suite.True(exists)

	// Get new edge.
	storedEdge, exists, err = suite.datastore.Get(ctx, newEdge.Id)
	suite.True(exists)
	suite.NoError(err)
	suite.Equal(newEdge, storedEdge)

	// Count edges.
	count, err := suite.datastore.Count(ctx)
	suite.NoError(err)
	suite.Equal(2, count)

	// Get batch.
	edges, err := suite.datastore.GetBatch(ctx, []string{id, id2})
	suite.NoError(err)
	suite.Len(edges, 2)
	suite.ElementsMatch([]*storage.ImageComponentEdge{edge, newEdge}, edges)

	// Delete both edges.
	suite.NoError(suite.datastore.Delete(ctx, id, id2))

	// Exists tests.
	exists, err = suite.datastore.Exists(ctx, id)
	suite.NoError(err)
	suite.False(exists)
	exists, err = suite.datastore.Exists(ctx, id2)
	suite.NoError(err)
	suite.False(exists)
}

func (suite *ImageComponentEdgeDataStoreTestSuite) TestBasicSearch() {
	id1 := edgeID.EdgeID{
		ParentID: "id1",
		ChildID:  "comp1",
	}.ToString()
	edge := getTestImageComponentEdge(id1)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Basic unscoped search.
	results, err := suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	// Upsert edge.
	suite.NoError(suite.datastore.Upsert(ctx, edge))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Basic unscoped search.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    edge.GetId(),
		Level: v1.SearchCategory_IMAGE_COMPONENT_EDGE,
	})

	// Basic scoped search.
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	// Search edges.
	edges, err := suite.datastore.SearchRawEdges(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.NotNil(edges)
	suite.Len(edges, 1)

	suite.Equal(edge, edges[0])

	// Upsert new edge.
	id2 := edgeID.EdgeID{
		ParentID: "id2",
		ChildID:  "comp1",
	}.ToString()
	newEdge := getTestImageComponentEdge(id2)
	suite.NoError(suite.datastore.Upsert(ctx, newEdge))

	// Ensure the CVEs are indexed.
	indexingDone = concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Search multiple edges.
	edges, err = suite.datastore.SearchRawEdges(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(edges, 2)

	suite.deleteTestEdges(ctx)

	// Ensure search does not find anything.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func getTestImageComponentEdge(id string) *storage.ImageComponentEdge {
	return &storage.ImageComponentEdge{
		Id: id,
	}
}

func (suite *ImageComponentEdgeDataStoreTestSuite) deleteTestEdges(ctx context.Context) {
	id1 := edgeID.EdgeID{
		ParentID: "id1",
		ChildID:  "comp1",
	}.ToString()
	id2 := edgeID.EdgeID{
		ParentID: "id2",
		ChildID:  "comp1",
	}.ToString()
	suite.NoError(suite.datastore.Delete(ctx, id1, id2))

	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()
}
