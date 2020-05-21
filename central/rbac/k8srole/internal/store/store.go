package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the k8srole store
type Store interface {
	Get(id string) (*storage.K8SRole, bool, error)
	GetMany(ids []string) ([]*storage.K8SRole, []int, error)
	Walk(fn func(role *storage.K8SRole) error) error

	Upsert(role *storage.K8SRole) error
	Delete(id string) error
}
