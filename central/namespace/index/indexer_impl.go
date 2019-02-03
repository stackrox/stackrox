package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/namespace/index/mappings"
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

type namespaceWrapper struct {
	*storage.Namespace `json:"namespace"`
	Type               string `json:"type"`
}

// AddNamespace adds the cluster to the index
func (b *indexerImpl) AddNamespace(namespace *storage.Namespace) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Namespace")
	return b.index.Index(namespace.GetId(), &namespaceWrapper{Type: v1.SearchCategory_NAMESPACES.String(), Namespace: namespace})
}

func (b *indexerImpl) processBatch(namespaces []*storage.Namespace) error {
	batch := b.index.NewBatch()
	for _, namespace := range namespaces {
		batch.Index(namespace.GetId(), &namespaceWrapper{Type: v1.SearchCategory_NAMESPACES.String(), Namespace: namespace})
	}
	return b.index.Batch(batch)
}

// AddNamespaces adds a slice of namespaces to the index
func (b *indexerImpl) AddNamespaces(namespaces []*storage.Namespace) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Namespace")
	batchManager := batcher.New(len(namespaces), batchSize)
	for start, end, ok := batchManager.Next(); ok; start, end, ok = batchManager.Next() {
		if err := b.processBatch(namespaces[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteNamespace deletes the namespace from the index
func (b *indexerImpl) DeleteNamespace(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Namespace")
	return b.index.Delete(id)
}

// Search takes a Query and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Namespace")
	return blevesearch.RunSearchRequest(v1.SearchCategory_NAMESPACES, q, b.index, mappings.OptionsMap)
}
