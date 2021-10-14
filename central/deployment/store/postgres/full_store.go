package postgres

import (
	"database/sql"

	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/objects"
)

type fullStore struct {
	Store
}

func NewFullStore(db *sql.DB) store.Store {
	return &fullStore{
		Store: New(db),
	}
}

func (f *fullStore) ListDeployment(id string) (*storage.ListDeployment, bool, error) {
	dep, exists, err := f.Get(id)
	if !exists || err != nil {
		return nil, exists, err
	}
	return objects.ToListDeployment(dep), true, nil
}

func (f *fullStore) ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error) {
	deployments, missing, err := f.GetMany(ids)
	if err != nil {
		return nil, nil, err
	}
	listDeps := make([]*storage.ListDeployment, 0, len(deployments))
	for _, d := range deployments {
		listDeps = append(listDeps, objects.ToListDeployment(d))
	}
	return listDeps, missing, err
}

func (f *fullStore) GetDeployment(id string) (*storage.Deployment, bool, error) {
	return f.Get(id)
}

func (f *fullStore) GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error) {
	return f.GetMany(ids)
}

func (f *fullStore) CountDeployments() (int, error) {
	return f.Count()
}

func (f *fullStore) UpsertDeployment(deployment *storage.Deployment) error {
	return f.Upsert(deployment)
}

func (f *fullStore) RemoveDeployment(id string) error {
	return f.Delete(id)
}

func (f *fullStore) GetDeploymentIDs() ([]string, error) {
	return f.GetIDs()
}
