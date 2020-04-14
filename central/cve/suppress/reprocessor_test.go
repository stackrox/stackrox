package suppress

import (
	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/types"
	clusterIndexer "github.com/stackrox/rox/central/cluster/index"
	clusterCVEEdgeIndexer "github.com/stackrox/rox/central/clustercveedge/index"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	cveSearch "github.com/stackrox/rox/central/cve/search"
	cveStore "github.com/stackrox/rox/central/cve/store/dackbox"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgDackBox "github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
	"time"
)

func TestUnsuppressCVEs(t *testing.T) {
	expiredCVEs := []*storage.CVE{
		{
			Id:             "cve1",
			Suppressed:     true,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() - int64(3*24*time.Hour)},
		},
		{
			Id:             "cve2",
			Suppressed:     true,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() - int64(2*24*time.Hour)},
		},
		{
			Id:             "cve3",
			Suppressed:     true,
			SuppressExpiry: &types.Timestamp{Seconds: time.Now().Unix() - int64(24*time.Hour)},
		},
		{
			Id:             "cve4",
			Suppressed:     false,
			SuppressExpiry: &types.Timestamp{},
		},
	}

	later := types.TimestampNow().Seconds + int64(time.Hour)
	unexpiredCVEs := []*storage.CVE{
		{
			Id:             "cve5",
			Suppressed:     true,
			SuppressExpiry: &types.Timestamp{Seconds: later},
		},
	}

	db := testutils.BadgerDBForT(t)
	defer utils.IgnoreError(db.Close)
	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	dacky, reg, indexQ := testDackBoxInstance(t, db, bleveIndex)
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})

	ds := createDataStore(t, dacky, bleveIndex)

	err = ds.Upsert(reprocessorCtx, append(expiredCVEs, unexpiredCVEs...)...)
	require.NoError(t, err)

	// ensure the cves are indexed
	indexingDone := concurrency.NewSignal()
	indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	loop := NewLoop(ds).(*cveUnsuppressLoopImpl)
	loop.unsuppressCVEsWithExpiredSuppressState()

	for _, cve := range expiredCVEs {
		actual, _, err := ds.Get(reprocessorCtx, cve.Id)
		assert.NoError(t, err)
		assert.False(t, actual.Suppressed)
	}

	for _, cve := range unexpiredCVEs {
		actual, _, err := ds.Get(reprocessorCtx, cve.Id)
		assert.NoError(t, err)
		assert.True(t, actual.Suppressed)
	}
}

func createDataStore(t *testing.T, dacky *pkgDackBox.DackBox, bleveIndex bleve.Index) cveDataStore.DataStore {
	store, err := cveStore.New(dacky, concurrency.NewKeyFence())
	utils.Must(err)

	cveIndexer := cveIndex.New(bleveIndex)
	searcher := cveSearch.New(store, dacky, cveIndexer,
		clusterCVEEdgeIndexer.New(bleveIndex),
		componentCVEEdgeIndexer.New(bleveIndex),
		componentIndexer.New(bleveIndex),
		imageComponentEdgeIndexer.New(bleveIndex),
		imageIndexer.New(bleveIndex),
		deploymentIndexer.New(bleveIndex),
		clusterIndexer.New(bleveIndex))

	ds, err := cveDataStore.New(store, cveIndexer, searcher)
	require.NoError(t, err)
	return ds
}

func testDackBoxInstance(t *testing.T, db *badger.DB, index bleve.Index) (*pkgDackBox.DackBox, indexer.WrapperRegistry, queue.WaitableQueue) {
	indexingQ := queue.NewWaitableQueue()
	dacky, err := pkgDackBox.NewDackBox(db, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	reg := indexer.NewWrapperRegistry()
	lazy := indexer.NewLazy(indexingQ, reg, index, dacky.AckIndexed)
	lazy.Start()

	return dacky, reg, indexingQ
}
