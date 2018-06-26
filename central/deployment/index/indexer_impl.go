package index

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/deployment/index/mappings"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
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

// DeleteDeployment deletes the deployment from the index
func (b *indexerImpl) DeleteDeployment(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Deployment")
	return b.index.Delete(id)
}

// SearchDeployments takes a SearchRequest and finds any matches
func (b *indexerImpl) SearchDeployments(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Deployment")
	return blevesearch.RunSearchRequest(v1.SearchCategory_DEPLOYMENTS.String(), request, b.index, ScopeToDeploymentQuery, mappings.ObjectMap)
}

// ScopeToDeploymentQuery returns a deployment query for the given scope.
func ScopeToDeploymentQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("deployment.cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("deployment.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("deployment.labels.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("deployment.labels.value", scope.GetLabel().GetValue()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchNoneQuery()
	}
	return conjunctionQuery
}
