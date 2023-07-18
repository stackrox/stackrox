package store

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// Store stores and retrieves values from the storage.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, from, to time.Time) ([]storage.BillingMetricsRecord, error)
	Insert(ctx context.Context, rec *storage.BillingMetricsRecord) error
}
