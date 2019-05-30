package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator/index/internal/index"
	ops "github.com/stackrox/rox/pkg/metrics"
)

// AlertIndex provides storage functionality for alerts.
type indexerImpl struct {
	bleveIndex bleve.Index
	index.Indexer
}

// DeleteIndicator deletes the indicator from the index
func (b *indexerImpl) DeleteProcessIndicators(ids ...string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.RemoveMany, "ProcessIndicator")
	batch := b.bleveIndex.NewBatch()
	for _, id := range ids {
		batch.Delete(id)
	}
	return b.bleveIndex.Batch(batch)
}
