package index

import (
	bleve "github.com/blevesearch/bleve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer encapsulates the deployment indexer
type Indexer interface {
	AddDeployment(deployment *storage.Deployment) error
	AddDeployments(deployments []*storage.Deployment) error
	DeleteDeployment(id string) error
	DeleteDeployments(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
	Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
}

// New creates a new deployment indexer
func New(index bleve.Index, processIndex bleve.Index) Indexer {
	return &indexerImpl{index: index, processIndex: processIndex}
}
