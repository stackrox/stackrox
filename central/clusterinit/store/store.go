package store

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

var (
	// ErrInitBundleNotFound signals that a requested init bundle could not be located in the store.
	ErrInitBundleNotFound = errors.New("init bundle not found")

	// ErrInitBundleIDCollision signals that an init bundle could not be added to the store due to an ID collision.
	ErrInitBundleIDCollision = errors.New("init bundle ID collision")

	// ErrInitBundleDuplicateName signals that an init bundle could not be added because the name already exists on a non-revoked init bundle
	ErrInitBundleDuplicateName = errors.New("init bundle already exists")
)

// Store interface for managing persisted cluster init bundles.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	Get(ctx context.Context, id string) (*storage.InitBundleMeta, error)
	Add(ctx context.Context, bundleMeta *storage.InitBundleMeta) error
	Revoke(ctx context.Context, id string) error
}
