package postgres

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
)

func compileSubQuery(ctx context.Context, sub *v1.MatchFieldSubQuery, fields map[string]searchFieldMetadata) (*pgsearch.QueryEntry, error) {
	field := fields[sub.GetField()]
	if field.baseField == nil || field.derivedMetadata != nil || !isSingleValueFieldType(field.baseField.DataType) {
		return nil, errox.InvalidArgs.Newf("subquery requires a stored field with one value per record, got %q", sub.GetField())
	}
	schema := mapping.GetTableFromCategory(sub.GetCategory())
	if schema == nil {
		return nil, errox.InvalidArgs.Newf("no schema registered for subquery category %s", sub.GetCategory())
	}
	if len(sub.GetQuery().GetSelects()) != 1 {
		return nil, errox.InvalidArgs.New("subquery must select exactly one field")
	}
	selection := sub.GetQuery().GetSelects()[0]
	if search.IsDerivedField(selection.GetField().GetName()) || selection.GetField().GetAggregateFunc() != "" ||
		selection.GetFilter() != nil || len(sub.GetQuery().GetGroupBy().GetFields()) != 0 {
		return nil, errox.InvalidArgs.New("subqueries do not support derived selections, aggregates, select filters or grouping")
	}
	// The view SELECT compiler applies the inner query's access and scope filters.
	// Assign parameter numbers after combining both queries so their values stay in order.
	compiled, err := standardizeSelectQueryAndPopulatePath(ctx, sub.GetQuery().CloneVT(), schema, SELECT, nil)
	if err != nil {
		return nil, err
	}
	if compiled == nil {
		return pgsearch.NewFalseQuery(), nil
	}
	// Fields such as deployment counts can add aggregation during compilation.
	if len(compiled.GroupBys) != 0 || compiled.Having != "" {
		return nil, errox.InvalidArgs.New("subqueries do not support aggregate filters or grouping")
	}
	for _, order := range compiled.Pagination.OrderBys {
		if order.Field.DerivedField {
			return nil, errox.InvalidArgs.New("subqueries do not support derived ordering")
		}
	}
	if len(compiled.SelectedFields) == 0 {
		return nil, errox.InvalidArgs.New("subquery did not produce a selected field")
	}
	selected := compiled.SelectedFields[0]
	if !isSingleValueFieldType(selected.FieldType) {
		return nil, errox.InvalidArgs.New("subquery cannot select an array or map field")
	}
	// Highlights can add columns with the same name as the selected field.
	// Keep only the explicit selection so the outer reference is unambiguous.
	compiled.SelectedFields = compiled.SelectedFields[:1]
	outer := selectQueryField(sub.GetField(), field.baseField, false, aggregatefunc.Unset, "")
	// Wrap the inner query to keep its table names and any extra sorting columns
	// separate from the outer query. EXISTS checks for a matching value without
	// duplicating outer records when several inner records contain that value.
	return &pgsearch.QueryEntry{Where: pgsearch.WhereClause{
		Query: fmt.Sprintf("EXISTS (SELECT 1 FROM (%s) AS subquery WHERE subquery.%s = %s)",
			compiled.sqlTemplate(), selected.Alias, outer.SelectPath),
		Values: compiled.Data,
	}}, nil
}

func isSingleValueFieldType(dataType postgres.DataType) bool {
	switch dataType {
	case postgres.Map, postgres.StringArray, postgres.EnumArray, postgres.IntArray:
		return false
	default:
		return true
	}
}
