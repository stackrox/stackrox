package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for policies.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.Policy, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Policy, []int, error)
	GetAll(ctx context.Context) ([]*storage.Policy, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, obj *storage.Policy) error
	UpsertMany(ctx context.Context, objs []*storage.Policy) error

	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
}
