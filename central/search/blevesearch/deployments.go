package blevesearch

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

var deploymentObjectMap = map[string]string{
	"image":      "containers.image",
	"deployment": "",
}

// AddDeployment adds the deployment to the index
func (b *Indexer) AddDeployment(deployment *v1.Deployment) error {
	return b.deploymentIndex.Index(deployment.GetId(), deployment)
}

// DeleteDeployment deletes the deployment from the index
func (b *Indexer) DeleteDeployment(id string) error {
	return b.deploymentIndex.Delete(id)
}

func scopeToDeploymentQuery(scope *v1.Scope) *query.ConjunctionQuery {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("namespace", scope.GetNamespace()))
	}
	if scope.GetLabel() != nil {
		conjunctionQuery.AddQuery(newMatchQuery("labels."+scope.GetLabel().GetKey(), scope.GetLabel().GetValue()))
	}
	return conjunctionQuery
}

// SearchDeployments takes a SearchRequest and finds any matches
func (b *Indexer) SearchDeployments(request *v1.SearchRequest) ([]string, error) {
	return runSearchRequest(request, b.deploymentIndex, scopeToDeploymentQuery, deploymentObjectMap)
}
