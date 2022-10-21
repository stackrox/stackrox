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

	if obj == nil {
		return errors.New("passed collection must not be nil")
	}

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

// graphEdgeUpdates stores added and removed edges from the DAG graph
type graphEdgeUpdates struct {
	srcId   string
	added   []string
	removed []string
}

func (ds *datastoreImpl) updateCollectionInGraph(obj *storage.ResourceCollection) (*graphEdgeUpdates, error) {
	ds.initGraphOnce()
	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	if obj == nil {
		return nil, errors.New("passed collection must not be nil")
	}

	// return object struct
	ret := &graphEdgeUpdates{
		obj.GetId(),
		make([]string, 0),
		make([]string, 0),
	}

	// get edges for the current object
	curEdges, err := ds.graph.GetDescendants(obj.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "update attempt on unknown collection")
	}

	for _, edge := range obj.GetEmbeddedCollections() {
		_, present := curEdges[edge.GetId()]
		if !present {
			ret.added = append(ret.added, edge.GetId())
		} else {
			// if we visit an edge, set the value to nil
			curEdges[edge.GetId()] = nil
		}
	}

	// if a value is nil we visited it, if it's not nil we want to remove that edge
	for key, value := range curEdges {
		if value != nil {
			ret.removed = append(ret.removed, key)
		}
	}

	// since adding can cause cycles but removing never can, we remove first
	// we iterate back to front so that if we fail a deletion we can remove the index from our returned list of deletions
	for idx := len(ret.removed) - 1; idx >= 0; idx-- {
		deleteErr := ds.graph.DeleteEdge(obj.GetId(), ret.removed[idx])
		if deleteErr != nil {
			ret.removed = append(ret.removed[:idx], ret.removed[idx+1:]...)
			log.Errorf("Failed to remove collection edge from internal state object (%v)", deleteErr)
		}
	}

	// add new edges
	for idx, addId := range ret.added {
		err = ds.graph.AddEdge(obj.GetId(), addId)
		if err != nil {

			// make note of where we stopped adding edges
			ret.added = ret.added[:idx]

			ds.undoEdgeUpdatesInGraph(ret)
			return nil, err
		}
	}
	return ret, nil
}

func (ds *datastoreImpl) undoEdgeUpdatesInGraphWithLock(updates *graphEdgeUpdates) {
	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	ds.undoEdgeUpdatesInGraph(updates)
}

func (ds *datastoreImpl) undoEdgeUpdatesInGraph(updates *graphEdgeUpdates) {

	// ensure lock is acquired before calling this function
	if ds.graphLock.TryLock() {
		log.Errorf("function called without lock")
		ds.graphLock.Unlock()
		return
	}

	if updates == nil {
		log.Infof("passed updates object was nil")
		return
	}

	// remove added edges first since removing can never cause cycles
	for _, added := range updates.added {
		deleteErr := ds.graph.DeleteEdge(updates.srcId, added)
		if deleteErr != nil {
			log.Infof("tried to delete an edge that didn't exist (%v)", deleteErr)
		}
	}

	// then restore edges that were removed
	for _, removed := range updates.removed {
		restoreErr := ds.graph.AddEdge(updates.srcId, removed)
		if restoreErr != nil {
			log.Errorf("Failed to restore collection edge in internal state object (%v)", restoreErr)
		}
	}
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

	if obj == nil {
		return errors.New("passed collection must not be nil")
	}

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
	if collection == nil {
		return errors.New("passed collection must not be nil")
	}

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

	if collection == nil {
		return errors.New("passed collection must not be nil")
	}

	return ds.dryRunAddCollectionInGraph(collection)
}

func (ds *datastoreImpl) UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	if collection == nil {
		return errors.New("passed collection must not be nil")
	}

	// update graph first to detect cycles
	edgeUpdates, err := ds.updateCollectionInGraph(collection)
	if err != nil {
		return err
	}

	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		// if we fail to upsert, try to restore the graph
		ds.undoEdgeUpdatesInGraphWithLock(edgeUpdates)
	}
	return err
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
