package index

import (
	"context"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/activecomponent/datastore/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const (
	resourceName = "ActiveComponent"
)

type indexerImpl struct {
	index bleve.Index
}

type activeComponentWrapper struct {
	ActiveComponent *storage.ActiveComponent `json:"active_component"`
	Type            string                   `json:"type"`
}

func (b *indexerImpl) Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, resourceName)
	return blevesearch.RunCountRequest(v1.SearchCategory_ACTIVE_COMPONENT, q, b.index, mappings.OptionsMap, opts...)
}

func (b *indexerImpl) Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, resourceName)
	return blevesearch.RunSearchRequest(v1.SearchCategory_ACTIVE_COMPONENT, q, b.index, mappings.OptionsMap, opts...)
}
