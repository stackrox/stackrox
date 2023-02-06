package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NodeCVE, []int, error)

	UpsertMany(ctx context.Context, cves []*storage.NodeCVE) error
}
