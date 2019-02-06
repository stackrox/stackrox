package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// DataStore is a wrapper around a store that provides search functionality
type DataStore interface {
	store.Store
}

// New returns a new datastore
func New(store store.Store, indexer index.Indexer) DataStore {
	return &datastoreImpl{
		store:      store,
		indexer:    indexer,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}

type datastoreImpl struct {
	indexer    index.Indexer
	store      store.Store
	keyedMutex *concurrency.KeyedMutex
}

// ListNodes returns all nodes in the store
func (d *datastoreImpl) ListNodes() ([]*storage.Node, error) {
	return d.store.ListNodes()
}

// GetNode returns an individual node
func (d *datastoreImpl) GetNode(id string) (*storage.Node, error) {
	return d.store.GetNode(id)
}

// CountNodes returns the number of nodes
func (d *datastoreImpl) CountNodes() (int, error) {
	return d.store.CountNodes()
}

// UpsertNode adds a node to the store and the indexer
func (d *datastoreImpl) UpsertNode(node *storage.Node) error {
	d.keyedMutex.Lock(node.GetId())
	defer d.keyedMutex.Unlock(node.GetId())
	if err := d.store.UpsertNode(node); err != nil {
		return err
	}
	return d.indexer.AddNode(node)
}

// RemoveNode deletes a node from the store and the indexer
func (d *datastoreImpl) RemoveNode(id string) error {
	d.keyedMutex.Lock(id)
	defer d.keyedMutex.Unlock(id)
	if err := d.store.RemoveNode(id); err != nil {
		return err
	}
	return d.indexer.DeleteNode(id)
}
