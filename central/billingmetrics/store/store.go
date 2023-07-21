package store

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// Store stores and retrieves values from the storage.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) ([]storage.BillingMetrics, error)
	Insert(ctx context.Context, rec *storage.BillingMetrics) error
}
