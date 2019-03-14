package datastore

import (
	"fmt"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper DataStore

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(id string) (*storage.NamespaceMetadata, bool, error)
	GetNamespaces() ([]*storage.NamespaceMetadata, error)
	AddNamespace(*storage.NamespaceMetadata) error
	UpdateNamespace(*storage.NamespaceMetadata) error
	RemoveNamespace(id string) error

	Search(q *v1.Query) ([]search.Result, error)
	SearchNamespaces(q *v1.Query) ([]*storage.NamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store and indexer
func New(store store.Store, indexer index.Indexer) (DataStore, error) {
	ds := &datastoreImpl{
		store:      store,
		indexer:    indexer,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

type datastoreImpl struct {
	store   store.Store
	indexer index.Indexer

	keyedMutex *concurrency.KeyedMutex
}

func (b *datastoreImpl) buildIndex() error {
	namespaces, err := b.GetNamespaces()
	if err != nil {
		return err
	}
	return b.indexer.AddNamespaces(namespaces)
}

// GetNamespace returns namespace with given id.
func (b *datastoreImpl) GetNamespace(id string) (namespace *storage.NamespaceMetadata, exists bool, err error) {
	return b.store.GetNamespace(id)
}

// GetNamespaces retrieves namespaces matching the request from bolt
func (b *datastoreImpl) GetNamespaces() ([]*storage.NamespaceMetadata, error) {
	return b.store.GetNamespaces()
}

// AddNamespace adds a namespace to bolt
func (b *datastoreImpl) AddNamespace(namespace *storage.NamespaceMetadata) error {
	b.keyedMutex.Lock(namespace.GetId())
	defer b.keyedMutex.Unlock(namespace.GetId())
	if err := b.store.AddNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespace(namespace)
}

// UpdateNamespace updates a namespace to bolt
func (b *datastoreImpl) UpdateNamespace(namespace *storage.NamespaceMetadata) error {
	b.keyedMutex.Lock(namespace.GetId())
	defer b.keyedMutex.Unlock(namespace.GetId())
	if err := b.store.UpdateNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespace(namespace)
}

// RemoveNamespace removes a namespace.
func (b *datastoreImpl) RemoveNamespace(id string) error {
	b.keyedMutex.Lock(id)
	defer b.keyedMutex.Unlock(id)
	if err := b.store.RemoveNamespace(id); err != nil {
		return err
	}
	return b.indexer.DeleteNamespace(id)
}

func (b *datastoreImpl) Search(q *v1.Query) ([]search.Result, error) {
	return b.indexer.Search(q)
}

func (b *datastoreImpl) SearchNamespaces(q *v1.Query) ([]*storage.NamespaceMetadata, error) {
	results, err := b.indexer.Search(q)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	nsSlice := make([]*storage.NamespaceMetadata, 0, len(results))
	for _, res := range results {
		ns, exists, err := b.GetNamespace(res.ID)
		if err != nil {
			return nil, fmt.Errorf("retrieving namespace %q: %v", res.ID, err)
		}
		if !exists {
			// This could be due to a race where it's deleted in the time between
			// the search and the query to Bolt.
			continue
		}
		nsSlice = append(nsSlice, ns)
	}
	return nsSlice, nil
}
