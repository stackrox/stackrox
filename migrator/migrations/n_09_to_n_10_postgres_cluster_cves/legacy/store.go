// This file was originally generated with
// //go:generate cp ../../../../central/cve/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.CVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ClusterCVE, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Exists(ctx context.Context, id string) (bool, error)

	Upsert(ctx context.Context, cves ...*storage.CVE) error
	UpsertMany(ctx context.Context, cves []*storage.ClusterCVE) error
	Delete(ctx context.Context, ids ...string) error
}
