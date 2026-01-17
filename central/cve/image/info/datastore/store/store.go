package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for ImageCVEInfos.
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.ImageCVEInfo, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ImageCVEInfo, []int, error)
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(*storage.ImageCVEInfo) error) error

	Upsert(ctx context.Context, time *storage.ImageCVEInfo) error
	UpsertMany(ctx context.Context, times []*storage.ImageCVEInfo) error
}
