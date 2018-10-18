package datastore

import (
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	deploymentSearch "github.com/stackrox/rox/central/deployment/search"
	deploymentStore "github.com/stackrox/rox/central/deployment/store"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	deploymentStore    deploymentStore.Store
	deploymentIndexer  deploymentIndex.Indexer
	deploymentSearcher deploymentSearch.Searcher

	processDataStore processDataStore.DataStore
}

func (ds *datastoreImpl) Search(q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.deploymentSearcher.Search(q)
}

func (ds *datastoreImpl) ListDeployment(id string) (*v1.ListDeployment, bool, error) {
	return ds.deploymentStore.ListDeployment(id)
}

func (ds *datastoreImpl) SearchListDeployments(q *v1.Query) ([]*v1.ListDeployment, error) {
	return ds.deploymentSearcher.SearchListDeployments(q)
}

// ListDeployments returns all deploymentStore in their minimal form
func (ds *datastoreImpl) ListDeployments() ([]*v1.ListDeployment, error) {
	return ds.deploymentStore.ListDeployments()
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.deploymentSearcher.SearchDeployments(q)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(q *v1.Query) ([]*v1.Deployment, error) {
	return ds.deploymentSearcher.SearchRawDeployments(q)
}

// GetDeployment
func (ds *datastoreImpl) GetDeployment(id string) (*v1.Deployment, bool, error) {
	return ds.deploymentStore.GetDeployment(id)
}

// GetDeployments
func (ds *datastoreImpl) GetDeployments() ([]*v1.Deployment, error) {
	return ds.deploymentStore.GetDeployments()
}

// CountDeployments
func (ds *datastoreImpl) CountDeployments() (int, error) {
	return ds.deploymentStore.CountDeployments()
}

// UpsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) UpsertDeployment(deployment *v1.Deployment) error {
	if err := ds.deploymentStore.UpsertDeployment(deployment); err != nil {
		return err
	}
	return ds.deploymentIndexer.AddDeployment(deployment)
}

// UpdateDeployment updates a deployment in deploymentStore and in the deploymentIndexer
func (ds *datastoreImpl) UpdateDeployment(deployment *v1.Deployment) error {
	if err := ds.deploymentStore.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.deploymentIndexer.AddDeployment(deployment)
}

// RemoveDeployment removes an alert from the deploymentStore and the deploymentIndexer
func (ds *datastoreImpl) RemoveDeployment(id string) error {
	if err := ds.deploymentStore.RemoveDeployment(id); err != nil {
		return err
	}
	if err := ds.deploymentIndexer.DeleteDeployment(id); err != nil {
		return err
	}
	return ds.processDataStore.RemoveProcessIndicatorsByDeployment(id)
}
