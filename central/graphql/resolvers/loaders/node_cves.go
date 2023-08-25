package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var nodeCveLoaderType = reflect.TypeOf(storage.NodeCVE{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.NodeCVE{}), func() interface{} {
		return NewNodeCVELoader(nodeCVEDataStore.Singleton())
	})
}

// NewNodeCVELoader creates a new loader for nodeCVE data.
func NewNodeCVELoader(ds nodeCVEDataStore.DataStore) NodeCVELoader {
	return &nodeCVELoaderImpl{
		loaded: make(map[string]*storage.NodeCVE),
		ds:     ds,
	}
}

// GetNodeCVELoader returns the NodeCVELoader from the context if it exists.
func GetNodeCVELoader(ctx context.Context) (NodeCVELoader, error) {
	loader, err := GetLoader(ctx, nodeCveLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(NodeCVELoader), nil
}

// NodeCVELoader loads nodeCVE data, and stores already loaded nodeCVEs for other ops in the same context to use.
type NodeCVELoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.NodeCVE, error)
	FromID(ctx context.Context, id string) (*storage.NodeCVE, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.NodeCVE, error)
	GetIDs(ctx context.Context, query *v1.Query) ([]string, error)
	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// nodeCVELoaderImpl implements the NodeCVELoader interface.
type nodeCVELoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.NodeCVE

	ds nodeCVEDataStore.DataStore
}

// FromIDs loads a set of nodeCVEs from a set of ids.
func (idl *nodeCVELoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.NodeCVE, error) {
	nodeCves, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return nodeCves, nil
}

// FromID loads a nodeCVE from an ID.
func (idl *nodeCVELoaderImpl) FromID(ctx context.Context, id string) (*storage.NodeCVE, error) {
	nodeCves, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return nodeCves[0], nil
}

// FromQuery loads a set of nodeCVEs that match a query.
func (idl *nodeCVELoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.NodeCVE, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *nodeCVELoaderImpl) GetIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}

func (idl *nodeCVELoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (idl *nodeCVELoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *nodeCVELoaderImpl) load(ctx context.Context, ids []string) ([]*storage.NodeCVE, error) {
	nodeCves, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		nodeCves, err = idl.ds.GetBatch(ctx, collectMissing(ids, missing))
		if err != nil {
			return nil, err
		}
		idl.setAll(nodeCves)
		nodeCves, missing = idl.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all node cves could be found: %s", strings.Join(missingIDs, ","))
	}
	return nodeCves, nil
}

func (idl *nodeCVELoaderImpl) setAll(nodeCves []*storage.NodeCVE) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, cve := range nodeCves {
		idl.loaded[cve.GetId()] = cve
	}
}

func (idl *nodeCVELoaderImpl) readAll(ids []string) (nodeCves []*storage.NodeCVE, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		cve, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			nodeCves = append(nodeCves, cve)
		}
	}
	return
}
