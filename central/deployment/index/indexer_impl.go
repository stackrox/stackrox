package index

import (
	"time"

	"github.com/blevesearch/bleve/document"
	"github.com/blevesearch/bleve/index/upsidedown"
	"github.com/stackrox/rox/central/deployment/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/blevehelper"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

const resourceName = "Deployment"

type indexerImpl struct {
	index *blevehelper.BleveWrapper
}

type deploymentWrapper struct {
	*storage.Deployment `json:"deployment"`
	Type                string `json:"type"`
}

func (b *indexerImpl) memIndex(deployment *deploymentWrapper, udc *upsidedown.UpsideDownCouch) error {
	doc := document.NewDocument(deployment.GetId())
	err := b.index.Mapping().MapDocument(doc, deployment)
	if err != nil {
		return err
	}
	return udc.UpdateWithAnalysis(doc, udc.Analyze(doc), nil)
}

func (b *indexerImpl) AddDeployment(deployment *storage.Deployment) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Deployment")

	wrapper := &deploymentWrapper{
		Deployment: deployment,
		Type:       v1.SearchCategory_DEPLOYMENTS.String(),
	}
	bleveIndex, _, err := b.index.Advanced()
	if err != nil {
		return err
	}
	udc, ok := bleveIndex.(*upsidedown.UpsideDownCouch)
	if ok {
		return b.memIndex(wrapper, udc)
	}
	if err := b.index.Index.Index(deployment.GetId(), wrapper); err != nil {
		return err
	}
	return b.index.IncTxnCount()
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
	return b.index.IncTxnCount()
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
	return b.index.IncTxnCount()
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
	return b.index.IncTxnCount()
}

func (b *indexerImpl) GetTxnCount() uint64 {
	return b.index.GetTxnCount()
}

func (b *indexerImpl) ResetIndex() error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Reset, "Deployment")
	return blevesearch.ResetIndex(v1.SearchCategory_DEPLOYMENTS, b.index.Index)
}

func (b *indexerImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Deployment")
	return blevesearch.RunSearchRequest(v1.SearchCategory_DEPLOYMENTS, q, b.index.Index, mappings.OptionsMap, opts...)
}

func (b *indexerImpl) SetTxnCount(seq uint64) error {
	return b.index.SetTxnCount(seq)
}
