package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for pods.
//
//go:generate mockgen-wrapper
type Store interface {
	GetIDs(ctx context.Context) ([]string, error)

	Get(ctx context.Context, id string) (*storage.Pod, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Pod, []int, error)
	Walk(ctx context.Context, fn func(obj *storage.Pod) error) error

	Upsert(ctx context.Context, pod *storage.Pod) error
	Delete(ctx context.Context, id string) error
}
