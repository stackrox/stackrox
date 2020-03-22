package store

import (
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/storecache"
	bbolt "go.etcd.io/bbolt"
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
func New(db *bbolt.DB, cache storecache.Cache) (Store, error) {
	return newStore(db, cache)
}
