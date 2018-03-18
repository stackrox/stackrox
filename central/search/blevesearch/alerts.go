package blevesearch

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

// This map converts generic definitions of resource targets to the scoped version
// e.g. image.name.registry becomes deployment.containers.image.name.registry in the alert struct
var alertObjectMap = map[string]string{
	"image": "deployment.containers.image",
	"alert": "",
}

// AddAlert adds the alert to the index
func (b *Indexer) AddAlert(alert *v1.Alert) error {
	return b.alertIndex.Index(alert.GetId(), alert)
}

// DeleteAlert deletes the alert from the index
func (b *Indexer) DeleteAlert(id string) error {
	return b.alertIndex.Delete(id)
}

func scopeToAlertQuery(scope *v1.Scope) *query.ConjunctionQuery {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("deployment.cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("deployment.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel() != nil {
		conjunctionQuery.AddQuery(newMatchQuery("deployment.labels."+scope.GetLabel().GetKey(), scope.GetLabel().GetValue()))
	}
	return conjunctionQuery
}

// SearchAlerts takes a SearchRequest and finds any matches
func (b *Indexer) SearchAlerts(request *v1.SearchRequest) ([]string, error) {
	return runSearchRequest(request, b.alertIndex, scopeToAlertQuery, alertObjectMap)
}
