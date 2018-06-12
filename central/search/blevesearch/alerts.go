package blevesearch

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

type alertWrapper struct {
	*v1.Alert `json:"alert"`
	Type      string `json:"type"`
}

// AddAlert adds the alert to the index
func (b *Indexer) AddAlert(alert *v1.Alert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Alert")
	return b.globalIndex.Index(alert.GetId(), &alertWrapper{Type: v1.SearchCategory_ALERTS.String(), Alert: alert})
}

// DeleteAlert deletes the alert from the index
func (b *Indexer) DeleteAlert(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Alert")
	return b.globalIndex.Delete(id)
}

func scopeToAlertQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("alert.deployment.cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("alert.deployment.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("alert.deployment.labels.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("alert.deployment.labels.value", scope.GetLabel().GetValue()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchNoneQuery()
	}
	return conjunctionQuery
}

// SearchAlerts takes a SearchRequest and finds any matches
func (b *Indexer) SearchAlerts(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Alert")
	searchField := search.AlertOptionsMap[search.Stale]
	if request.Fields == nil {
		request.Fields = make(map[string]*v1.ParsedSearchRequest_Values)
	}
	if values, ok := request.Fields[searchField.GetFieldPath()]; !ok || len(values.Values) == 0 {
		request.Fields[searchField.GetFieldPath()] = &v1.ParsedSearchRequest_Values{
			Values: []string{"false"},
			Field:  searchField,
		}
	}
	return runSearchRequest(v1.SearchCategory_ALERTS.String(), request, b.globalIndex, scopeToAlertQuery, alertObjectMap)
}
