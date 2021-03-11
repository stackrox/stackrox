package store

import "github.com/stackrox/rox/generated/storage"

// Store is a store for watched images.
type Store interface {
	Upsert(obj *storage.WatchedImage) error
	Walk(fn func(obj *storage.WatchedImage) error) error
	Delete(name string) error
	Exists(name string) (bool, error)
}
