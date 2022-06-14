package store

import (
	"context"

	storage "github.com/stackrox/rox/generated/storage"
)

// Store implements the interface for process baseline results.
type Store interface {
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*storage.ProcessBaselineResults, bool, error)
	Upsert(ctx context.Context, baselineresults *storage.ProcessBaselineResults) error
}
