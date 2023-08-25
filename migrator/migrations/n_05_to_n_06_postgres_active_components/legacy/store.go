// This file was originally generated with
// //go:generate  cp ../../../../central/activecomponent/datastore/internal/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for active component.
type Store interface {
	GetMany(ctx context.Context, ids []string) ([]*storage.ActiveComponent, []int, error)
	GetIDs(ctx context.Context) ([]string, error)
	UpsertMany(ctx context.Context, activeComponents []*storage.ActiveComponent) error
}
