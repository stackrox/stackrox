package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/clustercveedge/index"
	"github.com/stackrox/rox/central/clustercveedge/search"
	dackboxStore "github.com/stackrox/rox/central/clustercveedge/store/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
)

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, _ *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence dackboxConcurrency.KeyFence) (DataStore, error) {
	storage, err := dackboxStore.New(dacky, keyFence)
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	cveIndexer := cveIndex.New(bleveIndex)
	searcher := search.New(storage, indexer, cveIndexer, dacky)
	return New(dacky, storage, indexer, searcher)
}
