package store

import (
	bbolt "github.com/etcd-io/bbolt"
	storage "github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for process whitelists.
type Store interface {
	AddWhitelist(whitelist *storage.ProcessWhitelist) error
	DeleteWhitelist(id string) error
	GetWhitelist(id string) (*storage.ProcessWhitelist, error)
	GetWhitelists(ids []string) ([]*storage.ProcessWhitelist, []int, error)
	ListWhitelists() ([]*storage.ProcessWhitelist, error)
	UpdateWhitelist(whitelist *storage.ProcessWhitelist) error
	WalkAll(fn func(whitelist *storage.ProcessWhitelist) error) error
}

// New provides a new instance of Store.
func New(db *bbolt.DB) (Store, error) {
	return newStore(db)
}
