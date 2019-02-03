package store

import (
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(id string) (*storage.Namespace, bool, error)
	GetNamespaces() ([]*storage.Namespace, error)
	AddNamespace(*storage.Namespace) error
	UpdateNamespace(*storage.Namespace) error
	RemoveNamespace(id string) error

	Search(q *v1.Query) ([]search.Result, error)
}

// New returns a new DataStore instance using the provided store and indexer
func New(store store.Store, indexer index.Indexer) (DataStore, error) {
	ds := &datastoreImpl{
		store:   store,
		indexer: indexer,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

type datastoreImpl struct {
	store   store.Store
	indexer index.Indexer
}

func (b *datastoreImpl) buildIndex() error {
	namespaces, err := b.GetNamespaces()
	if err != nil {
		return err
	}
	return b.indexer.AddNamespaces(namespaces)
}

// GetNamespace returns namespace with given id.
func (b *datastoreImpl) GetNamespace(id string) (namespace *storage.Namespace, exists bool, err error) {
	return b.store.GetNamespace(id)
}

// GetNamespaces retrieves namespaces matching the request from bolt
func (b *datastoreImpl) GetNamespaces() ([]*storage.Namespace, error) {
	return b.store.GetNamespaces()
}

// AddNamespace adds a namespace to bolt
func (b *datastoreImpl) AddNamespace(namespace *storage.Namespace) error {
	if err := b.store.AddNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespace(namespace)
}

// UpdateNamespace updates a namespace to bolt
func (b *datastoreImpl) UpdateNamespace(namespace *storage.Namespace) error {
	if err := b.store.UpdateNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespace(namespace)
}

// RemoveNamespace removes a namespace.
func (b *datastoreImpl) RemoveNamespace(id string) error {
	if err := b.store.RemoveNamespace(id); err != nil {
		return err
	}
	return b.indexer.DeleteNamespace(id)
}

func (b *datastoreImpl) Search(q *v1.Query) ([]search.Result, error) {
	return b.indexer.Search(q)
}
