package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/node/datastore/search"
	"github.com/stackrox/rox/central/node/datastore/store"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	typ = "Node"
)

var (
	log = logging.LoggerForModule()

	nodesSAC = sac.ForResource(resources.Node)
)

type datastoreImpl struct {
	keyedMutex *concurrency.KeyedMutex

	storage  store.Store
	searcher search.Searcher

	risks riskDS.DataStore

	nodeRanker          *ranking.Ranker
	nodeComponentRanker *ranking.Ranker
}

func newDatastoreImpl(storage store.Store, searcher search.Searcher, risks riskDS.DataStore,
	nodeRanker *ranking.Ranker, nodeComponentRanker *ranking.Ranker) *datastoreImpl {
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,

		risks: risks,

		nodeRanker:          nodeRanker,
		nodeComponentRanker: nodeComponentRanker,

		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	return ds
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "Search")

	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "Count")

	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchNodes(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "SearchNodes")

	return ds.searcher.SearchNodes(ctx, q)
}

// SearchRawNodes delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "SearchRawNodes")

	nodes, err := ds.searcher.SearchRawNodes(ctx, q)
	if err != nil {
		return nil, err
	}

	ds.updateNodePriority(nodes...)

	return nodes, nil
}

// CountNodes delegates to the underlying store.
func (ds *datastoreImpl) CountNodes(ctx context.Context) (int, error) {
	if ok, err := nodesSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return ds.storage.Count(ctx)
	}

	return ds.Count(ctx, pkgSearch.EmptyQuery())
}

func (ds *datastoreImpl) canReadNode(ctx context.Context, id string) (bool, error) {
	if ok, err := nodesSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	queryForNode := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.NodeID, id).ProtoQuery()
	if results, err := ds.searcher.Search(ctx, queryForNode); err != nil {
		return false, err
	} else if len(results) > 0 {
		return true, nil
	}

	return false, nil
}

// GetNode delegates to the underlying store.
func (ds *datastoreImpl) GetNode(ctx context.Context, id string) (*storage.Node, bool, error) {
	if ok, err := ds.canReadNode(ctx, id); err != nil || !ok {
		return nil, false, err
	}

	node, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	ds.updateNodePriority(node)

	return node, true, nil
}

// GetNodesBatch delegates to the underlying store.
func (ds *datastoreImpl) GetNodesBatch(ctx context.Context, ids []string) ([]*storage.Node, error) {
	var nodes []*storage.Node
	var err error
	var ok bool

	if ok, err = nodesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	}

	if ok {
		nodes, _, err = ds.storage.GetMany(ctx, ids)
		if err != nil {
			return nil, err
		}
	} else {
		idsQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.NodeID, ids...).ProtoQuery()
		nodes, err = ds.SearchRawNodes(ctx, idsQuery)
		if err != nil {
			return nil, err
		}
	}

	ds.updateNodePriority(nodes...)

	return nodes, nil
}

// GetManyNodeMetadata gets the node data without the scan.
func (ds *datastoreImpl) GetManyNodeMetadata(ctx context.Context, ids []string) ([]*storage.Node, error) {
	nodes, missingIdx, err := ds.storage.GetManyNodeMetadata(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(missingIdx) > 0 {
		log.Errorf("Could not fetch %d/%d nodes", len(missingIdx), len(ids))
	}
	ds.updateNodePriority(nodes...)
	return nodes, nil
}

// UpsertNode dedupes the node with the underlying storage and adds the node to the index.
func (ds *datastoreImpl) UpsertNode(ctx context.Context, node *storage.Node) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "UpsertNode")

	if node.GetId() == "" {
		return errors.New("cannot upsert a node without an id")
	}

	if ok, err := nodesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(node.GetId())
	defer ds.keyedMutex.Unlock(node.GetId())

	ds.updateComponentRisk(node)
	enricher.FillScanStats(node)

	if err := ds.storage.Upsert(ctx, node); err != nil {
		return err
	}
	// If the node in db is latest, this node object will be carrying its risk score
	ds.nodeRanker.Add(node.GetId(), node.GetRiskScore())
	return nil
}

func (ds *datastoreImpl) DeleteNodes(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "DeleteNodes")

	if ok, err := nodesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.deleteNodeFromStore(ctx, ids...)
}

func (ds *datastoreImpl) DeleteAllNodesForCluster(ctx context.Context, clusterID string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "DeleteAllNodesForCluster")

	if ok, err := nodesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	results, err := ds.searcher.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, clusterID).ProtoQuery())
	if err != nil {
		return err
	}
	return ds.deleteNodeFromStore(ctx, pkgSearch.ResultsToIDs(results)...)
}

func (ds *datastoreImpl) deleteNodeFromStore(ctx context.Context, ids ...string) error {
	errorList := errorhelpers.NewErrorList("deleting nodes")
	deleteRiskCtx := sac.WithAllAccess(context.Background())
	for _, id := range ids {
		if err := ds.storage.Delete(ctx, id); err != nil {
			errorList.AddError(err)
			continue
		}
		if err := ds.risks.RemoveRisk(deleteRiskCtx, id, storage.RiskSubjectType_NODE); err != nil {
			errorList.AddError(err)
			continue
		}
	}
	return errorList.ToError()
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), typ, "Exists")
	return ds.storage.Exists(ctx, id)
}

func (ds *datastoreImpl) initializeRankers() {
	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Node)))

	results, err := ds.searcher.Search(readCtx, pkgSearch.EmptyQuery())
	if err != nil {
		log.Errorf("initializing node rankers: %v", err)
		return
	}

	for _, id := range pkgSearch.ResultsToIDs(results) {
		node, found, err := ds.storage.GetNodeMetadata(readCtx, id)
		if err != nil {
			log.Errorf("retrieving node for ranker initialization: %v", err)
			continue
		} else if !found {
			continue
		}

		ds.nodeRanker.Add(id, node.GetRiskScore())
	}
}

func (ds *datastoreImpl) updateNodePriority(nodes ...*storage.Node) {
	for _, node := range nodes {
		node.Priority = ds.nodeRanker.GetRankForID(node.GetId())
		for _, component := range node.GetScan().GetComponents() {
			component.Priority = ds.nodeComponentRanker.GetRankForID(scancomponent.ComponentID(component.GetName(), component.GetVersion(), node.GetScan().GetOperatingSystem()))
		}
	}
}

func (ds *datastoreImpl) updateComponentRisk(node *storage.Node) {
	for _, component := range node.GetScan().GetComponents() {
		component.RiskScore = ds.nodeComponentRanker.GetScoreForID(scancomponent.ComponentID(component.GetName(), component.GetVersion(), node.GetScan().GetOperatingSystem()))
	}
}
