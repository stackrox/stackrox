package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the role binding store
type Store interface {
	Get(ctx context.Context, id string) (*storage.K8SRoleBinding, bool, error)
	GetByQuery(ctx context.Context, q *v1.Query) ([]*storage.K8SRoleBinding, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.K8SRoleBinding, []int, error)
	Walk(ctx context.Context, fn func(binding *storage.K8SRoleBinding) error) error
	Upsert(ctx context.Context, rolebinding *storage.K8SRoleBinding) error
	Delete(ctx context.Context, id string) error
}
