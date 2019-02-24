package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const batchSize = 5000

type secretWrapper struct {
	// Json name of this field must match what is used in secret/search/options/map
	*storage.Secret `json:"secret"`
	Type            string `json:"type"`
}

func wrap(secret *storage.Secret) *secretWrapper {
	return &secretWrapper{Type: v1.SearchCategory_SECRETS.String(), Secret: secret}
}

type indexerImpl struct {
	index bleve.Index
}

func (i *indexerImpl) UpsertSecret(secret *storage.Secret) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Secret")
	return i.index.Index(secret.GetId(), wrap(secret))
}

func (i *indexerImpl) processBatch(secrets []*storage.Secret) error {
	batch := i.index.NewBatch()
	for _, secret := range secrets {
		if err := batch.Index(secret.GetId(), wrap(secret)); err != nil {
			return err
		}
	}
	return i.index.Batch(batch)
}

func (i *indexerImpl) UpsertSecrets(secrets ...*storage.Secret) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Secret")
	batchManager := batcher.New(len(secrets), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := i.processBatch(secrets[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (i *indexerImpl) RemoveSecret(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Secret")
	return i.index.Delete(id)
}
