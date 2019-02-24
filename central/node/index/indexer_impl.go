package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/node/index/mappings"
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

type nodeWrapper struct {
	*storage.Node `json:"node"`
	Type          string `json:"type"`
}

// AddNode adds the node to the index
func (b *indexerImpl) AddNode(node *storage.Node) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Node")
	return b.index.Index(node.GetId(), &nodeWrapper{Type: v1.SearchCategory_NODES.String(), Node: node})
}

func (b *indexerImpl) processBatch(nodes []*storage.Node) error {
	batch := b.index.NewBatch()
	for _, node := range nodes {
		if err := batch.Index(node.GetId(), &nodeWrapper{Type: v1.SearchCategory_NODES.String(), Node: node}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

// AddNodes adds a slice of nodes to the index
func (b *indexerImpl) AddNodes(nodes []*storage.Node) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Node")
	batchManager := batcher.New(len(nodes), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := b.processBatch(nodes[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteNode deletes the node from the index
func (b *indexerImpl) DeleteNode(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Node")
	return b.index.Delete(id)
}

// Search takes a Query and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Node")
	return blevesearch.RunSearchRequest(v1.SearchCategory_NODES, q, b.index, mappings.OptionsMap)
}
