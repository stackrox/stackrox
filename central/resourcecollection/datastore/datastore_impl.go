package datastore

import (
	"context"

	"github.com/heimdalr/dag"
	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log           = logging.LoggerForModule()
	initBatchSize = 20
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	graphLock sync.Mutex
	graph     *dag.DAG
}

func resetLocalGraph(ds *datastoreImpl) {

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	if ds.graph != nil {
		ds.graph = nil
	}
}

func (ds *datastoreImpl) initGraph(ctx context.Context) error {
	if ds.graph != nil {
		return nil
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	ds.graph = dag.NewDAG()

	// add ids first
	ids, err := ds.storage.GetIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		err = ds.graph.AddVertexByID(id, id)
		if err != nil {
			return err
		}
	}

	// then add edges by batches
	for i := 0; i < len(ids); i += initBatchSize {
		var objs []*storage.ResourceCollection
		if i+initBatchSize < len(ids) {
			objs, _, err = ds.storage.GetMany(ctx, ids[i:i+initBatchSize])
		} else {
			objs, _, err = ds.storage.GetMany(ctx, ids[i:])
		}
		if err != nil {
			return err
		}
		for _, obj := range objs {
			for _, edge := range obj.GetEmbeddedCollections() {
				err = ds.graph.AddEdge(obj.GetId(), edge.GetId())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ds *datastoreImpl) addCollectionToGraph(ctx context.Context, obj *storage.ResourceCollection) error {
	if err := ds.initGraph(ctx); err != nil {
		return err
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	err := ds.graph.AddVertexByID(obj.GetId(), obj.GetId())
	if err != nil {
		return err
	}
	for _, edge := range obj.GetEmbeddedCollections() {
		err = ds.graph.AddEdge(obj.GetId(), edge.GetId())
		if err != nil {
			deleteErr := ds.graph.DeleteVertex(obj.GetId())
			if deleteErr != nil {
				log.Errorf("Failed to delete collection, might result in bad state (%v)", deleteErr)
			}
			return err
		}
	}
	return nil
}

func (ds *datastoreImpl) deleteCollectionFromGraph(ctx context.Context, id string) error {
	if err := ds.initGraph(ctx); err != nil {
		return err
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	return ds.graph.DeleteVertex(id)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	return ds.searcher.SearchCollections(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchResults(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error) {
	collection, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return collection, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ResourceCollection, error) {
	collections, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func (ds *datastoreImpl) AddCollection(ctx context.Context, collection *storage.ResourceCollection) error {

	// add to graph first to detect any cycles
	err := ds.addCollectionToGraph(ctx, collection)
	if err != nil {
		return err
	}

	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		// if we fail to upsert, update the graph
		deleteErr := ds.deleteCollectionFromGraph(ctx, collection.GetId())
		if deleteErr != nil {
			log.Errorf("Failed to remove collection from internal state object (%v)", deleteErr)
		}
		return err
	}
	return err
}

func (ds *datastoreImpl) DeleteCollection(ctx context.Context, id string) error {

	// delete from storage first so postgres can tell if the collection is referenced by another
	err := ds.storage.Delete(ctx, id)
	if err != nil {
		return err
	}
	deleteErr := ds.deleteCollectionFromGraph(ctx, id)
	if deleteErr != nil {
		log.Errorf("Failed to remove collection from internal state object (%v)", deleteErr)
	}
	return err
}
