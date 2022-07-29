// This file was originally generated with
// //go:generate cp ../../../../central/cve/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
type Store interface {
	GetMany(ctx context.Context, ids []string) ([]*storage.CVE, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, cves ...*storage.CVE) error
	UpsertMany(ctx context.Context, cves []*storage.CVE) error
}
