package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for process baselines.
//go:generate mockgen-wrapper
type Store interface {
	Get(id string) (*storage.ProcessBaseline, bool, error)
	GetMany(ids []string) ([]*storage.ProcessBaseline, []int, error)
	Walk(fn func(baseline *storage.ProcessBaseline) error) error

	Upsert(baseline *storage.ProcessBaseline) error

	Delete(id string) error
}
