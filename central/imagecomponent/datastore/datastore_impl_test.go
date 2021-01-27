package datastore

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	clusterIndexer "github.com/stackrox/rox/central/cluster/index"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	componentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	componentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	componentStore "github.com/stackrox/rox/central/imagecomponent/store/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestImageComponentDataStore(t *testing.T) {
	suite.Run(t, new(ImageComponentDataStoreTestSuite))
}

type ImageComponentDataStoreTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	blevePath string
	indexQ    queue.WaitableQueue
	datastore DataStore

	mockRisk *mockRisks.MockDataStore
}

func (suite *ImageComponentDataStoreTestSuite) SetupSuite() {
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
	reg.RegisterWrapper(componentDackBox.Bucket, componentIndex.Wrapper{})

	suite.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(suite.T()))

	store, _ := componentStore.New(dacky, concurrency.NewKeyFence())
	index := componentIndex.New(bleveIndex)
	searcher := componentSearch.New(store, dacky, cveIndex.New(bleveIndex), componentCVEEdgeIndex.New(bleveIndex), index,
		imageComponentEdgeIndex.New(bleveIndex), imageIndex.New(bleveIndex), nodeComponentEdgeIndex.New(bleveIndex),
		nodeIndex.New(bleveIndex), deploymentIndex.New(bleveIndex, bleveIndex), clusterIndexer.New(bleveIndex))
	suite.datastore, _ = New(dacky, store, index, searcher, suite.mockRisk, ranking.ComponentRanker())
}

func (suite *ImageComponentDataStoreTestSuite) TearDownSuite() {
	_ = os.RemoveAll(suite.blevePath)
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageComponentDataStoreTestSuite) TestBasicSearch() {
	component := getTestImageComponent("id1", "name1", "ver1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Basic unscoped search.
	results, err := suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	// Upsert component.
	suite.NoError(suite.datastore.Upsert(ctx, component))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Basic unscoped search.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    component.GetId(),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})

	// Basic scoped search.
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	// Search components.
	components, err := suite.datastore.SearchRawImageComponents(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.NotNil(components)
	suite.Len(components, 1)
	suite.Equal(component, components[0])

	// Upsert new component.
	newComponent := getTestImageComponent("id2", "name2", "ver2")
	suite.NoError(suite.datastore.Upsert(ctx, newComponent))

	// Ensure the CVEs are indexed.
	indexingDone = concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Search multiple components.
	components, err = suite.datastore.SearchRawImageComponents(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(components, 2)

	// Search for just one node.
	components, err = suite.datastore.SearchRawImageComponents(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(components, 1)
	suite.Equal(component, components[0])

	suite.deleteTestImageComponents(ctx)

	// Ensure search does not find anything.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *ImageComponentDataStoreTestSuite) TestSearchByComponent() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))
	suite.upsertTestImageComponents(ctx)

	// Search by Component.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    "id1",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id1", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id2",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id3",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestImageComponents(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id1",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id2",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func getTestImageComponent(id, name, version string) *storage.ImageComponent {
	return &storage.ImageComponent{
		Id:        id,
		Name:      name,
		Version:   version,
		Source:    storage.SourceType_INFRASTRUCTURE,
		Priority:  1,
		RiskScore: 30,
	}
}

func (suite *ImageComponentDataStoreTestSuite) upsertTestImageComponents(ctx context.Context) {
	image := getTestImageComponent("id1", "name1", "ver1")

	// Upsert component.
	suite.NoError(suite.datastore.Upsert(ctx, image))

	// Upsert new component.
	newComponent := getTestImageComponent("id2", "name2", "ver2")
	suite.NoError(suite.datastore.Upsert(ctx, newComponent))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()
}

func (suite *ImageComponentDataStoreTestSuite) deleteTestImageComponents(ctx context.Context) {
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_IMAGE_COMPONENT).Return(nil)
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id2", storage.RiskSubjectType_IMAGE_COMPONENT).Return(nil)
	suite.NoError(suite.datastore.Delete(ctx, "id1", "id2"))

	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()
}
