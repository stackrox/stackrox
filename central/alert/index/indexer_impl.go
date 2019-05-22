package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

type listAlertWrapper struct {
	*storage.ListAlert `json:"alert"`
	Type               string `json:"type"`
}

// AddAlert adds the alert to the indexer
func (b *indexerImpl) AddListAlert(alert *storage.ListAlert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Alert")
	return b.index.Index(alert.GetId(), &listAlertWrapper{Type: v1.SearchCategory_ALERTS.String(), ListAlert: alert})
}

func (b *indexerImpl) processBatch(alerts []*storage.ListAlert) error {
	batch := b.index.NewBatch()
	for _, alert := range alerts {
		if err := batch.Index(alert.GetId(), &listAlertWrapper{Type: v1.SearchCategory_ALERTS.String(), ListAlert: alert}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

// AddListAlerts adds the alerts to the indexer
func (b *indexerImpl) AddListAlerts(alerts []*storage.ListAlert) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "ListAlert")
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
