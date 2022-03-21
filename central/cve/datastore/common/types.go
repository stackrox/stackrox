package common

import (
	"context"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to node CVE storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	Get(ctx context.Context, id string) (*storage.CVE, bool, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.CVE, error)

	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, ids ...string) error
	Unsuppress(ctx context.Context, ids ...string) error

	Delete(ctx context.Context, ids ...string) error
}

// CVESuppressionCache holds suppressed vulnerabilities' information.
type CVESuppressionCache map[string]SuppressionCacheEntry

// SuppressionCacheEntry represents cache entry for suppressed resources.
type SuppressionCacheEntry struct {
	Suppressed         bool
	SuppressActivation *types.Timestamp
	SuppressExpiry     *types.Timestamp
}
