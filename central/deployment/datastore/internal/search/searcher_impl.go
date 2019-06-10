package search

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/mappings"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	deploymentsSearchHelper = sac.ForResource(resources.Deployment).MustCreateSearchHelper(mappings.OptionsMap, sac.ClusterIDAndNamespaceFields)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store

	indexer index.Indexer
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *searcherImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	deployments, err := ds.searchDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *searcherImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	deployments, _, err := ds.searchListDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

func (ds *searcherImpl) searchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	ids := search.ResultsToIDs(results)
	deployments, missingIndices, err := ds.storage.ListDeploymentsWithIDs(ids...)
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return deployments, results, nil
}

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	deployments, results, err := ds.searchListDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, convertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	deployments, _, err := ds.storage.GetDeploymentsWithIDs(ids...)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return deploymentsSearchHelper.Apply(ds.indexer.Search)(ctx, q)
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
