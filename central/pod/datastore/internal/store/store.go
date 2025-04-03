package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for pods.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.Pod, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Pod, []int, error)
	Walk(ctx context.Context, fn func(obj *storage.Pod) error) error
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(pod *storage.Pod) error) error

	Upsert(ctx context.Context, pod *storage.Pod) error
	Delete(ctx context.Context, id ...string) error
}
