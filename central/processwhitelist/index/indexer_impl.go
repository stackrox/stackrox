package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/processwhitelist/index/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

type indexerImpl struct {
	index bleve.Index
}

type processWhitelistWrapper struct {
	*storage.ProcessWhitelist `json:"process_whitelist"`
	Type                      string `json:"type"`
}

func (b *indexerImpl) AddWhitelist(whitelist *storage.ProcessWhitelist) error {
	return b.index.Index(whitelist.GetId(), &processWhitelistWrapper{Type: v1.SearchCategory_PROCESS_WHITELISTS.String(), ProcessWhitelist: whitelist})
}

func (b *indexerImpl) processBatch(whitelists []*storage.ProcessWhitelist) error {
	batch := b.index.NewBatch()
	for _, whitelist := range whitelists {
		if err := batch.Index(whitelist.GetId(), &processWhitelistWrapper{Type: v1.SearchCategory_PROCESS_WHITELISTS.String(), ProcessWhitelist: whitelist}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

func (b *indexerImpl) AddWhitelists(whitelists []*storage.ProcessWhitelist) error {
	batchManager := batcher.New(len(whitelists), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := b.processBatch(whitelists[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (b *indexerImpl) DeleteWhitelist(id string) error {
	return b.index.Delete(id)
}

func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	return blevesearch.RunSearchRequest(v1.SearchCategory_PROCESS_WHITELISTS, q, b.index, mappings.OptionsMap)
}
