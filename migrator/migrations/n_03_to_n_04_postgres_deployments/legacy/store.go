package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality.
type Store interface {
	GetListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error)
	GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error)

	Get(ctx context.Context, id string) (*storage.Deployment, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Deployment, []int, error)

	Count(ctx context.Context) (int, error)
	Upsert(ctx context.Context, deployment *storage.Deployment) error
	UpsertMany(ctx context.Context, deployments []*storage.Deployment) error
	Delete(ctx context.Context, id string) error

	GetIDs(ctx context.Context) ([]string, error)
}
