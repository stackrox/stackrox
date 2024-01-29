package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
)

// RunSelectRequestForSchema executes a select request against the database for given schema. The input query must
// explicitly specify select fields.
func RunSelectRequestForSchema[T any](ctx context.Context, db postgres.DB, schema *walker.Schema, q *v1.Query) ([]*T, error) {
	var query *query
	var err error
	// Add this to be safe and convert panics to errors,
	// since we do a lot of casting and other operations that could potentially panic in this code.
	// Panics are expected ONLY in the event of a programming error, all foreseeable errors are handled
	// the usual way.
	defer func() {
		if r := recover(); r != nil {
			if query != nil {
				log.Errorf("Query issue: %s: %v", query.AsSQL(), r)
			} else {
				log.Errorf("Unexpected error running search request: %v", r)
			}
			debug.PrintStack()
			err = fmt.Errorf("unexpected error running search request: %v", r)
		}
	}()

	query, err = standardizeSelectQueryAndPopulatePath(ctx, q, schema, SELECT)
	if err != nil {
		return nil, err
	}
	// A nil-query implies no results.
	if query == nil {
		return nil, nil
	}
	return pgutils.Retry2(func() ([]*T, error) {
		return retryableRunSelectRequestForSchema[T](ctx, db, query)
	})
}

func standardizeSelectQueryAndPopulatePath(ctx context.Context, q *v1.Query, schema *walker.Schema, queryType QueryType) (*query, error) {
	nowForQuery := time.Now()

	var err error
	q, err = scopeContextToQuery(ctx, q)
	if err != nil {
		return nil, err
	}

	standardizeFieldNamesInQuery(q)
	innerJoins, dbFields := getJoinsAndFields(schema, q)
	if len(q.GetSelects()) == 0 && q.GetQuery() == nil {
		return nil, nil
	}

	parsedQuery := &query{
		Schema:     schema,
		QueryType:  queryType,
		From:       schema.Table,
		InnerJoins: innerJoins,
	}

	if err = populateSelect(parsedQuery, schema, q.GetSelects(), dbFields, nowForQuery); err != nil {
		return nil, errors.Wrapf(err, "failed to parse select portion of query -- %s --", q.String())
	}

	queryEntry, err := compileQueryToPostgres(schema, q, dbFields, nowForQuery)
	if err != nil {
		return nil, err
	}

	if queryEntry != nil {
		parsedQuery.Where = queryEntry.Where.Query
		parsedQuery.Data = append(parsedQuery.Data, queryEntry.Where.Values...)
		// TODO(ROX-14940): We won't need this once highlights is removed and fields can only be selected when explicitly specified in the query.
		parsedQuery.SelectedFields = append(parsedQuery.SelectedFields, queryEntry.SelectedFields...)
		if queryEntry.Having != nil {
			parsedQuery.Having = queryEntry.Having.Query
			parsedQuery.Data = append(parsedQuery.Data, queryEntry.Having.Values...)
		}
	}

	if err := populateGroupBy(parsedQuery, q.GetGroupBy(), schema, dbFields); err != nil {
		return nil, err
	}
	if err := populatePagination(parsedQuery, q.GetPagination(), schema, dbFields); err != nil {
		return nil, err
	}
	if err := applyPaginationForSearchAfter(parsedQuery); err != nil {
		return nil, err
	}
	return parsedQuery, nil
}

func retryableRunSelectRequestForSchema[T any](ctx context.Context, db postgres.DB, query *query) ([]*T, error) {
	if len(query.SelectedFields) == 0 {
		return nil, errors.New("select fields required for select query")
	}

	queryStr := query.AsSQL()

	rows, err := tracedQuery(ctx, db, queryStr, query.Data...)
	if err != nil {
		return nil, errors.Wrapf(err, "error executing query %s", queryStr)
	}
	defer rows.Close()

	var scannedRows []*T
	if err := pgxscan.ScanAll(&scannedRows, rows); err != nil {
		return nil, err
	}
	return scannedRows, rows.Err()
}

func populateSelect(querySoFar *query, schema *walker.Schema, querySelects []*v1.QuerySelect, queryFields map[string]searchFieldMetadata, nowForQuery time.Time) error {
	if len(querySelects) == 0 {
		return errors.New("select portion of the query cannot be empty")
	}

	for idx, qs := range querySelects {
		field := qs.GetField()
		fieldMetadata := queryFields[field.GetName()]
		dbField := fieldMetadata.baseField
		if dbField == nil {
			return errors.Errorf("field %s in select portion of query does not exist in table %s or connected tables", field, schema.Table)
		}
		// TODO(mandar): Add support for the following.
		if dbField.DataType == postgres.StringArray || dbField.DataType == postgres.IntArray ||
			dbField.DataType == postgres.EnumArray || dbField.DataType == postgres.Map {
			return errors.Errorf("field %s in select portion of query is unsupported", field)
		}

		if qs.GetFilter() == nil {
			querySoFar.SelectedFields = append(querySoFar.SelectedFields,
				selectQueryField(field.GetName(), dbField, field.GetDistinct(), aggregatefunc.GetAggrFunc(field.GetAggregateFunc()), ""),
			)
			continue
		}

		// SQL constraint
		if field.GetAggregateFunc() == aggregatefunc.Unset.Name() {
			return errors.New("FILTER clause can only be applied to aggregate functions")
		}

		filter := qs.GetFilter()
		qe, err := compileQueryToPostgres(schema, filter.GetQuery(), queryFields, nowForQuery)
		if err != nil {
			return errors.New("failed to parse filter in select portion of query")
		}
		if qe == nil || qe.Where.Query == "" {
			return nil
		}
		querySoFar.Data = append(querySoFar.Data, qe.Where.Values...)

		selectField := selectQueryField(field.GetName(), dbField, field.GetDistinct(), aggregatefunc.GetAggrFunc(field.GetAggregateFunc()), qe.Where.Query)
		if alias := filter.GetName(); alias != "" {
			selectField.Alias = alias
		} else {
			selectField.Alias = fmt.Sprintf("%s_%d", selectField.Alias, idx)
		}
		querySoFar.SelectedFields = append(querySoFar.SelectedFields, selectField)
	}
	return nil
}

func selectQueryField(searchField string, field *walker.Field, selectDistinct bool, aggrFunc aggregatefunc.AggrFunc, filter string) pgsearch.SelectQueryField {
	var cast string
	var dataType postgres.DataType
	if field.SQLType == "uuid" {
		cast = "::text"
	}

	selectPath := qualifyColumn(field.Schema.Table, field.ColumnName, cast)
	if selectDistinct {
		selectPath = fmt.Sprintf("distinct(%s)", selectPath)
	}
	if aggrFunc != aggregatefunc.Unset {
		selectPath = aggrFunc.String(selectPath)
		dataType = aggrFunc.DataType()
	}
	if filter != "" {
		selectPath = fmt.Sprintf("%s filter (where %s)", selectPath, filter)
	}
	if dataType == "" {
		dataType = field.DataType
	}
	return pgsearch.SelectQueryField{
		SelectPath:   selectPath,
		Alias:        strings.Join(strings.Fields(searchField+" "+aggrFunc.Name()), "_"),
		FieldType:    dataType,
		DerivedField: aggrFunc != aggregatefunc.Unset,
	}
}
