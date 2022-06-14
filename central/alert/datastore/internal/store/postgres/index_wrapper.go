package postgres

import (
	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
)

// NewIndexWrapper returns a wrapper around the generated postgres indexer code
// which satisfies the alert bleve indexer interface (using storage.ListAlert
// instead of storage.Alert as input parameters)
func NewIndexWrapper(db *pgxpool.Pool) *indexWrapperImpl {
	return &indexWrapperImpl{
		indexer: NewIndexer(db),
	}
}

type indexWrapperImpl struct {
	indexer *indexerImpl
}

func (w *indexWrapperImpl) AddListAlert(_ *storage.ListAlert) error {
	return nil
}

func (w *indexWrapperImpl) AddListAlerts(_ []*storage.ListAlert) error {
	return nil
}

func (w *indexWrapperImpl) Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	return w.indexer.Count(q, opts...)
}

func (w *indexWrapperImpl) DeleteListAlert(_ string) error {
	return nil
}

func (w *indexWrapperImpl) DeleteListAlerts(_ []string) error {
	return nil
}

func (w *indexWrapperImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (w *indexWrapperImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}

func (w *indexWrapperImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	return w.indexer.Search(q, opts...)
}
