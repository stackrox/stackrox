package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for Image Components.
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.ImageComponent, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageComponent, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ImageComponent, []int, error)

	Exists(ctx context.Context, id string) (bool, error)
}
