package index

import (
	"bytes"
	"context"
	"time"

	bleve "github.com/blevesearch/bleve"
	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	batcher "github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
	mappings "github.com/stackrox/rox/pkg/search/options/processindicators"
)

const batchSize = 5000

const resourceName = "ProcessIndicator"

type indexerImpl struct {
	index bleve.Index
}

type processIndicatorWrapper struct {
	*storage.ProcessIndicator `json:"process_indicator"`
	Type                      string `json:"type"`
}

func (b *indexerImpl) AddProcessIndicator(processindicator *storage.ProcessIndicator) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "ProcessIndicator")
	if err := b.index.Index(processindicator.GetId(), &processIndicatorWrapper{
		ProcessIndicator: processindicator,
		Type:             v1.SearchCategory_PROCESS_INDICATORS.String(),
	}); err != nil {
		return err
	}
	return nil
}

func (b *indexerImpl) AddProcessIndicators(processindicators []*storage.ProcessIndicator) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "ProcessIndicator")
	batchManager := batcher.New(len(processindicators), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := b.processBatch(processindicators[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (b *indexerImpl) processBatch(processindicators []*storage.ProcessIndicator) error {
	batch := b.index.NewBatch()
	for _, processindicator := range processindicators {
		if err := batch.Index(processindicator.GetId(), &processIndicatorWrapper{
			ProcessIndicator: processindicator,
			Type:             v1.SearchCategory_PROCESS_INDICATORS.String(),
		}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

func (b *indexerImpl) DeleteProcessIndicator(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	if err := b.index.Delete(id); err != nil {
		return err
	}
	return nil
}

func (b *indexerImpl) DeleteProcessIndicators(ids []string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.RemoveMany, "ProcessIndicator")
	batch := b.index.NewBatch()
	for _, id := range ids {
		batch.Delete(id)
	}
	if err := b.index.Batch(batch); err != nil {
		return err
	}
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return b.index.SetInternal([]byte(resourceName), []byte("old"))
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	data, err := b.index.GetInternal([]byte(resourceName))
	if err != nil {
		return false, err
	}
	return !bytes.Equal([]byte("old"), data), nil
}

func (b *indexerImpl) Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "ProcessIndicator")
	return blevesearch.RunSearchRequest(v1.SearchCategory_PROCESS_INDICATORS, q, b.index, mappings.OptionsMap, opts...)
}

// Count returns the number of search results from the query
func (b *indexerImpl) Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "ProcessIndicator")
	return blevesearch.RunCountRequest(v1.SearchCategory_PROCESS_INDICATORS, q, b.index, mappings.OptionsMap, opts...)
}
