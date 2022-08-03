package globaldatastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/node/datastore"
	dackboxDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()

	nodesSAC = sac.ForResource(resources.Node)
)

type globalDataStore struct {
	datastore dackboxDatastore.DataStore
}

// New creates and returns a new GlobalDataStore.
func New(datastore dackboxDatastore.DataStore) (*globalDataStore, error) {
	return &globalDataStore{
		datastore: datastore,
	}, nil
}

func (s *globalDataStore) GetAllClusterNodeStores(ctx context.Context, writeAccess bool) (map[string]datastore.DataStore, error) {
	accessMode := storage.Access_READ_ACCESS
	if writeAccess {
		accessMode = storage.Access_READ_WRITE_ACCESS
	}

	results, err := s.datastore.Search(ctx, search.EmptyQuery())
	if err != nil {
		return nil, err
	}
	nodeIDs := search.ResultsToIDSet(results)

	clusterIDs := set.NewStringSet()
	for nodeID := range nodeIDs {
		node, exists, err := s.datastore.GetNode(ctx, nodeID)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		clusterIDs.Add(node.GetClusterId())
	}

	if ok, err := nodesSAC.AccessAllowed(ctx, accessMode); err != nil {
		return nil, err
	} else if !ok {
		scopeChecker := nodesSAC.ScopeChecker(ctx, accessMode)
		// Pass 1: Mark requests for all clusters as pending
		for clusterID := range clusterIDs {
			scopeChecker.TryAllowed(sac.ClusterScopeKey(clusterID))
		}
		if err := scopeChecker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		// Pass 2: Filter out clusters for which we have no access.
		for clusterID := range clusterIDs {
			if scopeChecker.TryAllowed(sac.ClusterScopeKey(clusterID)) != sac.Allow {
				clusterIDs.Remove(clusterID)
			}
		}
	}

	dataStores := make(map[string]datastore.DataStore, clusterIDs.Cardinality())
	for clusterID := range clusterIDs {
		dataStores[clusterID] = newDatastoreShim(clusterID, s.datastore)
	}
	return dataStores, nil
}

func (s *globalDataStore) GetClusterNodeStore(ctx context.Context, clusterID string, writeAccess bool) (datastore.DataStore, error) {
	accessMode := storage.Access_READ_ACCESS
	if writeAccess {
		accessMode = storage.Access_READ_WRITE_ACCESS
	}

	if ok, err := nodesSAC.AccessAllowed(ctx, accessMode, sac.ClusterScopeKey(clusterID)); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	return newDatastoreShim(clusterID, s.datastore), nil
}

func (s *globalDataStore) RemoveClusterNodeStores(ctx context.Context, clusterIDs ...string) error {
	// Stop early, otherwise, the clusterID query turns into an empty query that matches all
	if len(clusterIDs) == 0 {
		return nil
	}

	if !features.PostgresDatastore.Enabled() {
		if ok, err := nodesSAC.WriteAllowed(ctx); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterIDs...).ProtoQuery()
	results, err := s.datastore.Search(ctx, q)
	if err != nil {
		return errors.Wrap(err, "searching for nodes")
	}

	nodeIDs := search.ResultsToIDs(results)
	if err := s.datastore.DeleteNodes(ctx, nodeIDs...); err != nil {
		return errors.Wrap(err, "deleting nodes from storage")
	}
	return nil
}

func (s *globalDataStore) CountAllNodes(ctx context.Context) (int, error) {
	return s.datastore.CountNodes(ctx)
}

// SearchResults returns any node matches to the query
func (s *globalDataStore) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	searchResults := make([]*v1.SearchResult, 0, len(results))

	for _, r := range results {
		node, exists, err := s.datastore.GetNode(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		searchResults = append(searchResults, &v1.SearchResult{
			Id:             r.ID,
			Name:           node.Name,
			Category:       v1.SearchCategory_NODES,
			FieldToMatches: search.GetProtoMatchesMap(r.Matches),
			Score:          r.Score,
			Location:       fmt.Sprintf("%s/%s", node.GetClusterName(), node.GetName()),
		})
	}
	return searchResults, nil
}

// SearchRawNodes returns nodes that match a query
func (s *globalDataStore) SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	nodes := make([]*storage.Node, 0, len(results))
	for _, r := range results {
		node, exists, err := s.datastore.GetNode(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		if exists {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// Search returns any node matches to the query
func (s *globalDataStore) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.datastore.Search(ctx, q)
}

// Count returns the number of nodes matches the query
func (s *globalDataStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.datastore.Count(ctx, q)
}
