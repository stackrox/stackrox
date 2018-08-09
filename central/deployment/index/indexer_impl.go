package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/central/deployment/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// AlertIndex provides storage functionality for alerts.
type indexerImpl struct {
	index bleve.Index
}

type deploymentWrapper struct {
	*v1.Deployment `json:"deployment"`
	Type           string `json:"type"`
}

// AddDeployment adds the deployment to the index
func (b *indexerImpl) AddDeployment(deployment *v1.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Deployment")
	return b.index.Index(deployment.GetId(), &deploymentWrapper{Type: v1.SearchCategory_DEPLOYMENTS.String(), Deployment: deployment})
}

// AddDeployments adds the deployments to the index
func (b *indexerImpl) AddDeployments(deployments []*v1.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "AddBatch", "Deployment")

	batch := b.index.NewBatch()
	for _, deployment := range deployments {
		batch.Index(deployment.GetId(), &deploymentWrapper{Type: v1.SearchCategory_DEPLOYMENTS.String(), Deployment: deployment})
	}
	return b.index.Batch(batch)
}

// DeleteDeployment deletes the deployment from the index
func (b *indexerImpl) DeleteDeployment(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Deployment")
	return b.index.Delete(id)
}

// SearchDeployments takes a SearchRequest and finds any matches
func (b *indexerImpl) SearchDeployments(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Deployment")
	return blevesearch.RunSearchRequest(v1.SearchCategory_DEPLOYMENTS.String(), request, b.index, ScopeToDeploymentQuery, mappings.OptionsMap)
}

// ScopeToDeploymentQuery returns a deployment query for the given scope.
func ScopeToDeploymentQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewMatchPhrasePrefixQuery("deployment.cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewMatchPhrasePrefixQuery("deployment.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewMatchPhrasePrefixQuery("deployment.labels.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewMatchPhrasePrefixQuery("deployment.labels.value", scope.GetLabel().GetValue()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchNoneQuery()
	}
	return conjunctionQuery
}
