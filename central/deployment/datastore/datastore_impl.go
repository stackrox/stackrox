package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/deployment/index"
	"bitbucket.org/stack-rox/apollo/central/deployment/search"
	"bitbucket.org/stack-rox/apollo/central/deployment/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) ListDeployment(id string) (*v1.ListDeployment, bool, error) {
	return ds.storage.ListDeployment(id)
}

func (ds *datastoreImpl) SearchListDeployments(request *v1.ParsedSearchRequest) ([]*v1.ListDeployment, error) {
	return ds.searcher.SearchListDeployments(request)
}

// ListDeployments returns all deployments in their minimal form
func (ds *datastoreImpl) ListDeployments() ([]*v1.ListDeployment, error) {
	return ds.storage.ListDeployments()
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchDeployments(request)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	return ds.searcher.SearchRawDeployments(request)
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

// AddDeployment inserts an alert into storage and into the indexer
func (ds *datastoreImpl) AddDeployment(alert *v1.Deployment) error {
	if err := ds.storage.AddDeployment(alert); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(alert)
}

// UpdateDeployment updates an alert in storage and in the indexer
func (ds *datastoreImpl) UpdateDeployment(alert *v1.Deployment) error {
	if err := ds.storage.UpdateDeployment(alert); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(alert)
}

// RemoveDeployment removes an alert from the storage and the indexer
func (ds *datastoreImpl) RemoveDeployment(id string) error {
	if err := ds.storage.RemoveDeployment(id); err != nil {
		return err
	}
	return ds.indexer.DeleteDeployment(id)
}
