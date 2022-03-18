
package postgres

import (
    "reflect"
	"time"

	mappings "{{.OptionsPath}}"
	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

func init() {
	mapping.RegisterCategoryToTable(v1.{{.SearchCategory}}, walker.Walk(reflect.TypeOf((*{{.Type}})(nil)), baseTable))
}

func NewIndexer(db *pgxpool.Pool) *indexerImpl {
	return &indexerImpl {
		db: db,
	}
}

type indexerImpl struct {
	db *pgxpool.Pool
}

func (b *indexerImpl) Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "{{.TrimmedType}}")

	return postgres.RunCountRequest(v1.{{.SearchCategory}}, q, b.db, mappings.OptionsMap)
}

func (b *indexerImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "{{.TrimmedType}}")

	return postgres.RunSearchRequest(v1.{{.SearchCategory}}, q, b.db, mappings.OptionsMap)
}

//// Stubs for satisfying interfaces

func (b *indexerImpl) Add{{.TrimmedType}}(deployment *{{.Type}}) error {
	return nil
}

func (b *indexerImpl) Add{{.TrimmedType}}s(_ []*{{.Type}}) error {
	return nil
}

func (b *indexerImpl) Delete{{.TrimmedType}}(id string) error {
	return nil
}

func (b *indexerImpl) Delete{{.TrimmedType}}s(_ []string) error {
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}
