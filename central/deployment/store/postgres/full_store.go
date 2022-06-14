package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/stackrox/central/deployment/store"
	"github.com/stackrox/stackrox/central/deployment/store/types"
	"github.com/stackrox/stackrox/generated/storage"
)

// NewFullStore augments the generated store with ListDeployment functions
func NewFullStore(ctx context.Context, db *pgxpool.Pool) store.Store {
	return &fullStoreImpl{
		Store: New(db),
	}
}

type fullStoreImpl struct {
	Store
}

// GetListDeployment returns the list deployment of the passed ID
func (f *fullStoreImpl) GetListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	dep, exists, err := f.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}
	return types.ConvertDeploymentToDeploymentList(dep), true, nil
}

// GetManyListDeployments returns the list deployments as specified by the passed IDs
func (f *fullStoreImpl) GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error) {
	deployments, missing, err := f.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	listDeployments := make([]*storage.ListDeployment, 0, len(deployments))
	for _, d := range deployments {
		listDeployments = append(listDeployments, types.ConvertDeploymentToDeploymentList(d))
	}
	return listDeployments, missing, nil
}
