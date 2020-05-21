package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the role binding store
type Store interface {
	Get(id string) (*storage.K8SRoleBinding, bool, error)
	GetMany(ids []string) ([]*storage.K8SRoleBinding, []int, error)
	Walk(fn func(binding *storage.K8SRoleBinding) error) error
	Upsert(rolebinding *storage.K8SRoleBinding) error
	Delete(id string) error
}
