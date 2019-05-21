package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/deployment/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

type deploymentWrapper struct {
	*storage.Deployment `json:"deployment"`
	Type                string `json:"type"`
}

// AddDeployment adds the deployment to the index
func (b *indexerImpl) AddDeployment(deployment *storage.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Deployment")
	return b.index.Index(deployment.GetId(), &deploymentWrapper{Type: v1.SearchCategory_DEPLOYMENTS.String(), Deployment: deployment})
}

func (b *indexerImpl) processBatch(deployments []*storage.Deployment) error {
	batch := b.index.NewBatch()
	for _, deployment := range deployments {
		if err := batch.Index(deployment.GetId(), &deploymentWrapper{Type: v1.SearchCategory_DEPLOYMENTS.String(), Deployment: deployment}); err != nil {
			return err
		}
	}
	return b.index.Batch(batch)
}

// AddDeployments adds the deployments to the index
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

// DeleteDeployment deletes the deployment from the index
func (b *indexerImpl) DeleteDeployment(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Deployment")
	return b.index.Delete(id)
}

// Search takes a Query and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Deployment")
	return blevesearch.RunSearchRequest(v1.SearchCategory_DEPLOYMENTS, q, b.index, mappings.OptionsMap)
}
