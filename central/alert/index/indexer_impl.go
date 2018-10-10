package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
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
func (b *indexerImpl) SearchAlerts(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Alert")

	var querySpecifiesStateField bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.ViolationState.String() {
			querySpecifiesStateField = true
		}
	})

	// By default, set stale to false.
	if !querySpecifiesStateField {
		q = search.ConjunctionQuery(q, search.NewQueryBuilder().AddStrings(search.ViolationState, v1.ViolationState_ACTIVE.String()).ProtoQuery())
	}

	return blevesearch.RunSearchRequest(v1.SearchCategory_ALERTS, q, b.index, mappings.OptionsMap)
}
