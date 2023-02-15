
package postgres

import (
    "reflect"
	"time"

	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/postgres"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

// NewIndexer returns new indexer for `{{.Type}}`.
func NewIndexer(db *postgres.DB) *indexerImpl {
	return &indexerImpl {
		db: db,
	}
}

type indexerImpl struct {
	db *postgres.DB
}

func (b *indexerImpl) Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "{{.TrimmedType}}")

	return pgSearch.RunCountRequest(ctx, v1.{{.SearchCategory}}, q, b.db)
}

func (b *indexerImpl) Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "{{.TrimmedType}}")

	return pgSearch.RunSearchRequest(ctx, v1.{{.SearchCategory}}, q, b.db)
}

//// Stubs for satisfying interfaces

func (b *indexerImpl) Add{{.TrimmedType}}(deployment *{{.Type}}) error {
	return nil
}

func (b *indexerImpl) Add{{pluralType .TrimmedType}}(_ []*{{.Type}}) error {
	return nil
}

func (b *indexerImpl) Delete{{.TrimmedType}}(id string) error {
	return nil
}

func (b *indexerImpl) Delete{{pluralType .TrimmedType}}(_ []string) error {
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}
