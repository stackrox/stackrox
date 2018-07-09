package index

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/alert/index/mappings"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

type indexerImpl struct {
	index bleve.Index
}

type alertWrapper struct {
	*v1.Alert `json:"alert"`
	Type      string `json:"type"`
}

// AddAlert adds the alert to the indexer
func (b *indexerImpl) AddAlert(alert *v1.Alert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Alert")
	return b.index.Index(alert.GetId(), &alertWrapper{Type: v1.SearchCategory_ALERTS.String(), Alert: alert})
}

// AddAlerts adds the alerts to the indexer
func (b *indexerImpl) AddAlerts(alerts []*v1.Alert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "AddBatch", "Alert")
	batch := b.index.NewBatch()
	for _, alert := range alerts {
		batch.Index(alert.GetId(), &alertWrapper{Type: v1.SearchCategory_ALERTS.String(), Alert: alert})
	}
	return b.index.Batch(batch)
}

// DeleteAlert deletes the alert from the indexer
func (b *indexerImpl) DeleteAlert(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Alert")
	return b.index.Delete(id)
}

// SearchAlerts takes a SearchRequest and finds any matches
func (b *indexerImpl) SearchAlerts(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Alert")
	if request.Fields == nil {
		request.Fields = make(map[string]*v1.ParsedSearchRequest_Values)
	}
	if values, ok := request.Fields[search.Stale]; !ok || len(values.Values) == 0 {
		request.Fields[search.Stale] = &v1.ParsedSearchRequest_Values{
			Values: []string{"false"},
		}
	}
	return blevesearch.RunSearchRequest(v1.SearchCategory_ALERTS.String(), request, b.index, ScopeToAlertQuery, mappings.OptionsMap)
}

// ScopeToAlertQuery returns an alert query for the given scope.
func ScopeToAlertQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("alert.deployment.cluster_name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("alert.deployment.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("alert.deployment.labels.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("alert.deployment.labels.value", scope.GetLabel().GetValue()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchNoneQuery()
	}
	return conjunctionQuery
}
