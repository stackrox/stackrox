package store

import "github.com/stackrox/rox/generated/api/v1"

// Store implements a store of all nodes in a cluster.
type Store interface {
	ListNodes() ([]*v1.Node, error)
	GetNode(id string) (*v1.Node, error)
	CountNodes() (int, error)

	UpsertNode(node *v1.Node) error
	RemoveNode(id string) error
}
