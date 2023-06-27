// This file was originally generated with
// //go:generate cp central/group/datastore/internal/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store updates and utilizes groups, which are attribute to role mappings.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.Group, error)
	Upsert(ctx context.Context, group *storage.Group) error
	UpsertOldFormat(ctx context.Context, group *storage.Group) error
}
