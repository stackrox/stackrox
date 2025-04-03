package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store encapsulates the role binding store
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.K8SRoleBinding, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.K8SRoleBinding, []int, error)
	Walk(ctx context.Context, fn func(binding *storage.K8SRoleBinding) error) error
	Upsert(ctx context.Context, rolebinding *storage.K8SRoleBinding) error
	Delete(ctx context.Context, id ...string) error
}
