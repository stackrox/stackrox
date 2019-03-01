package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

type indexerImpl struct {
	index bleve.Index
}

type alertWrapper struct {
	*storage.Alert `json:"alert"`
	Type           string `json:"type"`
}

// AddAlert adds the alert to the indexer
func (b *indexerImpl) AddAlert(alert *storage.Alert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Alert")
	return b.index.Index(alert.GetId(), &alertWrapper{Type: v1.SearchCategory_ALERTS.String(), Alert: alert})
}

func (b *indexerImpl) processBatch(alerts []*storage.Alert) error {
	batch := b.index.NewBatch()
	for _, alert := range alerts {
		if err := batch.Index(alert.GetId(), &alertWrapper{Type: v1.SearchCategory_ALERTS.String(), Alert: alert}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

// AddAlerts adds the alerts to the indexer
func (b *indexerImpl) AddAlerts(alerts []*storage.Alert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Alert")
	batchManager := batcher.New(len(alerts), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := b.processBatch(alerts[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAlert deletes the alert from the indexer
func (b *indexerImpl) DeleteAlert(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Alert")
	return b.index.Delete(id)
}

// Search takes a SearchRequest and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Alert")

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
		cq := search.ConjunctionQuery(q, search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
		cq.Pagination = q.GetPagination()
		q = cq
	}

	return blevesearch.RunSearchRequest(v1.SearchCategory_ALERTS, q, b.index, mappings.OptionsMap)
}
