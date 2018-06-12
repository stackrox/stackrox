package blevesearch

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

type deploymentWrapper struct {
	*v1.Deployment `json:"deployment"`
	Type           string `json:"type"`
}

// AddDeployment adds the deployment to the index
func (b *Indexer) AddDeployment(deployment *v1.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Deployment")
	return b.globalIndex.Index(deployment.GetId(), &deploymentWrapper{Type: v1.SearchCategory_DEPLOYMENTS.String(), Deployment: deployment})
}

// DeleteDeployment deletes the deployment from the index
func (b *Indexer) DeleteDeployment(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Deployment")
	return b.globalIndex.Delete(id)
}

func scopeToDeploymentQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("deployment.cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("deployment.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("deployment.labels.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("deployment.labels.value", scope.GetLabel().GetValue()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchNoneQuery()
	}
	return conjunctionQuery
}

// SearchDeployments takes a SearchRequest and finds any matches
func (b *Indexer) SearchDeployments(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Deployment")
	return runSearchRequest(v1.SearchCategory_DEPLOYMENTS.String(), request, b.globalIndex, scopeToDeploymentQuery, deploymentObjectMap)
}
