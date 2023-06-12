package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var nodeLoaderType = reflect.TypeOf(storage.Node{})

func init() {
	RegisterTypeFactory(nodeLoaderType, func() interface{} {
		return NewNodeLoader(nodeDatastore.Singleton())
	})
}

// NewNodeLoader creates a new loader for node data.
func NewNodeLoader(ds nodeDatastore.DataStore) NodeLoader {
	return &nodeLoaderImpl{
		loaded: make(map[string]*storage.Node),
		ds:     ds,
	}
}

// GetNodeLoader returns the NodeLoader from the context if it exists.
func GetNodeLoader(ctx context.Context) (NodeLoader, error) {
	loader, err := GetLoader(ctx, nodeLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(NodeLoader), nil
}

// NodeLoader loads node data, and stores already loaded node for other ops in the same context to use.
type NodeLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.Node, error)
	FromID(ctx context.Context, id string) (*storage.Node, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Node, error)
	FullNodeWithID(ctx context.Context, id string) (*storage.Node, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// nodeLoaderImpl implements the NodeLoader interface.
type nodeLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.Node

	ds nodeDatastore.DataStore
}

// FromIDs loads a set of nodes from a set of ids.
func (ndl *nodeLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.Node, error) {
	nodes, err := ndl.load(ctx, ids, false)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// FromID loads a node from an ID. Does not load scan components and vulnerabilities when Postgres is enabled.
func (ndl *nodeLoaderImpl) FromID(ctx context.Context, id string) (*storage.Node, error) {
	nodes, err := ndl.load(ctx, []string{id}, false)
	if err != nil {
		return nil, err
	}
	return nodes[0], nil
}

// FullNodeWithID loads full node from an ID.
func (ndl *nodeLoaderImpl) FullNodeWithID(ctx context.Context, id string) (*storage.Node, error) {
	node, err := ndl.FromID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Do not load the full node if full scan is already available.
	if node.GetComponents() == 0 || len(node.GetScan().GetComponents()) > 0 {
		return node, nil
	}

	ndl.lock.Lock()
	delete(ndl.loaded, id)
	ndl.lock.Unlock()

	nodes, err := ndl.load(ctx, []string{id}, true)
	if err != nil {
		return nil, err
	}
	return nodes[0], nil
}

// FromQuery loads a set of nodes that match a query.
func (ndl *nodeLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Node, error) {
	results, err := ndl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return ndl.FromIDs(ctx, search.ResultsToIDs(results))
}

// CountFromQuery returns the number of nodes that match a given query.
func (ndl *nodeLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	numResults, err := ndl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(numResults), nil
}

// CountAll returns the total number of nodes.
func (ndl *nodeLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := ndl.ds.CountNodes(ctx)
	return int32(count), err
}

func (ndl *nodeLoaderImpl) load(ctx context.Context, ids []string, pullFullObject bool) ([]*storage.Node, error) {
	nodes, missing := ndl.readAll(ids)
	if len(missing) > 0 {
		var err error
		// `pullFullObject` is only supported on Postgres.
		if pullFullObject {
			nodes, err = ndl.ds.GetNodesBatch(ctx, collectMissing(ids, missing))
		} else {
			nodes, err = ndl.ds.GetManyNodeMetadata(ctx, collectMissing(ids, missing))
		}
		if err != nil {
			return nil, err
		}
		ndl.setAll(nodes)
		nodes, missing = ndl.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all nodes could be found: %s", strings.Join(missingIDs, ","))
	}
	return nodes, nil
}

func (ndl *nodeLoaderImpl) setAll(nodes []*storage.Node) {
	ndl.lock.Lock()
	defer ndl.lock.Unlock()

	for _, node := range nodes {
		ndl.loaded[node.GetId()] = node
	}
}

func (ndl *nodeLoaderImpl) readAll(ids []string) (nodes []*storage.Node, missing []int) {
	ndl.lock.RLock()
	defer ndl.lock.RUnlock()

	for idx, id := range ids {
		node, isLoaded := ndl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			nodes = append(nodes, node)
		}
	}
	return
}
