// This file was originally generated with
// //go:generate cp ../../../../central/compliance/datastore/internal/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface for accessing stored compliance data
type Store interface {
	Walk(ctx context.Context, fn func(obj *storage.ComplianceRunMetadata) error) error
	UpsertMany(ctx context.Context, objs []*storage.ComplianceRunMetadata) error
}
