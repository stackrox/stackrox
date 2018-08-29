package datastore

import (
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/search"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) ListDeployment(id string) (*v1.ListDeployment, bool, error) {
	return ds.storage.ListDeployment(id)
}

func (ds *datastoreImpl) SearchListDeployments(q *v1.Query) ([]*v1.ListDeployment, error) {
	return ds.searcher.SearchListDeployments(q)
}

// ListDeployments returns all deployments in their minimal form
func (ds *datastoreImpl) ListDeployments() ([]*v1.ListDeployment, error) {
	return ds.storage.ListDeployments()
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchDeployments(q)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(q *v1.Query) ([]*v1.Deployment, error) {
	return ds.searcher.SearchRawDeployments(q)
}

// GetDeployment
func (ds *datastoreImpl) GetDeployment(id string) (*v1.Deployment, bool, error) {
	return ds.storage.GetDeployment(id)
}

// GetDeployments
func (ds *datastoreImpl) GetDeployments() ([]*v1.Deployment, error) {
	return ds.storage.GetDeployments()
}

// CountDeployments
func (ds *datastoreImpl) CountDeployments() (int, error) {
	return ds.storage.CountDeployments()
}

// UpsertDeployment inserts a deployment into storage and into the indexer
func (ds *datastoreImpl) UpsertDeployment(deployment *v1.Deployment) error {
	if err := ds.storage.UpsertDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// UpdateDeployment updates a deployment in storage and in the indexer
func (ds *datastoreImpl) UpdateDeployment(deployment *v1.Deployment) error {
	if err := ds.storage.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// RemoveDeployment removes an alert from the storage and the indexer
func (ds *datastoreImpl) RemoveDeployment(id string) error {
	if err := ds.storage.RemoveDeployment(id); err != nil {
		return err
	}
	return ds.indexer.DeleteDeployment(id)
}
