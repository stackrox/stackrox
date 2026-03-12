package store

import (
	"context"

	"github.com/stackrox/rox/central/deployment/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality.
//
//go:generate mockgen-wrapper
type Store interface {
	GetListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error)
	GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error)
	SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error)

	Get(ctx context.Context, id string) (*storage.StoredDeployment, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.StoredDeployment, []int, error)
	Walk(ctx context.Context, fn func(deployment *storage.StoredDeployment) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(deployment *storage.StoredDeployment) error) error

	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Upsert(ctx context.Context, deployment *storage.StoredDeployment) error
	Delete(ctx context.Context, id string) error

	GetIDs(ctx context.Context) ([]string, error)

	GetContainerImageViews(ctx context.Context, q *v1.Query) ([]*views.ContainerImageView, error)
}
