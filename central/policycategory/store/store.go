package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for policy categories.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.PolicyCategory, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.PolicyCategory, []int, error)
	Upsert(ctx context.Context, obj *storage.PolicyCategory) error
	UpsertMany(ctx context.Context, objs []*storage.PolicyCategory) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	Walk(ctx context.Context, fn func(obj *storage.PolicyCategory) error) error
}
