// This file was originally generated with
// //go:generate cp ../../../../central/image/datastore/store/store.go .

package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for images.
type Store interface {
	Count(ctx context.Context) (int, error)

	Get(ctx context.Context, id string) (*storage.Image, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, image *storage.Image) error
}
