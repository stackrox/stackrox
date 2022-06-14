package suppress

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/gogo/protobuf/types"
	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	clusterCVEEdgeDataStore "github.com/stackrox/stackrox/central/clustercveedge/datastore"
	clusterCVEEdgeIndexer "github.com/stackrox/stackrox/central/clustercveedge/index"
	clusterCVEEdgeSearcher "github.com/stackrox/stackrox/central/clustercveedge/search"
	clusterCVEEdgeStore "github.com/stackrox/stackrox/central/clustercveedge/store/dackbox"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	"github.com/stackrox/stackrox/central/cve/converter"
	cveDackbox "github.com/stackrox/stackrox/central/cve/dackbox"
	cveDataStore "github.com/stackrox/stackrox/central/cve/datastore"
	cveIndex "github.com/stackrox/stackrox/central/cve/index"
	cveSearch "github.com/stackrox/stackrox/central/cve/search"
	cveStore "github.com/stackrox/stackrox/central/cve/store/dackbox"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globalindex"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	pkgDackBox "github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/indexer"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsuppressCVEs(t *testing.T) {
	expiredCVEs := []*storage.CVE{
		{
			Id:             "cve1",
			Suppressed:     true,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() - int64(3*24*time.Hour)},
		},
		{
			Id:             "cve2",
			Suppressed:     true,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() - int64(2*24*time.Hour)},
		},
		{
			Id:             "cve3",
			Suppressed:     true,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() - int64(24*time.Hour)},
		},
		{
			Id:             "cve4",
			Suppressed:     false,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{},
		},
	}

	later := types.TimestampNow().Seconds + int64(time.Hour)
	unexpiredCVEs := []*storage.CVE{
		{
			Id:             "cve5",
			Suppressed:     true,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{Seconds: later},
		},
		{
			Id:             "cve6",
			Suppressed:     true,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix()},
		},
		{
			Id:             "cve7",
			Suppressed:     true,
			Type:           storage.CVE_K8S_CVE,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() + int64(24*time.Hour)},
		},
	}

	db := rocksdbtest.RocksDBForT(t)
	defer db.Close()
	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	dacky, reg, indexQ := testDackBoxInstance(t, db, bleveIndex)
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})

	cveDataStore, edgeDataStore := createDataStore(t, dacky, indexQ, bleveIndex)

	cveClusters := []*storage.Cluster{{Id: "id"}}
	parts := make([]converter.ClusterCVEParts, 0, len(expiredCVEs)+len(unexpiredCVEs))
	for _, expiredCVE := range expiredCVEs {
		parts = append(parts, converter.NewClusterCVEParts(expiredCVE, cveClusters, "fixVersions"))
	}
	for _, unexpiredCVE := range unexpiredCVEs {
		parts = append(parts, converter.NewClusterCVEParts(unexpiredCVE, cveClusters, "fixVersions"))
	}
	err = edgeDataStore.Upsert(reprocessorCtx, parts...)
	require.NoError(t, err)

	// ensure the cves are indexed
	indexingDone := concurrency.NewSignal()
	indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	loop := NewLoop(cveDataStore).(*cveUnsuppressLoopImpl)
	loop.unsuppressCVEsWithExpiredSuppressState()

	for _, cve := range expiredCVEs {
		actual, _, err := cveDataStore.Get(reprocessorCtx, cve.Id)
		assert.NoError(t, err)
		assert.False(t, actual.Suppressed)
	}

	for _, cve := range unexpiredCVEs {
		actual, _, err := cveDataStore.Get(reprocessorCtx, cve.Id)
		assert.NoError(t, err)
		assert.True(t, actual.Suppressed)
	}

	newSig := concurrency.NewSignal()
	indexQ.PushSignal(&newSig)
	newSig.Wait()
}

func createDataStore(t *testing.T, dacky *pkgDackBox.DackBox, indexQ queue.WaitableQueue, bleveIndex bleve.Index) (cveDataStore.DataStore, clusterCVEEdgeDataStore.DataStore) {
	cveStorage := cveStore.New(dacky, concurrency.NewKeyFence())

	cveIndexer := cveIndex.New(bleveIndex)
	cveSearcher := cveSearch.New(cveStorage, dacky, cveIndexer,
		clusterCVEEdgeIndexer.New(bleveIndex),
		componentCVEEdgeIndexer.New(bleveIndex),
		componentIndexer.New(bleveIndex),
		imageComponentEdgeIndexer.New(bleveIndex),
		imageCVEEdgeIndexer.New(bleveIndex),
		imageIndexer.New(bleveIndex),
		nodeComponentEdgeIndexer.New(bleveIndex),
		nodeIndexer.New(bleveIndex),
		deploymentIndexer.New(bleveIndex, bleveIndex),
		clusterIndexer.New(bleveIndex))

	cveDataStore, err := cveDataStore.New(dacky, indexQ, cveStorage, cveIndexer, cveSearcher)
	require.NoError(t, err)

	edgeStorage, err := clusterCVEEdgeStore.New(dacky, concurrency.NewKeyFence())
	require.NoError(t, err)
	edgeIndexer := clusterCVEEdgeIndexer.New(bleveIndex)
	edgeSearcher := clusterCVEEdgeSearcher.New(edgeStorage, edgeIndexer, cveIndexer, dacky)
	edgeDataStore, err := clusterCVEEdgeDataStore.New(dacky, edgeStorage, edgeIndexer, edgeSearcher)
	require.NoError(t, err)
	return cveDataStore, edgeDataStore
}

func testDackBoxInstance(t *testing.T, db *rocksdb.RocksDB, index bleve.Index) (*pkgDackBox.DackBox, indexer.WrapperRegistry, queue.WaitableQueue) {
	indexingQ := queue.NewWaitableQueue()
	dacky, err := pkgDackBox.NewRocksDBDackBox(db, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	reg := indexer.NewWrapperRegistry()
	lazy := indexer.NewLazy(indexingQ, reg, index, dacky.AckIndexed)
	lazy.Start()

	return dacky, reg, indexingQ
}
