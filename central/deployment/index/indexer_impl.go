package index

import (
	"bytes"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/options/deployments"
)

const batchSize = 5000

const resourceName = "Deployment"

type indexerImpl struct {
	index bleve.Index
}

type deploymentWrapper struct {
	*storage.Deployment `json:"deployment"`
	Type                string `json:"type"`
}

func (b *indexerImpl) AddDeployment(deployment *storage.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Deployment")

	wrapper := &deploymentWrapper{
		Deployment: deployment,
		Type:       v1.SearchCategory_DEPLOYMENTS.String(),
	}
	if err := b.index.Index(deployment.GetId(), wrapper); err != nil {
		return err
	}
	return nil
}

func (b *indexerImpl) AddDeployments(deployments []*storage.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Deployment")
	batchManager := batcher.New(len(deployments), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := b.processBatch(deployments[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (b *indexerImpl) processBatch(deployments []*storage.Deployment) error {
	batch := b.index.NewBatch()
	for _, deployment := range deployments {
		if err := batch.Index(deployment.GetId(), &deploymentWrapper{
			Deployment: deployment,
			Type:       v1.SearchCategory_DEPLOYMENTS.String(),
		}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

func (b *indexerImpl) DeleteDeployment(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Deployment")
	if err := b.index.Delete(id); err != nil {
		return err
	}
	return nil
}

func (b *indexerImpl) DeleteDeployments(ids []string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.RemoveMany, "Deployment")
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

func (b *indexerImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Deployment")
	return blevesearch.RunSearchRequest(v1.SearchCategory_DEPLOYMENTS, q, b.index, deployments.OptionsMap, opts...)
}
