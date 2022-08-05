package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the k8srole store
type Store interface {
	Get(ctx context.Context, id string) (*storage.K8SRole, bool, error)
	GetByQuery(ctx context.Context, q *v1.Query) ([]*storage.K8SRole, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.K8SRole, []int, error)
	Walk(ctx context.Context, fn func(role *storage.K8SRole) error) error

	Upsert(ctx context.Context, role *storage.K8SRole) error
	Delete(ctx context.Context, id string) error
}
