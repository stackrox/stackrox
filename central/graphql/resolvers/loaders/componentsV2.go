package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	componentV2LoaderType = reflect.TypeOf(storage.ImageComponentV2{})

	log = logging.LoggerForModule()
)

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.ImageComponentV2{}), func() interface{} {
		return NewComponentV2Loader(datastore.Singleton())
	})
}

// NewComponentV2Loader creates a new loader for component data.
func NewComponentV2Loader(ds datastore.DataStore) ComponentV2Loader {
	return &componentV2LoaderImpl{
		loaded: make(map[string]*storage.ImageComponentV2),
		ds:     ds,
	}
}

// GetComponentV2Loader returns the ComponentLoader from the context if it exists.
func GetComponentV2Loader(ctx context.Context) (ComponentV2Loader, error) {
	loader, err := GetLoader(ctx, componentV2LoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ComponentV2Loader), nil
}

// ComponentV2Loader loads component data, and stores already loaded components for other ops in the same context to use.
type ComponentV2Loader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ImageComponentV2, error)
	FromID(ctx context.Context, id string) (*storage.ImageComponentV2, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageComponentV2, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// componentV2LoaderImpl implements the ComponentDataLoader interface.
type componentV2LoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ImageComponentV2

	ds datastore.DataStore
}

// FromIDs loads a set of components from a set of ids.
func (idl *componentV2LoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ImageComponentV2, error) {
	components, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

// FromID loads an component from an ID.
func (idl *componentV2LoaderImpl) FromID(ctx context.Context, id string) (*storage.ImageComponentV2, error) {
	components, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return components[0], nil
}

// FromQuery loads a set of components that match a query.
func (idl *componentV2LoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageComponentV2, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *componentV2LoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	numResults, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(numResults), nil
}

func (idl *componentV2LoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *componentV2LoaderImpl) load(ctx context.Context, ids []string) ([]*storage.ImageComponentV2, error) {
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

func (idl *componentV2LoaderImpl) setAll(components []*storage.ImageComponentV2) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, component := range components {
		idl.loaded[component.GetId()] = component
	}
}

func (idl *componentV2LoaderImpl) readAll(ids []string) (components []*storage.ImageComponentV2, missing []int) {
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
