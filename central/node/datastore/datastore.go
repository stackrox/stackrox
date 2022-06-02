package datastore

import (
	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper

// DataStore is a wrapper around a store that provides search functionality
type DataStore interface {
	ListNodes() ([]*storage.Node, error)
	GetNode(id string) (*storage.Node, error)
	CountNodes() (int, error)

	UpsertNode(node *storage.Node) error
	RemoveNode(id string) error
}
