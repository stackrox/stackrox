package postgres

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
)

func compileSubQuery(ctx context.Context, sub *v1.MatchFieldSubQuery, fields map[string]searchFieldMetadata) (*pgsearch.QueryEntry, error) {
	field := fields[sub.GetField()]
	if field.baseField == nil || field.derivedMetadata != nil || !isSubQueryScalar(field.baseField.DataType) {
		return nil, errox.InvalidArgs.Newf("subquery requires a scalar field, got %q", sub.GetField())
	}
	schema := mapping.GetTableFromCategory(sub.GetCategory())
	if schema == nil {
		return nil, errox.InvalidArgs.Newf("no schema registered for subquery category %s", sub.GetCategory())
	}
	if len(sub.GetQuery().GetSelects()) != 1 {
		return nil, errox.InvalidArgs.New("subquery must select exactly one field")
	}
	// Use the same SELECT compiler as views, including the inner category's SAC
	// and scope filters. Keep placeholders unnumbered until the full query is built.
	compiled, err := standardizeSelectQueryAndPopulatePath(ctx, sub.GetQuery().CloneVT(), schema, SELECT, nil)
	if err != nil {
		return nil, err
	}
	if compiled == nil {
		return pgsearch.NewFalseQuery(), nil
	}
	selected := compiled.SelectedFields[0]
	if !isSubQueryScalar(selected.FieldType) {
		return nil, errox.InvalidArgs.New("subquery must select a scalar field")
	}
	outer := selectQueryField(sub.GetField(), field.baseField, false, aggregatefunc.Unset, "")
	// The derived table isolates inner table names and any extra columns required
	// for SELECT DISTINCT, grouping or ordering. EXISTS does not duplicate outer rows.
	return &pgsearch.QueryEntry{Where: pgsearch.WhereClause{
		Query: fmt.Sprintf("EXISTS (SELECT 1 FROM (%s) AS subquery WHERE subquery.%s = %s)",
			compiled.sqlTemplate(), selected.Alias, outer.SelectPath),
		Values: compiled.Data,
	}}, nil
}

func isSubQueryScalar(dataType postgres.DataType) bool {
	switch dataType {
	case postgres.Map, postgres.StringArray, postgres.EnumArray, postgres.IntArray:
		return false
	default:
		return true
	}
}
