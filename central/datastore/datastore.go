package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DataStore is a wrapper around the flow of data
type DataStore struct {
	db.Storage // This is an embedded type so we don't have to override all functions. Indexing is a subset of Storage
	indexer    search.Indexer
}

// NewDataStore takes in a storage implementation and and indexer implementation
func NewDataStore(storage db.Storage, indexer search.Indexer) *DataStore {
	return &DataStore{
		Storage: storage,
		indexer: indexer,
	}
}

// Close closes both the database and the indexer
func (ds *DataStore) Close() {
	ds.Storage.Close()
	ds.indexer.Close()
}

// AddAlert inserts an alert into storage and into the indexer
func (ds *DataStore) AddAlert(alert *v1.Alert) error {
	if err := ds.Storage.AddAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// UpdateAlert updates an alert in storage and in the indexer
func (ds *DataStore) UpdateAlert(alert *v1.Alert) error {
	if err := ds.Storage.UpdateAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// RemoveAlert removes an alert from the storage and the indexer
func (ds *DataStore) RemoveAlert(id string) error {
	if err := ds.Storage.RemoveAlert(id); err != nil {
		return err
	}
	return ds.indexer.DeleteAlert(id)
}

// SearchAlerts retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	results, err := ds.indexer.SearchAlerts(request)
	if err != nil {
		return nil, err
	}
	// TODO(cgorman) Optimize into one request
	var protoResults []*v1.SearchResult
	for _, result := range results {
		alert, exists, err := ds.GetAlert(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		protoResults = append(protoResults, search.ConvertAlert(alert, result))
	}
	return protoResults, nil
}

// SearchImages retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	results, err := ds.indexer.SearchImages(request)
	if err != nil {
		return nil, err
	}
	var protoResults []*v1.SearchResult
	for _, result := range results {
		image, exists, err := ds.GetImage(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		protoResults = append(protoResults, search.ConvertImage(image, result))
	}
	return protoResults, nil
}

// SearchPolicies retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	results, err := ds.indexer.SearchPolicies(request)
	if err != nil {
		return nil, err
	}
	var protoResults []*v1.SearchResult
	for _, result := range results {
		policy, exists, err := ds.GetPolicy(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		protoResults = append(protoResults, search.ConvertPolicy(policy, result))
	}
	return protoResults, nil
}

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	results, err := ds.indexer.SearchDeployments(request)
	if err != nil {
		return nil, err
	}
	var protoResults []*v1.SearchResult
	for _, result := range results {
		deployment, exists, err := ds.GetDeployment(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		protoResults = append(protoResults, search.ConvertDeployment(deployment, result))
	}
	return protoResults, nil
}

// AddDeployment adds a deployment into the storage and the indexer
func (ds *DataStore) AddDeployment(deployment *v1.Deployment) error {
	if err := ds.Storage.AddDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// UpdateDeployment updates a deployment in the storage and the indexer
func (ds *DataStore) UpdateDeployment(deployment *v1.Deployment) error {
	if err := ds.Storage.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// RemoveDeployment removes a deployment from the storage and the indexer
func (ds *DataStore) RemoveDeployment(id string) error {
	if err := ds.Storage.RemoveDeployment(id); err != nil {
		return err
	}
	return ds.indexer.DeleteDeployment(id)
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *DataStore) AddPolicy(policy *v1.Policy) (string, error) {
	id, err := ds.Storage.AddPolicy(policy)
	if err != nil {
		return id, err
	}
	return id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *DataStore) UpdatePolicy(policy *v1.Policy) error {
	if err := ds.Storage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *DataStore) RemovePolicy(id string) error {
	if err := ds.Storage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

// AddImage adds an image to the storage and the indexer
func (ds *DataStore) AddImage(image *v1.Image) error {
	if err := ds.Storage.AddImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// UpdateImage updates an image in storage and the indexer
func (ds *DataStore) UpdateImage(image *v1.Image) error {
	if err := ds.Storage.UpdateImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// RemoveImage removes an image from storage and the indexer
func (ds *DataStore) RemoveImage(id string) error {
	if err := ds.Storage.RemoveImage(id); err != nil {
		return err
	}
	return ds.indexer.DeleteImage(id)
}
