package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/nodecomponent/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var nodeComponentLoaderType = reflect.TypeOf(storage.NodeComponent{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.NodeComponent{}), func() interface{} {
		return NewNodeComponentLoader(datastore.Singleton())
	})
}

// NewNodeComponentLoader creates a new loader for node component data.
func NewNodeComponentLoader(ds datastore.DataStore) NodeComponentLoader {
	return &nodeComponentLoaderImpl{
		loaded: make(map[string]*storage.NodeComponent),
		ds:     ds,
	}
}

// GetNodeComponentLoader returns the NodeComponentLoader from the context if it exists.
func GetNodeComponentLoader(ctx context.Context) (NodeComponentLoader, error) {
	loader, err := GetLoader(ctx, nodeComponentLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(NodeComponentLoader), nil
}

// NodeComponentLoader loads node component data, and stores already loaded node components for other ops in the same context to use.
type NodeComponentLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.NodeComponent, error)
	FromID(ctx context.Context, id string) (*storage.NodeComponent, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.NodeComponent, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// nodeComponentLoaderImpl implements the NodeComponentLoader interface.
type nodeComponentLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.NodeComponent

	ds datastore.DataStore
}

func (idl *nodeComponentLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.NodeComponent, error) {
	components, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

func (idl *nodeComponentLoaderImpl) FromID(ctx context.Context, id string) (*storage.NodeComponent, error) {
	components, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return components[0], nil
}

func (idl *nodeComponentLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.NodeComponent, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *nodeComponentLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	numResults, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(numResults), nil
}

func (idl *nodeComponentLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *nodeComponentLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.NodeComponent, error) {
	components, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		components, err = idl.ds.GetBatch(ctx, collectMissing(ids, missing))
		if err != nil {
			return nil, err
		}
		idl.setAll(components)
		components, missing = idl.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all components could be found: %s", strings.Join(missingIDs, ","))
	}
	return components, nil
}

func (idl *nodeComponentLoaderImpl) setAll(components []*storage.NodeComponent) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, component := range components {
		idl.loaded[component.GetId()] = component
	}
}

func (idl *nodeComponentLoaderImpl) readAll(ids []string) (components []*storage.NodeComponent, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		component, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			components = append(components, component)
		}
	}
	return
}
