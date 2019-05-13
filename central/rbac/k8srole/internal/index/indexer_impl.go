package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srole/search/options"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

type roleWrapper struct {
	// Json name of this field must match what is used in k8srole/search/options/map
	*storage.K8SRole `json:"k8srole"`
	Type             string `json:"type"`
}

func wrap(role *storage.K8SRole) *roleWrapper {
	return &roleWrapper{Type: v1.SearchCategory_ROLES.String(), K8SRole: role}
}

type indexerImpl struct {
	index bleve.Index
}

func (i *indexerImpl) processBatch(roles []*storage.K8SRole) error {
	batch := i.index.NewBatch()
	for _, role := range roles {
		if err := batch.Index(role.GetId(), wrap(role)); err != nil {
			return err
		}
	}
	return i.index.Batch(batch)
}

func (i *indexerImpl) UpsertRole(role *storage.K8SRole) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "K8SRole")
	return i.index.Index(role.GetId(), wrap(role))
}

func (i *indexerImpl) UpsertRoles(roles ...*storage.K8SRole) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "K8SRole")
	batchManager := batcher.New(len(roles), batchSize)
	for start, end, ok := batchManager.Next(); ok; {
		if err := i.processBatch(roles[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (i *indexerImpl) RemoveRole(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "K8SRole")
	return i.index.Delete(id)
}

func (i *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "K8sRole")
	return blevesearch.RunSearchRequest(v1.SearchCategory_ROLES, q, i.index, options.Map)
}
