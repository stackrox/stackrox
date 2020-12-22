package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store implements the interface for process baseline results.
type Store interface {
	Delete(id string) error
	Get(id string) (*storage.ProcessBaselineResults, bool, error)
	Upsert(baselineresults *storage.ProcessBaselineResults) error
}
