package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for Image Components.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.ImageComponent, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ImageComponent, []int, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.ImageComponent) error) error

	Walk(ctx context.Context, fn func(obj *storage.ImageComponent) error, useClones bool) error

	Exists(ctx context.Context, id string) (bool, error)
}
