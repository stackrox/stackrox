package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for process whitelists.
//go:generate mockgen-wrapper
type Store interface {
	Get(id string) (*storage.ProcessWhitelist, bool, error)
	GetMany(ids []string) ([]*storage.ProcessWhitelist, []int, error)
	Walk(fn func(whitelist *storage.ProcessWhitelist) error) error

	Upsert(whitelist *storage.ProcessWhitelist) error

	Delete(id string) error
}
