package search

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/deployment/index"
	"bitbucket.org/stack-rox/apollo/central/deployment/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store

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
func (ds *searcherImpl) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	deployments, _, err := ds.searchDeployments(request)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	deployments, results, err := ds.searchDeployments(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, convertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, []search.Result, error) {
	results, err := ds.indexer.SearchDeployments(request)
	if err != nil {
		return nil, nil, err
	}
	var deployments []*v1.Deployment
	var newResults []search.Result
	for _, result := range results {
		deployment, exists, err := ds.storage.GetDeployment(result.ID)
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

// ConvertDeployment returns proto search result from a deployment object and the internal search result
func convertDeployment(deployment *v1.Deployment, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_DEPLOYMENTS,
		Id:             deployment.GetId(),
		Name:           deployment.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s/%s", deployment.GetClusterName(), deployment.GetNamespace()),
	}
}
