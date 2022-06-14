package store

import (
	"context"

	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for CVEs.
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.CVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.CVE, []int, error)

	Exists(ctx context.Context, id string) (bool, error)

	Upsert(ctx context.Context, cves ...*storage.CVE) error
	Delete(ctx context.Context, ids ...string) error
}
