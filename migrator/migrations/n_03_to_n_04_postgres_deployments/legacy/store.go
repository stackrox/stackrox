package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality.
type Store interface {
	GetMany(ctx context.Context, ids []string) ([]*storage.Deployment, []int, error)
	UpsertMany(ctx context.Context, deployments []*storage.Deployment) error
	GetIDs(ctx context.Context) ([]string, error)
}
