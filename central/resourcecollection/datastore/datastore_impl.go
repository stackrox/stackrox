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
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	initBatchSize = 20
)

var (
	log         = logging.LoggerForModule()
	workflowSAC = sac.ForResource(resources.WorkflowAdministration)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	lock  sync.RWMutex
	graph *dag.DAG
	names set.Set[string]
}

type graphEntry struct {
	id string
}

func (ge graphEntry) ID() string {
	return ge.id
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

	// build graph object
	graph := dag.NewDAG()

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

			// track names, postgres guarantees uniqueness
			ds.names.Add(obj.GetName())

			// add object
			_, err = graph.AddVertex(graphEntry{
				id: obj.GetId(),
			})
			if err != nil {
				return errors.Wrap(err, "building collection graph")
			}
		}
	}

	// then add edges by batches
	for i := 0; i < len(ids); i += initBatchSize {
		var parents []*storage.ResourceCollection
		if i+initBatchSize < len(ids) {
			parents, _, err = ds.storage.GetMany(ctx, ids[i:i+initBatchSize])
		} else {
			parents, _, err = ds.storage.GetMany(ctx, ids[i:])
		}
		if err != nil {
			return errors.Wrap(err, "building collection graph")
		}
		for _, parent := range parents {
			for _, child := range parent.GetEmbeddedCollections() {
				err = graph.AddEdge(parent.GetId(), child.GetId())
				if err != nil {
					return errors.Wrap(err, "building collection graph")
				}
			}
		}
	}

	// set the graph object
	ds.graph = graph

	return nil
}

// addCollectionToGraph creates a copy of the existing DAG and returns that copy with the collection added, or an appropriate error
func (ds *datastoreImpl) addCollectionToGraph(obj *storage.ResourceCollection) (*dag.DAG, error) {
	ds.initGraphOnce()

	var err error

	// input validation
	if err = verifyCollectionObjectNotEmpty(obj); err != nil {
		return nil, err
	}
	if obj.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "new collection must not have a pre-set `id`")
	}

	// create graph copy to do this operation on
	graph, err := ds.graph.Copy()
	if err != nil {
		return nil, err
	}

	// add vertex
	id, err := graph.AddVertex(graphEntry{
		id: uuid.NewV4().String(),
	})
	if err != nil {
		return nil, err
	}

	// add edges
	for _, child := range obj.GetEmbeddedCollections() {
		err = graph.AddEdge(id, child.GetId())
		if err != nil {
			return nil, err
		}
	}

	// set id for the object and return the new graph
	obj.Id = id
	return graph, nil
}

// updateCollectionInGraph creates a copy of the existing DAG and returns that copy with the collection updated, or an appropriate error
func (ds *datastoreImpl) updateCollectionInGraph(obj *storage.ResourceCollection) (*dag.DAG, error) {
	ds.initGraphOnce()

	var err error

	if err = verifyCollectionObjectNotEmpty(obj); err != nil {
		return nil, err
	}

	addChildIDs := make([]string, 0)
	removeChildIDs := make([]string, 0)

	// create graph copy to use for operation
	graph, err := ds.graph.Copy()
	if err != nil {
		return nil, err
	}

	// get current children edges for the object
	curChildren, err := graph.GetChildren(obj.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "could not update collection (%s)", obj.GetId())
	}

	// determine additions
	for _, newChild := range obj.GetEmbeddedCollections() {
		_, present := curChildren[newChild.GetId()]
		if present {
			// if present in both we remove from curChildren
			delete(curChildren, newChild.GetId())
		} else {
			addChildIDs = append(addChildIDs, newChild.GetId())
		}
	}

	// determine deletions, any remaining edge in curChildren should be removed
	for key := range curChildren {
		removeChildIDs = append(removeChildIDs, key)
	}

	// since adding can cause cycles but removing never can, we remove first
	for _, childID := range removeChildIDs {
		err = graph.DeleteEdge(obj.GetId(), childID)
		if err != nil {
			return nil, err
		}
	}

	// add new edges
	for _, childID := range addChildIDs {
		err = graph.AddEdge(obj.GetId(), childID)
		if err != nil {
			return nil, err
		}
	}

	return graph, nil
}

// deleteCollectionFromGraph removes the collection from the DAG, or returns an appropriate error
func (ds *datastoreImpl) deleteCollectionFromGraph(id string) error {
	ds.initGraphOnce()

	// this function covers removal of the vertex and any edges to or from other vertices, it is also threadsafe
	return ds.graph.DeleteVertex(id)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.SearchCollections(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.SearchResults(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	collection, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return collection, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetMany(ctx context.Context, ids []string) ([]*storage.ResourceCollection, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	collections, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func (ds *datastoreImpl) addCollectionWorkflow(ctx context.Context, collection *storage.ResourceCollection, dryrun bool) error {

	// sanity checks
	if err := verifyCollectionObjectNotEmpty(collection); err != nil {
		return err
	}
	if collection.GetId() != "" {
		return errors.New("new collections must not have a preset `id`")
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// verify that the name is not already in use
	if collection.GetName() == "" || ds.names.Contains(collection.GetName()) {
		return errors.Errorf("collections must have non-empty, unique `name` values (%s)", collection.GetName())
	}

	// add to graph to detect any cycles, this also sets the `id` field
	graph, err := ds.addCollectionToGraph(collection)
	if err != nil {
		return err
	}

	// if this is a dryrun, we don't want to add to storage or make changes to objects
	if dryrun {
		collection.Id = ""
		return nil
	}

	// add to storage
	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		return err
	}

	// we've succeeded, now set all the values
	ds.names.Add(collection.GetName())
	ds.graph = graph
	return nil
}

func (ds *datastoreImpl) AddCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	return ds.addCollectionWorkflow(ctx, collection, false)
}

func (ds *datastoreImpl) DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error {

	// check for access since dryrun flow doesn't actually hit the postgres layer
	if ok, err := workflowSAC.WriteAllowed(ctx); err != nil || !ok {
		return err
	}

	return ds.addCollectionWorkflow(ctx, collection, true)
}

func (ds *datastoreImpl) updateCollectionWorkflow(ctx context.Context, collection *storage.ResourceCollection, dryrun bool) error {

	// sanity checks
	if err := verifyCollectionObjectNotEmpty(collection); err != nil {
		return err
	}
	if collection.GetId() == "" {
		return errors.New("update must be called on an existing collection")
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// resolve object to check if the name was changed
	storedCollection, ok, err := ds.storage.Get(ctx, collection.GetId())
	if err != nil || !ok {
		return errors.Wrap(err, "failed to resolve collection being updated")
	}
	if storedCollection.GetName() != collection.GetName() && ds.names.Contains(collection.GetName()) {
		return errors.Errorf("collection name already in use (%s)", collection.GetName())
	}

	// update graph first to detect cycles
	graph, err := ds.updateCollectionInGraph(collection)
	if err != nil {
		return err
	}

	// if this is a dryrun we don't want to make changes to the datastore or tracking objects
	if dryrun {
		return nil
	}

	// update datastore
	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		return err
	}

	// success, we now update objects
	if ds.names.Add(collection.GetName()) {
		ds.names.Remove(storedCollection.GetName())
	}
	ds.graph = graph
	return nil
}

func (ds *datastoreImpl) UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	return ds.updateCollectionWorkflow(ctx, collection, false)
}

func (ds *datastoreImpl) DeleteCollection(ctx context.Context, id string) error {

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// resolve object so we can get the name
	obj, ok, err := ds.storage.Get(ctx, id)
	if err != nil || !ok {
		return errors.Wrap(err, "failed to resolve collection for deletion")
	}

	// delete from storage first so postgres can tell if the collection is referenced by another
	err = ds.storage.Delete(ctx, id)
	if err != nil {
		return err
	}

	// update tracking collections
	ds.names.Remove(obj.GetName())
	err = ds.deleteCollectionFromGraph(id)
	if err != nil {
		return errors.Wrap(err, "failed to remove collection from internal state object after removing from datastore")
	}

	return err
}

func verifyCollectionObjectNotEmpty(obj *storage.ResourceCollection) error {
	if obj == nil {
		return errors.New("passed collection must be non nil")
	}
	return nil
}
