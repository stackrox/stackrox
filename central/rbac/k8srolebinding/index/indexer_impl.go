package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search/options"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

type roleBindingWrapper struct {
	// Json name of this field must match what is used in k8srole/search/options/map
	*storage.K8SRoleBinding `json:"k8srolebinding"`
	Type                    string `json:"type"`
}

func wrap(binding *storage.K8SRoleBinding) *roleBindingWrapper {
	return &roleBindingWrapper{Type: v1.SearchCategory_ROLEBINDINGS.String(), K8SRoleBinding: binding}
}

type indexerImpl struct {
	index bleve.Index
}

func (i *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	return blevesearch.RunSearchRequest(v1.SearchCategory_ROLEBINDINGS, q, i.index, options.Map)
}

func (i *indexerImpl) UpsertRoleBinding(binding *storage.K8SRoleBinding) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "K8SRoleBinding")
	return i.index.Index(binding.GetId(), wrap(binding))
}

func (i *indexerImpl) processBatch(bindings []*storage.K8SRoleBinding) error {
	batch := i.index.NewBatch()
	for _, binding := range bindings {
		if err := batch.Index(binding.GetId(), wrap(binding)); err != nil {
			return err
		}
	}
	return i.index.Batch(batch)
}

func (i *indexerImpl) UpsertRoleBindings(bindings ...*storage.K8SRoleBinding) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "K8SRoleBinding")
	batchManager := batcher.New(len(bindings), batchSize)
	for start, end, ok := batchManager.Next(); ok; {
		if err := i.processBatch(bindings[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (i *indexerImpl) RemoveRoleBinding(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "K8SRoleBinding")
	return i.index.Delete(id)
}
