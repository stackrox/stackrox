package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/namespace/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var namespaceLoaderType = reflect.TypeOf(storage.NamespaceMetadata{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.NamespaceMetadata{}), func() interface{} {
		return NewNamespaceLoader(datastore.Singleton())
	})
}

// NewNamespaceLoader creates a new loader for NamespaceMetaData.
func NewNamespaceLoader(ds datastore.DataStore) NamespaceLoader {
	return &namespaceLoaderImpl{
		loaded: make(map[string]*storage.NamespaceMetadata),
		ds:     ds,
	}
}

// GetNamespaceLoader returns the NamespaceLoader from the context if it exists.
func GetNamespaceLoader(ctx context.Context) (NamespaceLoader, error) {
	loader, err := GetLoader(ctx, namespaceLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(NamespaceLoader), nil
}

// NamespaceLoader loads namespace metadata, and stores already loaded metadata for other ops in the same context to use.
type NamespaceLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.NamespaceMetadata, error)
	FromID(ctx context.Context, id string) (*storage.NamespaceMetadata, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.NamespaceMetadata, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// namespaceLoaderImpl implements the NamespaceLoader interface.
type namespaceLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.NamespaceMetadata

	ds datastore.DataStore
}

// FromIDs loads a set of namespaces from a set of ids.
func (nsldr *namespaceLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.NamespaceMetadata, error) {
	namespaces, err := nsldr.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

// FromID loads a namespace from an ID.
func (nsldr *namespaceLoaderImpl) FromID(ctx context.Context, id string) (*storage.NamespaceMetadata, error) {
	namespaces, err := nsldr.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return namespaces[0], nil
}

// FromQuery loads a set of namespaces that match a query.
func (nsldr *namespaceLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.NamespaceMetadata, error) {
	results, err := nsldr.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return nsldr.FromIDs(ctx, search.ResultsToIDs(results))
}

// CountFromQuery counts the number of namespaces that match a query
func (nsldr *namespaceLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	numResults, err := nsldr.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(numResults), nil
}

// CountAll returns number of all namespaces
func (nsldr *namespaceLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := nsldr.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (nsldr *namespaceLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.NamespaceMetadata, error) {
	namespaces, missing := nsldr.readAll(ids)
	if len(missing) > 0 {
		var err error
		namespaces, err = nsldr.ds.GetManyNamespaces(ctx, collectMissing(ids, missing))
		if err != nil {
			return nil, err
		}
		nsldr.setAll(namespaces)
		namespaces, missing = nsldr.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all namespaces could be found: %s", strings.Join(missingIDs, ","))
	}
	return namespaces, nil
}

func (nsldr *namespaceLoaderImpl) setAll(namespaces []*storage.NamespaceMetadata) {
	nsldr.lock.Lock()
	defer nsldr.lock.Unlock()

	for _, namespace := range namespaces {
		nsldr.loaded[namespace.GetId()] = namespace
	}
}

func (nsldr *namespaceLoaderImpl) readAll(ids []string) (namespaces []*storage.NamespaceMetadata, missing []int) {
	nsldr.lock.RLock()
	defer nsldr.lock.RUnlock()

	for idx, id := range ids {
		namespace, isLoaded := nsldr.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			namespaces = append(namespaces, namespace)
		}
	}
	return
}
