package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imagecomponent/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var componentLoaderType = reflect.TypeOf(storage.ImageComponent{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.ImageComponent{}), func() interface{} {
		return NewComponentLoader(datastore.Singleton())
	})
}

// NewComponentLoader creates a new loader for component data.
func NewComponentLoader(ds datastore.DataStore) ComponentLoader {
	return &componentLoaderImpl{
		loaded: make(map[string]*storage.ImageComponent),
		ds:     ds,
	}
}

// GetComponentLoader returns the ComponentLoader from the context if it exists.
func GetComponentLoader(ctx context.Context) (ComponentLoader, error) {
	loader, err := GetLoader(ctx, componentLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ComponentLoader), nil
}

// ComponentLoader loads component data, and stores already loaded components for other ops in the same context to use.
type ComponentLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ImageComponent, error)
	FromID(ctx context.Context, id string) (*storage.ImageComponent, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageComponent, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// componentLoaderImpl implements the ComponentDataLoader interface.
type componentLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ImageComponent

	ds datastore.DataStore
}

// FromIDs loads a set of components from a set of ids.
func (idl *componentLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ImageComponent, error) {
	components, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

// FromID loads an component from an ID.
func (idl *componentLoaderImpl) FromID(ctx context.Context, id string) (*storage.ImageComponent, error) {
	components, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return components[0], nil
}

// FromQuery loads a set of components that match a query.
func (idl *componentLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageComponent, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *componentLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	numResults, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(numResults), nil
}

func (idl *componentLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *componentLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.ImageComponent, error) {
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

func (idl *componentLoaderImpl) setAll(components []*storage.ImageComponent) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, component := range components {
		idl.loaded[component.GetId()] = component
	}
}

func (idl *componentLoaderImpl) readAll(ids []string) (components []*storage.ImageComponent, missing []int) {
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
