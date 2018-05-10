package blevesearch

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/central/search"
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
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Alert")
	return b.alertIndex.Index(alert.GetId(), alert)
}

// DeleteAlert deletes the alert from the index
func (b *Indexer) DeleteAlert(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Alert")
	return b.alertIndex.Delete(id)
}

func scopeToAlertQuery(scope *v1.Scope) query.Query {
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

// SearchAlerts takes a SearchRequest and finds any matches
func (b *Indexer) SearchAlerts(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Alert")
	return runSearchRequest(request, b.alertIndex, scopeToAlertQuery, alertObjectMap)
}
