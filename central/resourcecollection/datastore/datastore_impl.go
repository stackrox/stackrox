package datastore

import (
	"context"

	"github.com/heimdalr/dag"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log           = logging.LoggerForModule()
	initBatchSize = 20
	workflowSAC   = sac.ForResource(resources.WorkflowAdministration)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	graphLock sync.Mutex
	graph     *dag.DAG
}

func (ds *datastoreImpl) initGraphOnce() {
	once.Do(ds.initGraphCrashOnError)
}

func (ds *datastoreImpl) initGraphCrashOnError() {
	err := ds.initGraph()
	utils.CrashOnError(err)
}

func (ds *datastoreImpl) initGraph() error {
	if ds.graph != nil {
		return nil
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	ds.graph = dag.NewDAG()
	ctx := sac.WithAllAccess(context.Background())

	// get ids first
	ids, err := ds.storage.GetIDs(ctx)
	if err != nil {
		return errors.Wrap(err, "building collection graph")
	}

	// add vertices by batches
	for i := 0; i < len(ids); i += initBatchSize {
		var objs []*storage.ResourceCollection
		if i+initBatchSize < len(ids) {
			objs, _, err = ds.storage.GetMany(ctx, ids[i:i+initBatchSize])
		} else {
			objs, _, err = ds.storage.GetMany(ctx, ids[i:])
		}
		if err != nil {
			return errors.Wrap(err, "building collection graph")
		}
		for _, obj := range objs {
			err = ds.graph.AddVertexByID(obj.GetId(), obj.GetName())
			if err != nil {
				return errors.Wrap(err, "building collection graph")
			}
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
			return errors.Wrap(err, "building collection graph")
		}
		for _, obj := range objs {
			for _, peer := range obj.GetEmbeddedCollections() {
				err = ds.graph.AddEdge(obj.GetId(), peer.GetId())
				if err != nil {
					return errors.Wrap(err, "building collection graph")
				}
			}
		}
	}
	return nil
}

func (ds *datastoreImpl) addCollectionToGraph(obj *storage.ResourceCollection) error {
	ds.initGraphOnce()
	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	err := ds.graph.AddVertexByID(obj.GetId(), obj.GetName())
	if err != nil {
		return err
	}
	for _, peer := range obj.GetEmbeddedCollections() {
		err = ds.graph.AddEdge(obj.GetId(), peer.GetId())
		if err != nil {
			deleteErr := ds.graph.DeleteVertex(obj.GetId())
			if deleteErr != nil {
				log.Errorf("Failed to remove collection from internal state object (%v)", deleteErr)
			}
			return err
		}
	}
	return nil
}

func (ds *datastoreImpl) deleteCollectionFromGraph(id string) error {
	ds.initGraphOnce()
	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	return ds.graph.DeleteVertex(id)
}

func (ds *datastoreImpl) dryRunAddCollectionInGraph(obj *storage.ResourceCollection) error {
	ds.initGraphOnce()
	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	var ret error

	// we essentially just add the obj to the graph and delete it before returning
	err := ds.graph.AddVertexByID(obj.GetId(), obj.GetName())
	if err != nil {
		return err
	}
	for _, peer := range obj.GetEmbeddedCollections() {
		err = ds.graph.AddEdge(obj.GetId(), peer.GetId())
		if err != nil {
			ret = err
			break
		}
	}

	err = ds.graph.DeleteVertex(obj.GetId())
	if err != nil && ret == nil {
		ret = err
	}

	return ret
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
	err := ds.addCollectionToGraph(collection)
	if err != nil {
		return err
	}

	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		// if we fail to upsert, update the graph
		deleteErr := ds.deleteCollectionFromGraph(collection.GetId())
		if deleteErr != nil {
			log.Errorf("Failed to remove collection from internal state object (%v)", deleteErr)
		}
		return err
	}
	return err
}

func (ds *datastoreImpl) DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	// check for access since dryrun flow doesn't actually hit the postgres layer
	if ok, err := workflowSAC.WriteAllowed(ctx); err != nil || !ok {
		return err
	}

	return ds.dryRunAddCollectionInGraph(collection)
}

func (ds *datastoreImpl) DeleteCollection(ctx context.Context, id string) error {

	// delete from storage first so postgres can tell if the collection is referenced by another
	err := ds.storage.Delete(ctx, id)
	if err != nil {
		return err
	}
	deleteErr := ds.deleteCollectionFromGraph(id)
	if deleteErr != nil {
		log.Errorf("Failed to remove collection from internal state object (%v)", deleteErr)
	}
	return err
}
