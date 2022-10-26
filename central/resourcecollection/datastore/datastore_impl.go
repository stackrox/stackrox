package datastore

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
	"github.com/stackrox/rox/pkg/set"
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
	names     set.Set[string]
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

			// track names
			if ds.names.Contains(obj.GetName()) {
				return fmt.Errorf("encountered duplicate name building collection graph (%s)", obj.GetName())
			}
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

func (ds *datastoreImpl) addCollectionToGraph(obj *storage.ResourceCollection) error {
	ds.initGraphOnce()

	var err error

	// input validation
	if err = verifyCollectionObjectNotEmpty(obj); err != nil {
		return err
	}
	if obj.GetId() != "" {
		return errors.New("invalid argument, added collection must not have a pre-set `id`")
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	obj.Id, err = ds.graph.AddVertex(graphEntry{
		id: uuid.New().String(),
	})
	if err != nil {
		return err
	}

	for _, child := range obj.GetEmbeddedCollections() {
		err = ds.graph.AddEdge(obj.GetId(), child.GetId())
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
	parentID           string
	addedChildrenIDs   []string
	removedChildrenIDs []string
}

func (ds *datastoreImpl) updateCollectionInGraph(obj *storage.ResourceCollection) (*graphEdgeUpdates, error) {
	return ds.updateCollectionInGraphWorkflow(obj, false)
}

func (ds *datastoreImpl) dryRunUpdateCollectionInGraph(obj *storage.ResourceCollection) error {
	_, err := ds.updateCollectionInGraphWorkflow(obj, true)
	return err
}

func (ds *datastoreImpl) updateCollectionInGraphWorkflow(obj *storage.ResourceCollection, dryrun bool) (*graphEdgeUpdates, error) {
	ds.initGraphOnce()

	var err error

	if err = verifyCollectionObjectNotEmpty(obj); err != nil {
		return nil, err
	}

	// return object struct
	ret := &graphEdgeUpdates{
		obj.GetId(),
		make([]string, 0),
		make([]string, 0),
	}

	// if we're doing a dryrun we don't need to lock since copy is threadsafe, and we won't touch the graph again
	if !dryrun {
		ds.graphLock.Lock()
		defer ds.graphLock.Unlock()
	}

	// create graph copy to use for operation
	graphCopy, err := ds.graph.Copy()
	if err != nil {
		return nil, err
	}

	// get current children edges for the object
	curChildren, err := graphCopy.GetChildren(obj.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "update attempt on unknown collection")
	}

	// determine additions
	for _, newChildren := range obj.GetEmbeddedCollections() {
		_, present := curChildren[newChildren.GetId()]
		if present {
			// if present in both we remove from curChildren
			delete(curChildren, newChildren.GetId())
		} else {
			ret.addedChildrenIDs = append(ret.addedChildrenIDs, newChildren.GetId())
		}
	}

	// determine deletions, any remaining edge in curChildren should be removed
	for key := range curChildren {
		ret.removedChildrenIDs = append(ret.removedChildrenIDs, key)
	}

	// since adding can cause cycles but removing never can, we remove first
	for _, childID := range ret.removedChildrenIDs {
		err = graphCopy.DeleteEdge(obj.GetId(), childID)
		if err != nil {
			return nil, err
		}
	}

	// add new edges
	for _, childID := range ret.addedChildrenIDs {
		err = graphCopy.AddEdge(obj.GetId(), childID)
		if err != nil {
			return nil, err
		}
	}

	// if this is a dryrun we just return and don't need to pass the operation object
	if dryrun {
		return nil, nil
	}

	// operation succeeded, update graph
	ds.graph = graphCopy

	return ret, nil
}

func (ds *datastoreImpl) undoEdgeUpdatesInGraph(updates *graphEdgeUpdates) error {
	if updates == nil {
		return errors.New("passed object must be non nil")
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	graphCopy, err := ds.graph.Copy()
	if err != nil {
		return err
	}

	// deletions will never cause cycles so do those first
	for _, addedChildId := range updates.addedChildrenIDs {
		err = graphCopy.DeleteEdge(updates.parentID, addedChildId)
		if err != nil {
			return errors.Wrap(err, "failed to remove added edge")
		}
	}

	// add back removed edges
	for _, removedChildID := range updates.removedChildrenIDs {
		err = graphCopy.AddEdge(updates.parentID, removedChildID)
		if err != nil {
			return errors.Wrap(err, "failed to restore removed edge")
		}
	}

	// update the graph
	ds.graph = graphCopy

	return nil
}

func (ds *datastoreImpl) deleteCollectionFromGraph(id string) error {
	ds.initGraphOnce()

	// this function covers removal of the vertex and any edges to or from other vertices, it is also threadsafe
	return ds.graph.DeleteVertex(id)
}

func (ds *datastoreImpl) dryRunAddCollectionInGraph(obj *storage.ResourceCollection) error {
	ds.initGraphOnce()

	var err error
	dryrunID := "dryrun"

	if err = verifyCollectionObjectNotEmpty(obj); err != nil {
		return err
	}

	if obj.GetId() != "" {
		return errors.New("invalid argument, added collection must not have a preset `id`")
	}

	ds.graphLock.Lock()
	defer ds.graphLock.Unlock()

	// we essentially just add the obj to the graph and delete it before returning
	dryrunID, err = ds.graph.AddVertex(graphEntry{
		id: dryrunID,
	})
	if err != nil {
		return err
	}
	for _, peer := range obj.GetEmbeddedCollections() {
		err = ds.graph.AddEdge(dryrunID, peer.GetId())
		if err != nil {
			break
		}
	}

	deleteErr := ds.graph.DeleteVertex(dryrunID)
	if err != nil {
		return err
	}
	return deleteErr
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
	if collection.GetId() != "" {
		return fmt.Errorf("added collections must not have an `id` preset (%s)", collection.GetId())
	}
	if collection.GetName() == "" || ds.names.Contains(collection.GetName()) {
		return fmt.Errorf("added collections must have non-empty, unique `name` values (%s)", collection.GetName())
	}
	ds.names.Add(collection.GetName())

	// add to graph first to detect any cycles, this also sets the `id` field
	err := ds.addCollectionToGraph(collection)
	if err != nil {
		ds.names.Remove(collection.GetName())
		return err
	}

	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		// if we fail to upsert, remove the name and update the graph
		ds.names.Remove(collection.GetName())
		deleteErr := ds.deleteCollectionFromGraph(collection.GetId())
		if deleteErr != nil {
			log.Errorf("Failed to remove collection from internal state object (%v)", deleteErr)
		}
		return err
	}
	return nil
}

func (ds *datastoreImpl) DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	if err := verifyCollectionObjectNotEmpty(collection); err != nil {
		return err
	}
	if collection.GetId() != "" {
		return fmt.Errorf("added collections must not have an `id` preset (%s)", collection.GetId())
	}
	if collection.GetName() == "" || ds.names.Contains(collection.GetName()) {
		return fmt.Errorf("added collections must have non-empty, unique `name` values (%s)", collection.GetName())
	}

	// check for access since dryrun flow doesn't actually hit the postgres layer
	if ok, err := workflowSAC.WriteAllowed(ctx); err != nil || !ok {
		return err
	}

	return ds.dryRunAddCollectionInGraph(collection)
}

func (ds *datastoreImpl) UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	if err := verifyCollectionObjectNotEmpty(collection); err != nil {
		return err
	}

	// resolve object to check if the name was changed
	obj, ok, err := ds.storage.Get(ctx, collection.GetId())
	if err != nil || !ok {
		return errors.Wrap(err, "failed to resolve collection being updated")
	}
	if obj.GetName() != collection.GetName() && ds.names.Contains(collection.GetName()) {
		return fmt.Errorf("collection name already in use (%s)", collection.GetName())
	}

	// update graph first to detect cycles
	edgeUpdates, err := ds.updateCollectionInGraph(collection)
	if err != nil {
		return err
	}

	// update datastore
	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		// if we fail to upsert, try to restore the graph
		err = ds.undoEdgeUpdatesInGraph(edgeUpdates)
		if err != nil {
			return errors.Wrap(err, "failed to restore state from invalid update operation")
		}
	}

	// update the tracked names if necessary
	if obj.GetName() != collection.GetName() {
		ds.names.Remove(obj.GetName())
		ds.names.Add(collection.GetName())
	}

	return err
}

func (ds *datastoreImpl) DeleteCollection(ctx context.Context, id string) error {

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
