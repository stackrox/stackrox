package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type secretWrapper struct {
	// Json name of this field must match what is used in secret/search/options/map
	*v1.Secret `json:"secret"`
	Type       string `json:"type"`
}

func wrap(secret *v1.Secret) *secretWrapper {
	return &secretWrapper{Type: v1.SearchCategory_SECRETS.String(), Secret: secret}
}

type indexerImpl struct {
	index bleve.Index
}

func (i *indexerImpl) UpsertSecret(secret *v1.Secret) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Secret")

	return i.index.Index(secret.GetId(), wrap(secret))
}

func (i *indexerImpl) UpsertSecrets(secrets ...*v1.Secret) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Secret")

	batch := i.index.NewBatch()
	for _, secret := range secrets {
		batch.Index(secret.GetId(), wrap(secret))
	}
	return i.index.Batch(batch)
}

func (i *indexerImpl) RemoveSecret(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Secret")

	return i.index.Delete(id)
}
