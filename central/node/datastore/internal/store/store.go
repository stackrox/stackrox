package store

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for nodes.
//go:generate mockgen-wrapper
type Store interface {
	GetNodes() ([]*storage.Node, error)
	CountNodes() (int, error)
	GetNode(id string) (*storage.Node, bool, error)
	// GetNodeMetadata gets the node without scan/component data.
	GetNodeMetadata(id string) (*storage.Node, bool, error)
	GetNodesBatch(ids []string) ([]*storage.Node, []int, error)

	Exists(id string) (bool, error)

	Upsert(node *storage.Node) error
	Delete(id string) error
}
