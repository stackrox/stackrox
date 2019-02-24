package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator/index/mappings"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

// AlertIndex provides storage functionality for alerts.
type indexerImpl struct {
	index bleve.Index
}

type indicatorWrapper struct {
	*storage.ProcessIndicator `json:"process_indicator"`
	Type                      string `json:"type"`
}

// AddProcessIndicator adds the indicator to the index
func (b *indexerImpl) AddProcessIndicator(indicator *storage.ProcessIndicator) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "ProcessIndicator")
	return b.index.Index(indicator.GetId(), &indicatorWrapper{Type: v1.SearchCategory_PROCESS_INDICATORS.String(), ProcessIndicator: indicator})
}

func (b *indexerImpl) processBatch(indicators []*storage.ProcessIndicator) error {
	batch := b.index.NewBatch()
	for _, indicator := range indicators {
		if err := batch.Index(indicator.GetId(), &indicatorWrapper{Type: v1.SearchCategory_PROCESS_INDICATORS.String(), ProcessIndicator: indicator}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

// AddIndicators adds the indicators to the indexer
func (b *indexerImpl) AddProcessIndicators(indicators []*storage.ProcessIndicator) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "ProcessIndicator")
	batchManager := batcher.New(len(indicators), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := b.processBatch(indicators[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteIndicator deletes the indicator from the index
func (b *indexerImpl) DeleteProcessIndicator(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	return b.index.Delete(id)
}

// DeleteIndicator deletes the indicator from the index
func (b *indexerImpl) DeleteProcessIndicators(ids ...string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.RemoveMany, "ProcessIndicator")
	batch := b.index.NewBatch()
	for _, id := range ids {
		batch.Delete(id)
	}
	return b.index.Batch(batch)
}

// SearchIndicators takes a SearchRequest and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "ProcessIndicator")
	return blevesearch.RunSearchRequest(v1.SearchCategory_PROCESS_INDICATORS, q, b.index, mappings.OptionsMap)
}
