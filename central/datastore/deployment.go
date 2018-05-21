package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DeploymentDataStore is an intermediary to DeploymentStorage.
type DeploymentDataStore interface {
	db.DeploymentStorage

	SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error)
}

// NewDeploymentDataStore provides a new instance of DeploymentDataStore
func NewDeploymentDataStore(storage db.DeploymentStorage, indexer search.DeploymentIndex) (DeploymentDataStore, error) {
	ds := &deploymentDataStoreImpl{
		DeploymentStorage: storage,
		indexer:           indexer,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

// deploymentDataStoreImpl provides an intermediary implementation layer for DeploymentStorage.
type deploymentDataStoreImpl struct {
	db.DeploymentStorage
	indexer search.DeploymentIndex
}

func (ds *deploymentDataStoreImpl) buildIndex() error {
	deployments, err := ds.GetDeployments()
	if err != nil {
		return err
	}
	for _, d := range deployments {
		if err := ds.indexer.AddDeployment(d); err != nil {
			logger.Errorf("Error inserting deployment %s (%s) into index: %s", d.GetId(), d.GetName(), err)
		}
	}
	return nil
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *deploymentDataStoreImpl) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	deployments, _, err := ds.searchDeployments(request)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *deploymentDataStoreImpl) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	deployments, results, err := ds.searchDeployments(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, search.ConvertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

// AddDeployment adds a deployment into the storage and the indexer
func (ds *deploymentDataStoreImpl) AddDeployment(deployment *v1.Deployment) error {
	if err := ds.DeploymentStorage.AddDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// UpdateDeployment updates a deployment in the storage and the indexer
func (ds *deploymentDataStoreImpl) UpdateDeployment(deployment *v1.Deployment) error {
	if err := ds.DeploymentStorage.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// RemoveDeployment removes a deployment from the storage and the indexer
func (ds *deploymentDataStoreImpl) RemoveDeployment(id string) error {
	if err := ds.DeploymentStorage.RemoveDeployment(id); err != nil {
		return err
	}
	// Remove from index since it is buried in the graveyard.
	return ds.indexer.DeleteDeployment(id)
}

func (ds *deploymentDataStoreImpl) searchDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, []search.Result, error) {
	results, err := ds.indexer.SearchDeployments(request)
	if err != nil {
		return nil, nil, err
	}
	var deployments []*v1.Deployment
	var newResults []search.Result
	for _, result := range results {
		deployment, exists, err := ds.GetDeployment(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		deployments = append(deployments, deployment)
		newResults = append(newResults, result)
	}
	return deployments, newResults, nil
}
