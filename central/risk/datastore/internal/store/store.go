package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store defines the interface for Risk storage
//go:generate mockgen-wrapper Store
type Store interface {
	Get(id string) (*storage.Risk, bool, error)
	GetMany(ids []string) ([]*storage.Risk, []int, error)
	Walk(func(risk *storage.Risk) error) error
	Upsert(risk *storage.Risk) error
	Delete(id string) error
}
