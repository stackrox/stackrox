package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality.
//
//go:generate mockgen-wrapper
type Store interface {
	GetListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error)
	GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error)

	Get(ctx context.Context, id string) (*storage.Deployment, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Deployment, []int, error)
	Walk(ctx context.Context, fn func(deployment *storage.Deployment) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(deployment *storage.Deployment) error) error

	Count(ctx context.Context) (int, error)
	Upsert(ctx context.Context, deployment *storage.Deployment) error
	Delete(ctx context.Context, id string) error

	GetIDs(ctx context.Context) ([]string, error)
}
