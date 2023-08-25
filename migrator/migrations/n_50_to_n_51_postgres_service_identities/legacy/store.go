// This file was originally generated with
// //go:generate cp ../../../../central/serviceidentities/internal/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for service identities.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.ServiceIdentity, error)
	Upsert(ctx context.Context, obj *storage.ServiceIdentity) error
}
