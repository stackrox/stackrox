package search

import (
	"fmt"

	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store

	indexer index.Indexer
}

func (ds *searcherImpl) buildIndex() error {
	deployments, err := ds.storage.GetDeployments()
	if err != nil {
		return err
	}
	return ds.indexer.AddDeployments(deployments)
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *searcherImpl) SearchRawDeployments(q *v1.Query) ([]*storage.Deployment, error) {
	deployments, err := ds.searchDeployments(q)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *searcherImpl) SearchListDeployments(q *v1.Query) ([]*storage.ListDeployment, error) {
	deployments, _, err := ds.searchListDeployments(q)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

func (ds *searcherImpl) searchListDeployments(q *v1.Query) ([]*storage.ListDeployment, []search.Result, error) {
	results, err := ds.indexer.Search(q)
	if err != nil {
		return nil, nil, err
	}
	var deployments []*storage.ListDeployment
	var newResults []search.Result
	for _, result := range results {
		deployment, exists, err := ds.storage.ListDeployment(result.ID)
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

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchDeployments(q *v1.Query) ([]*v1.SearchResult, error) {
	deployments, results, err := ds.searchListDeployments(q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, convertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchDeployments(q *v1.Query) ([]*storage.Deployment, error) {
	results, err := ds.indexer.Search(q)
	if err != nil {
		return nil, err
	}
	var deployments []*storage.Deployment
	for _, result := range results {
		deployment, exists, err := ds.storage.GetDeployment(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

// ConvertDeployment returns proto search result from a deployment object and the internal search result
func convertDeployment(deployment *storage.ListDeployment, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_DEPLOYMENTS,
		Id:             deployment.GetId(),
		Name:           deployment.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s/%s", deployment.GetCluster(), deployment.GetNamespace()),
	}
}
