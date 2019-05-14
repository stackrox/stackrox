package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/serviceaccount/search/options"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

type serviceAccountWrapper struct {
	// Json name of this field must match what is used in serviceaccount/search/options/map
	*storage.ServiceAccount `json:"serviceaccount"`
	Type                    string `json:"type"`
}

func wrap(sa *storage.ServiceAccount) *serviceAccountWrapper {
	return &serviceAccountWrapper{Type: v1.SearchCategory_SERVICE_ACCOUNTS.String(), ServiceAccount: sa}
}

type indexerImpl struct {
	index bleve.Index
}

func (i *indexerImpl) UpsertServiceAccount(sa *storage.ServiceAccount) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "ServiceAccount")
	return i.index.Index(sa.GetId(), wrap(sa))
}

func (i *indexerImpl) processBatch(serviceAccounts []*storage.ServiceAccount) error {
	batch := i.index.NewBatch()
	for _, sa := range serviceAccounts {
		if err := batch.Index(sa.GetId(), wrap(sa)); err != nil {
			return err
		}
	}
	return i.index.Batch(batch)
}

func (i *indexerImpl) UpsertServiceAccounts(serviceAccounts ...*storage.ServiceAccount) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "ServiceAccount")
	batchManager := batcher.New(len(serviceAccounts), batchSize)
	for start, end, ok := batchManager.Next(); ok; {
		if err := i.processBatch(serviceAccounts[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (i *indexerImpl) RemoveServiceAccount(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "ServiceAccount")
	return i.index.Delete(id)
}

func (i *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "ServiceAccount")
	return blevesearch.RunSearchRequest(v1.SearchCategory_SERVICE_ACCOUNTS, q, i.index, options.Map)
}
