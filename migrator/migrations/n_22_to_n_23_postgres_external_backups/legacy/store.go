// This file was originally generated with
// //go:generate cp ../../../../central/externalbackups/internal/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store implements a store of all external backups in a cluster.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.ExternalBackup, error)
	Upsert(ctx context.Context, backup *storage.ExternalBackup) error
}
