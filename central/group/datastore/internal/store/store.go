package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store updates and utilizes groups, which are attribute to role mappings.
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, propsId string) (*storage.Group, bool, error)
	GetAll(ctx context.Context) ([]*storage.Group, error)
	Walk(ctx context.Context, fn func(group *storage.Group) error) error
	Upsert(ctx context.Context, group *storage.Group) error
	UpsertMany(ctx context.Context, groups []*storage.Group) error
	Delete(ctx context.Context, propsId string) error
	DeleteMany(ctx context.Context, ids []string) error
}
