package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for ImageCVEEdges.
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string, imageID string, imageCveID string, imageCve string, imageCveOperatingSystem string) (bool, error)

	Get(ctx context.Context, id string, imageID string, imageCveID string, imageCve string, imageCveOperatingSystem string) (*storage.ImageCVEEdge, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ImageCVEEdge, []int, error)
}
