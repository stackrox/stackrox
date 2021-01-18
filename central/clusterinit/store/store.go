package store

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

var (
	// ErrInitBundleNotFound signals that a requested init bundle could not be located in the store.
	ErrInitBundleNotFound = errors.New("init bundle not found")

	// ErrInitBundleIDCollision signals that an init bundle could not be added to the store due to an ID collision.
	ErrInitBundleIDCollision = errors.New("init bundle ID collision")
)

// Store interface for managing persisted cluster init bundles.
type Store interface {
	GetAll() ([]*storage.InitBundleMeta, error)
	Get(id string) (*storage.InitBundleMeta, error)
	Add(bundleMeta *storage.InitBundleMeta) error
}
