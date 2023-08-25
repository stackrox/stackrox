// This file was originally generated with
// //go:generate cp ../../../../central/notifier/datastore/internal/store/store.go  .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for notifies
type Store interface {
	GetAll(ctx context.Context) ([]*storage.Notifier, error)
	Upsert(ctx context.Context, obj *storage.Notifier) error
}
