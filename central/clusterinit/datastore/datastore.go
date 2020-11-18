package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/generated/storage"
)

var (
	// ErrTokenIDCollision signals that a token failed to be added to the datastore due to a token ID collision.
	ErrTokenIDCollision = errors.New("token ID collision")
	// ErrTokenNotFound signals that a requested token could not be located in the datastore.
	ErrTokenNotFound = errors.New("token not found")
)

// DataStore interface for managing stored bootstrap tokens.
type DataStore interface {
	GetAll(ctx context.Context) ([]*storage.BootstrapTokenWithMeta, error)
	Get(ctx context.Context, tokenID string) (*storage.BootstrapTokenWithMeta, error)
	Add(ctx context.Context, tokenMeta *storage.BootstrapTokenWithMeta) error
	Delete(ctx context.Context, tokenID string) error
	SetActive(ctx context.Context, tokenID string, active bool) error
}
