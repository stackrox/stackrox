{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.Name|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.Name|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $idx, $pk := .}}{{if $idx}} AND {{end}}{{$pk.Name}} = ${{add $idx 1}}{{end}}{{end}}
{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}

{{- $ := . }}
{{- $pks := .Schema.LocalPrimaryKeys }}

{{- $singlePK := dict.nil }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{- end }}

package postgres

import (
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
	"github.com/stackrox/rox/pkg/logging"
)

func init() {
	mapping.RegisterCategoryToTable(v1.{{.SearchCategory}}, table)
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

func (b *indexerImpl) Search{{.TrimmedType}}s(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
    return nil, nil
}

func (b *indexerImpl) SearchRaw{{.TrimmedType}}s(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
    return nil, nil
}

func (b *indexerImpl) SearchList{{.TrimmedType}}s(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
    return nil, nil
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
