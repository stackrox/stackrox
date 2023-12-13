package datastore

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/heimdalr/dag"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	graphInitBatchSize = 200
	resourceType       = "Collection"
)

var (
	workflowSAC = sac.ForResource(resources.WorkflowAdministration)
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher

	lock  sync.RWMutex
	graph *dag.DAG
	names set.Set[string]
}

// graphEntry is the stored object in the dag.DAG
type graphEntry struct {
	id string
}

// ID returns the id of the object
func (ge graphEntry) ID() string {
	return ge.id
}

// initGraph initializes the dag.DAG for a given datastore instance, should be invoked during init time
func (ds *datastoreImpl) initGraph() error {

	// build graph object
	graph := dag.NewDAG()

	ctx := sac.WithAllAccess(context.Background())

	// get ids first
	ids, err := ds.storage.GetIDs(ctx)
	if err != nil {
		return errors.Wrap(err, "building collection graph")
	}

	// add vertices by batches
	for i := 0; i < len(ids); i += graphInitBatchSize {
		var objs []*storage.ResourceCollection
		if i+graphInitBatchSize < len(ids) {
			objs, _, err = ds.storage.GetMany(ctx, ids[i:i+graphInitBatchSize])
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
	for i := 0; i < len(ids); i += graphInitBatchSize {
		var parents []*storage.ResourceCollection
		if i+graphInitBatchSize < len(ids) {
			parents, _, err = ds.storage.GetMany(ctx, ids[i:i+graphInitBatchSize])
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

// addCollectionToGraphNoLock creates a copy of the existing DAG and returns that copy with the collection added, or an appropriate error
func (ds *datastoreImpl) addCollectionToGraphNoLock(obj *storage.ResourceCollection) (*dag.DAG, error) {
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

// updateCollectionInGraphNoLock creates a copy of the existing DAG and returns that copy with the collection updated, or an appropriate error
func (ds *datastoreImpl) updateCollectionInGraphNoLock(obj *storage.ResourceCollection) (*dag.DAG, error) {
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

// deleteCollectionFromGraphNoLock removes the collection from the DAG, or returns an appropriate error
func (ds *datastoreImpl) deleteCollectionFromGraphNoLock(id string) error {
	// this function covers removal of the vertex and any edges to or from other vertices, it is also threadsafe
	return ds.graph.DeleteVertex(id)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Search")
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Count")
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "SearchCollections")
	return ds.searcher.SearchCollections(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "SearchResults")
	return ds.searcher.SearchResults(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Get")
	collection, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return collection, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Exists")
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetMany(ctx context.Context, ids []string) ([]*storage.ResourceCollection, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "GetMany")
	collections, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func (ds *datastoreImpl) addCollectionWorkflow(ctx context.Context, collection *storage.ResourceCollection, dryrun bool) (string, error) {

	// sanity checks
	if err := verifyCollectionConstraints(collection); err != nil {
		return "", err
	}
	if collection.GetId() != "" {
		return "", errors.New("new collections must not have a preset `id`")
	}

	// check for access so we can fast fail before locking
	if ok, err := workflowSAC.WriteAllowed(ctx); err != nil || !ok {
		return "", err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// verify that the name is not already in use
	if collection.GetName() == "" || ds.names.Contains(collection.GetName()) {
		return "", errors.Errorf("collections must have non-empty, unique `name` values (%s)", collection.GetName())
	}

	// add to graph to detect any cycles, this also sets the `id` field
	graph, err := ds.addCollectionToGraphNoLock(collection)
	if err != nil {
		return "", err
	}

	// if this is a dryrun, we don't want to add to storage or make changes to objects
	if dryrun {
		collection.Id = ""
		return "", nil
	}

	// add to storage
	err = ds.storage.Upsert(ctx, collection)
	if err != nil {
		return "", err
	}

	// we've succeeded, now set all the values
	ds.names.Add(collection.GetName())
	ds.graph = graph
	return collection.GetId(), nil
}

func (ds *datastoreImpl) AddCollection(ctx context.Context, collection *storage.ResourceCollection) (string, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "AddCollection")
	return ds.addCollectionWorkflow(ctx, collection, false)
}

func (ds *datastoreImpl) DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DryRunAddCollection")
	_, err := ds.addCollectionWorkflow(ctx, collection, true)
	return err
}

func (ds *datastoreImpl) updateCollectionWorkflow(ctx context.Context, collection *storage.ResourceCollection, dryrun bool) error {

	// sanity checks
	if err := verifyCollectionConstraints(collection); err != nil {
		return err
	}
	if collection.GetId() == "" {
		return errors.New("update must be called on an existing collection")
	}

	// check for access so we can fast fail before locking
	if ok, err := workflowSAC.WriteAllowed(ctx); err != nil || !ok {
		return err
	}

	// if this a dryrun we don't ever end up calling upsert, so we only need to get a read lock
	if dryrun {
		ds.lock.RLock()
		defer ds.lock.RUnlock()
	} else {
		ds.lock.Lock()
		defer ds.lock.Unlock()
	}

	// resolve object to check if the name was changed
	storedCollection, ok, err := ds.storage.Get(ctx, collection.GetId())
	if err != nil || !ok {
		return errors.Wrap(err, "failed to resolve collection being updated")
	}
	if storedCollection.GetName() != collection.GetName() && ds.names.Contains(collection.GetName()) {
		return errors.Errorf("collection name already in use (%s)", collection.GetName())
	}

	// update graph first to detect cycles
	graph, err := ds.updateCollectionInGraphNoLock(collection)
	if err != nil {
		return err
	}

	collection.CreatedBy = storedCollection.GetCreatedBy()
	collection.CreatedAt = storedCollection.GetCreatedAt()

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
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "UpdateCollection")
	return ds.updateCollectionWorkflow(ctx, collection, false)
}

func (ds *datastoreImpl) DryRunUpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DryRunUpdateCollection")
	return ds.updateCollectionWorkflow(ctx, collection, true)
}

func (ds *datastoreImpl) DeleteCollection(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DeleteCollection")

	// check for access so we can fast fail before locking
	if ok, err := workflowSAC.WriteAllowed(ctx); err != nil || !ok {
		return err
	}

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
		if strings.Contains(err.Error(), "SQLSTATE 23503") {
			err = errox.ReferencedByAnotherObject
		}
		return errors.Wrap(err, "failed to delete collection")
	}

	// update tracking collections
	ds.names.Remove(obj.GetName())
	err = ds.deleteCollectionFromGraphNoLock(id)
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

func (ds *datastoreImpl) ResolveCollectionQuery(ctx context.Context, collection *storage.ResourceCollection) (*v1.Query, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "ResolveCollectionQuery")
	var collectionQueue []*storage.ResourceCollection
	var visitedCollection set.Set[string]
	var disjunctions []*v1.Query

	if err := verifyCollectionConstraints(collection); err != nil {
		return nil, err
	}

	collectionQueue = append(collectionQueue, collection)

	ds.lock.RLock()
	defer ds.lock.RUnlock()

	for len(collectionQueue) > 0 {

		// get first index and remove from list
		collection := collectionQueue[0]
		collectionQueue = collectionQueue[1:]

		if !visitedCollection.Add(collection.GetId()) {
			continue
		}

		// resolve the collection to queries
		queries, err := collectionToQueries(collection)
		if err != nil {
			return nil, err
		}
		disjunctions = append(disjunctions, queries...)

		// add embedded values
		embeddedList, _, err := ds.storage.GetMany(ctx, embeddedCollectionsToIDList(collection.GetEmbeddedCollections()))
		if err != nil {
			return nil, err
		}
		collectionQueue = append(collectionQueue, embeddedList...)
	}

	return pkgSearch.DisjunctionQuery(disjunctions...), nil
}

// collectionToQueries returns a list of queries derived from the given resource collection's storage.ResourceSelector list
// these should be combined as disjunct with any resolved embedded queries
func collectionToQueries(collection *storage.ResourceCollection) ([]*v1.Query, error) {
	var ret []*v1.Query

	for _, resourceSelector := range collection.GetResourceSelectors() {
		var selectorRuleQueries []*v1.Query
		for _, selectorRule := range resourceSelector.GetRules() {

			fieldLabel, present := supportedFieldNames[selectorRule.GetFieldName()]
			if !present {
				return nil, errors.Wrapf(errox.InvalidArgs, "unsupported field name %q", selectorRule.GetFieldName())
			}

			ruleValueQueries, err := ruleValuesToQueryList(fieldLabel, selectorRule.GetValues())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse collection to query")
			}
			if len(ruleValueQueries) > 0 {
				// rule values are conjunct or disjunct with each other depending on the operator
				switch selectorRule.GetOperator() {
				case storage.BooleanOperator_OR:
					selectorRuleQueries = append(selectorRuleQueries, pkgSearch.DisjunctionQuery(ruleValueQueries...))
				case storage.BooleanOperator_AND:
					selectorRuleQueries = append(selectorRuleQueries, pkgSearch.ConjunctionQuery(ruleValueQueries...))
				default:
					return nil, errors.Wrap(errox.InvalidArgs, "unsupported boolean operator")
				}
			}
		}
		if len(selectorRuleQueries) > 0 {
			// selector rules are conjunct with each other
			ret = append(ret, pkgSearch.ConjunctionQuery(selectorRuleQueries...))
		}
	}

	return ret, nil
}

func ruleValuesToQueryList(fieldLabel supportedFieldKey, ruleValues []*storage.RuleValue) ([]*v1.Query, error) {
	ret := make([]*v1.Query, 0, len(ruleValues))
	for _, ruleValue := range ruleValues {
		var query *v1.Query
		if fieldLabel.labelType {
			switch ruleValue.GetMatchType() {
			case storage.MatchType_EXACT:
				key, value := stringutils.Split2(ruleValue.GetValue(), "=")
				query = pkgSearch.NewQueryBuilder().AddMapQuery(fieldLabel.fieldLabel, fmt.Sprintf("%q", key), fmt.Sprintf("%q", value)).ProtoQuery()
			default:
				return nil, errors.Wrap(errox.InvalidArgs, "label rules should only use exact mating")
			}
		} else {
			switch ruleValue.GetMatchType() {
			case storage.MatchType_EXACT:
				query = pkgSearch.NewQueryBuilder().AddExactMatches(fieldLabel.fieldLabel, ruleValue.GetValue()).ProtoQuery()
			case storage.MatchType_REGEX:
				query = pkgSearch.NewQueryBuilder().AddRegexes(fieldLabel.fieldLabel, ruleValue.GetValue()).ProtoQuery()
			default:
				return nil, errors.Wrapf(errox.InvalidArgs, "unknown match type encountered %q", ruleValue.GetMatchType())
			}
		}
		ret = append(ret, query)
	}
	return ret, nil
}

func embeddedCollectionsToIDList(embeddedList []*storage.ResourceCollection_EmbeddedResourceCollection) []string {
	ret := make([]string, 0, len(embeddedList))
	for _, embedded := range embeddedList {
		ret = append(ret, embedded.GetId())
	}
	return ret
}

// verifyCollectionConstraints ensures the given collection is valid according to implementation constraints
//   - the collection object is not nil
//   - there is at most one storage.ResourceSelector
//   - only storage.BooleanOperator_OR is supported as an operator
//   - all storage.SelectorRule "FieldName" values are valid
//   - all storage.RuleValue fields compile as valid regex
//   - storage.RuleValue fields supplied when "FieldName" values provided
//   - storage.MatchType is EXACT when storage.SelectorRule is a label type
func verifyCollectionConstraints(collection *storage.ResourceCollection) error {

	// object not nil
	err := verifyCollectionObjectNotEmpty(collection)
	if err != nil {
		return err
	}

	// currently we only support one resource selector per collection from UX
	if collection.GetResourceSelectors() != nil && len(collection.GetResourceSelectors()) > 1 {
		return errors.Wrap(errox.InvalidArgs, "only 1 resource selector is supported per collection")
	}

	for _, resourceSelector := range collection.GetResourceSelectors() {
		for _, selectorRule := range resourceSelector.GetRules() {

			// currently we only support disjunction (OR) operations
			if selectorRule.GetOperator() != storage.BooleanOperator_OR {
				return errors.Wrapf(errox.InvalidArgs, "%q boolean operator unsupported", selectorRule.GetOperator().String())
			}

			// we have a short list of supported field name values
			labelVal, present := supportedFieldNames[selectorRule.GetFieldName()]
			if !present {
				return errors.Wrapf(errox.InvalidArgs, "unsupported field name %q", selectorRule.GetFieldName())
			}

			// we require at least one value if a field name is set
			if len(selectorRule.GetValues()) == 0 {
				return errors.Wrap(errox.InvalidArgs, "rule values required with a set field name")
			}
			for _, ruleValue := range selectorRule.GetValues() {

				// rule values must be valid regex
				if ruleValue.GetMatchType() == storage.MatchType_REGEX {
					_, err := regexp.Compile(ruleValue.GetValue())
					if err != nil {
						return errors.Wrapf(err, "failed to compile regex on %q rule", selectorRule.GetFieldName())
					}
				}

				// label rules only support exact matching and should be of the form 'key=value'
				if labelVal.labelType {
					if ruleValue.GetMatchType() != storage.MatchType_EXACT {
						return errors.Wrap(errox.InvalidArgs, "label types should only use exact matching")
					}
					if -1 == strings.IndexRune(ruleValue.GetValue(), '=') {
						return errors.Wrap(errox.InvalidArgs, "label values should be of the form 'key=value'")
					}
				}
			}
		}
	}

	return nil
}
