package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store implements a store of all nodes in a cluster.
type Store interface {
	ListNodes() ([]*storage.Node, error)
	GetNode(id string) (*storage.Node, error)
	CountNodes() (int, error)

	UpsertNode(node *storage.Node) error
	RemoveNode(id string) error
}

//go:generate mockgen-wrapper Store
