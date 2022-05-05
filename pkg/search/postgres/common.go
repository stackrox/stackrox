package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/walker"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()
)

// QueryType describe what type of query to execute
type QueryType int

// These are the currently supported query types
const (
	GET QueryType = iota
	COUNT
	DELETE
)

func replaceVars(s string) string {
	varNum := 1
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '$' && s[i+1] == '$' {
			s = s[:i+1] + fmt.Sprintf("%d", varNum) + s[i+2:]
			varNum++
		}
	}
	return s
}

type selectQuery struct {
	Query  string
	Fields []pgsearch.SelectQueryField
}

type query struct {
	Select     selectQuery
	From       string
	Where      string
	Pagination string
	Data       []interface{}
}

func (q *query) String() string {
	query := q.Select.Query + " from " + q.From
	if q.Where != "" {
		query += " where " + q.Where
	}
	if q.Pagination != "" {
		query += " " + q.Pagination
	}
	return query
}

func qualifyColumn(table, column string) string {
	return table + "." + column
}

func getPaginationQuery(pagination *v1.QueryPagination, schema *walker.Schema, queryFields map[string]*walker.Field) (string, error) {
	if pagination == nil {
		return "", nil
	}

	var orderByClauses []string
	for _, so := range pagination.GetSortOptions() {
		direction := "asc"
		if so.GetReversed() {
			direction = "desc"
		}
		dbField := queryFields[so.GetField()]
		if dbField == nil {
			return "", errors.Errorf("field %s does not exist in table %s or connected tables", so.GetField(), schema.Table)
		}
		orderByClauses = append(
			orderByClauses,
			fmt.Sprintf("%s %s", qualifyColumn(dbField.Schema.Table, dbField.ColumnName), direction),
		)
	}
	var paginationClause strings.Builder
	if len(orderByClauses) > 0 {
		paginationClause.WriteString(fmt.Sprintf("order by %s", strings.Join(orderByClauses, ", ")))
	}
	if pagination.GetLimit() > 0 {
		paginationClause.WriteString(fmt.Sprintf(" LIMIT %d", pagination.GetLimit()))
	}
	if pagination.GetOffset() > 0 {
		paginationClause.WriteString(fmt.Sprintf(" OFFSET %d", pagination.GetOffset()))
	}
	return paginationClause.String(), nil
}

func generateSelectFields(entry *pgsearch.QueryEntry, primaryKeys []walker.Field, selectType QueryType) selectQuery {
	var sel selectQuery
	if selectType == DELETE {
		sel.Query = "delete"
		return sel
	}

	if selectType == COUNT {
		sel.Query = "select count(*)"
		return sel
	}
	var pathsInSelectClause []string
	// Always select the primary keys first.
	for _, pk := range primaryKeys {
		pathsInSelectClause = append(pathsInSelectClause, qualifyColumn(pk.Schema.Table, pk.ColumnName))
	}

	if entry != nil {
		for _, selectedField := range entry.SelectedFields {
			pathsInSelectClause = append(pathsInSelectClause, selectedField.SelectPath)
		}
		sel.Fields = entry.SelectedFields
	}

	sel.Query = fmt.Sprintf("select %s", strings.Join(pathsInSelectClause, ","))
	return sel
}

func standardizeQueryAndPopulatePath(q *v1.Query, schema *walker.Schema, selectType QueryType) (*query, error) {
	standardizeFieldNamesInQuery(q)
	// Field can belong to multiple tables. Therefore, find all the tables reachable from starting table, that contain
	// query fields.
	// TODO(viswa): both getTableFieldsForQuery and getJoins do the same BFS, see if there's a way to simplify and do
	// it in one go.
	dbFields := getTableFieldsForQuery(schema, q)
	tables := make([]*walker.Schema, 0, len(dbFields))
	for _, f := range dbFields {
		tables = append(tables, f.Schema)
	}
	froms, joinsMap := getJoins(schema, tables...)

	queryEntry, err := compileQueryToPostgres(schema, q, dbFields, joinsMap)
	if err != nil {
		return nil, err
	}
	// If a non-empty query was passed, but we couldn't find a query, that means that the query is invalid
	// for this category. (For example, searching secrets by "Policy:"). In this case, we return a query that matches nothing.
	// This behaviour is helpful, for example, in Global Search, where a query that is invalid for a
	// certain category will just return no elements of that category.
	if q.GetQuery() != nil && queryEntry == nil {
		return nil, nil
	}

	fromClause := stringutils.JoinNonEmpty(", ", froms...)
	selQuery := generateSelectFields(queryEntry, schema.PrimaryKeys(), selectType)
	pagination, err := getPaginationQuery(q.GetPagination(), schema, dbFields)
	if err != nil {
		return nil, err
	}

	query := &query{
		Select:     selQuery,
		From:       fromClause,
		Pagination: pagination,
	}
	if queryEntry != nil {
		query.Where = queryEntry.Where.Query
		query.Data = queryEntry.Where.Values
	}
	return query, nil
}

func combineQueryEntries(entries []*pgsearch.QueryEntry, separator string) *pgsearch.QueryEntry {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) == 1 {
		return entries[0]
	}
	var queryStrings []string
	seenSelectFields := set.NewStringSet()
	newQE := &pgsearch.QueryEntry{}
	for _, entry := range entries {
		queryStrings = append(queryStrings, entry.Where.Query)
		newQE.Where.Values = append(newQE.Where.Values, entry.Where.Values...)
		for _, selectedField := range entry.SelectedFields {
			if seenSelectFields.Add(selectedField.SelectPath) {
				newQE.SelectedFields = append(newQE.SelectedFields, selectedField)
			}
		}
	}
	newQE.Where.Query = fmt.Sprintf("(%s)", strings.Join(queryStrings, separator))
	return newQE
}

func entriesFromQueries(
	table *walker.Schema,
	queries []*v1.Query,
	queryFields map[string]*walker.Field,
	joinMap map[string]string,
) ([]*pgsearch.QueryEntry, error) {
	var entries []*pgsearch.QueryEntry
	for _, q := range queries {
		entry, err := compileQueryToPostgres(table, q, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func collectFields(q *v1.Query) set.StringSet {
	var queries []*v1.Query
	collectedFields := set.NewStringSet()
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery, *v1.BaseQuery_MatchNoneQuery:
			// nothing to do
		case *v1.BaseQuery_MatchFieldQuery:
			collectedFields.Add(subBQ.MatchFieldQuery.GetField())
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				collectedFields.Add(q.GetField())
			}
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		queries = append(queries, sub.Conjunction.Queries...)
	case *v1.Query_Disjunction:
		queries = append(queries, sub.Disjunction.Queries...)
	case *v1.Query_BooleanQuery:
		queries = append(queries, sub.BooleanQuery.Must.Queries...)
		queries = append(queries, sub.BooleanQuery.MustNot.Queries...)
	}

	for _, query := range queries {
		collectedFields.AddAll(collectFields(query).AsSlice()...)
	}
	for _, sortOption := range q.GetPagination().GetSortOptions() {
		collectedFields.Add(sortOption.GetField())
	}
	return collectedFields
}

func getTableFieldsForQuery(schema *walker.Schema, q *v1.Query) map[string]*walker.Field {
	return getDBFieldsForSearchFields(schema, collectFields(q))
}

func getDBFieldsForSearchFields(schema *walker.Schema, searchFields set.StringSet) map[string]*walker.Field {
	if len(searchFields) == 0 {
		return nil
	}
	reachableFields := make(map[string]*walker.Field)

	schemaQ := []*walker.Schema{schema}

	visitedTables := set.NewStringSet()

	// We want to traverse shortest length from current schema to find the tables containing the getDBFieldsForSearchFields fields.
	// Therefore, perform BFS.
bfsLoop:
	for len(schemaQ) > 0 {
		curr := schemaQ[0]
		schemaQ = schemaQ[1:]
		if !visitedTables.Add(curr.Table) {
			continue
		}

		for _, f := range curr.Fields {
			field := f
			lowerCaseName := strings.ToLower(f.Search.FieldName)
			if searchFields.Remove(lowerCaseName) {
				reachableFields[lowerCaseName] = &field
				if len(searchFields) == 0 {
					break bfsLoop
				}
			}
		}
		for _, rel := range curr.AllRelationships() {
			schemaQ = append(schemaQ, rel.OtherSchema)
		}
	}
	return reachableFields
}

func withJoinClause(queryEntry *pgsearch.QueryEntry, dbField *walker.Field, joinMap map[string]string) {
	if queryEntry == nil {
		return
	}
	queryEntry.Where.Query = fmt.Sprintf("(%s)", stringutils.JoinNonEmpty(" and ", queryEntry.Where.Query, joinMap[dbField.Schema.Table]))
}

func compileQueryToPostgres(
	schema *walker.Schema,
	q *v1.Query,
	queryFields map[string]*walker.Field,
	joinMap map[string]string,
) (*pgsearch.QueryEntry, error) {

	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			return &pgsearch.QueryEntry{Where: pgsearch.WhereClause{
				Query:  fmt.Sprintf("%s.id = ANY($$::text[])", schema.Table),
				Values: []interface{}{subBQ.DocIdQuery.GetIds()},
			}}, nil
		case *v1.BaseQuery_MatchFieldQuery:
			qe, err := pgsearch.MatchFieldQuery(
				queryFields[subBQ.MatchFieldQuery.GetField()],
				subBQ.MatchFieldQuery.GetValue(),
				subBQ.MatchFieldQuery.GetHighlight(),
			)
			if err != nil {
				return nil, err
			}
			withJoinClause(qe, queryFields[subBQ.MatchFieldQuery.GetField()], joinMap)
			return qe, nil
		case *v1.BaseQuery_MatchNoneQuery:
			return nil, nil
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			var entries []*pgsearch.QueryEntry
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				qe, err := pgsearch.MatchFieldQuery(queryFields[q.GetField()], q.GetValue(), q.GetHighlight())
				if err != nil {
					return nil, err
				}
				if qe == nil {
					continue
				}

				withJoinClause(qe, queryFields[q.GetField()], joinMap)
				entries = append(entries, qe)
			}
			return combineQueryEntries(entries, " and "), nil
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		entries, err := entriesFromQueries(schema, sub.Conjunction.Queries, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		return combineQueryEntries(entries, " and "), nil
	case *v1.Query_Disjunction:
		entries, err := entriesFromQueries(schema, sub.Disjunction.Queries, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		return combineQueryEntries(entries, " or "), nil
	case *v1.Query_BooleanQuery:
		entries, err := entriesFromQueries(schema, sub.BooleanQuery.Must.Queries, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		cqe := combineQueryEntries(entries, " and ")
		if cqe == nil {
			cqe = pgsearch.NewTrueQuery()
		}

		entries, err = entriesFromQueries(schema, sub.BooleanQuery.MustNot.Queries, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		dqe := combineQueryEntries(entries, " or ")
		if dqe == nil {
			dqe = pgsearch.NewFalseQuery()
		}
		return &pgsearch.QueryEntry{
			Where: pgsearch.WhereClause{
				Query:  fmt.Sprintf("(%s and not (%s))", cqe.Where.Query, dqe.Where.Query),
				Values: append(cqe.Where.Values, dqe.Where.Values...),
			},
		}, nil
	}
	return nil, nil
}

func valueFromStringPtrInterface(value interface{}) string {
	return *(value.(*string))
}

func standardizeFieldNamesInQuery(q *v1.Query) {
	// Lowercase all field names in the query, for standardization.
	// There are certain places where we operate on the query fields directly as strings,
	// without access to the options map.
	// TODO: this could be made cleaner by refactoring the v1.Query object to directly have FieldLabels.
	searchPkg.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		switch bq := bq.Query.(type) {
		case *v1.BaseQuery_MatchFieldQuery:
			bq.MatchFieldQuery.Field = strings.ToLower(bq.MatchFieldQuery.Field)
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			for _, q := range bq.MatchLinkedFieldsQuery.Query {
				q.Field = strings.ToLower(q.Field)
			}
		}
	})

	for _, sortOption := range q.GetPagination().GetSortOptions() {
		sortOption.Field = strings.ToLower(sortOption.Field)
	}
}

// RunSearchRequest executes a request again the database
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool) (searchResults []searchPkg.Result, err error) {
	var query *query
	// Add this to be safe and convert panics to errors,
	// since we do a lot of casting and other operations that could potentially panic in this code.
	// Panics are expected ONLY in the event of a programming error, all foreseeable errors are handled
	// the usual way.
	defer func() {
		if r := recover(); r != nil {
			if query != nil {
				log.Errorf("Query issue: %s %+v: %v", query, query.Data, r)
			} else {
				log.Errorf("Unexpected error running search request: %v", r)
			}
			debug.PrintStack()
			err = fmt.Errorf("unexpected error running search request: %v", r)
		}
	}()

	schema := mapping.GetTableFromCategory(category)
	query, err = standardizeQueryAndPopulatePath(q, schema, GET)
	if err != nil {
		return nil, err
	}
	// A nil-query implies no results.
	if query == nil {
		return nil, nil
	}

	queryStr := query.String()
	rows, err := db.Query(context.Background(), replaceVars(queryStr), query.Data...)
	if err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
		return nil, err
	}
	defer rows.Close()
	log.Debugf("SEARCH: ran query %s; data %+v", queryStr, query.Data)

	numPrimaryKeys := len(schema.PrimaryKeys())
	highlightedResults := make([]interface{}, len(query.Select.Fields)+numPrimaryKeys)

	// Assumes that ids are strings.
	for i := 0; i < numPrimaryKeys; i++ {
		highlightedResults[i] = pointers.String("")
	}
	for i, field := range query.Select.Fields {
		highlightedResults[i+numPrimaryKeys] = mustAllocForDataType(field.FieldType)
	}
	for rows.Next() {
		if err := rows.Scan(highlightedResults...); err != nil {
			return nil, err
		}
		idParts := make([]string, 0, numPrimaryKeys)
		for i := 0; i < numPrimaryKeys; i++ {
			idParts = append(idParts, valueFromStringPtrInterface(highlightedResults[i]))
		}
		result := searchPkg.Result{
			ID: strings.Join(idParts, IDSeparator), // TODO: figure out what separator to use
		}
		if len(query.Select.Fields) > 0 {
			result.Matches = make(map[string][]string)
			for i, field := range query.Select.Fields {
				returnedValue := highlightedResults[i+numPrimaryKeys]
				if field.PostTransform != nil {
					returnedValue = field.PostTransform(returnedValue)
				}
				if matches := mustPrintForDataType(field.FieldType, returnedValue); len(matches) > 0 {
					result.Matches[field.FieldPath] = matches
				}
			}
		}
		searchResults = append(searchResults, result)
	}
	return searchResults, nil
}

// RunCountRequest executes a request for just the count against the database
func RunCountRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool) (int, error) {
	schema := mapping.GetTableFromCategory(category)
	return RunCountRequestForSchema(schema, q, db)
}

// RunCountRequestForSchema executes a request for just the count against the database
func RunCountRequestForSchema(schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) (int, error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, COUNT)
	if err != nil || query == nil {
		return 0, err
	}

	queryStr := query.String()
	var count int
	row := db.QueryRow(context.Background(), replaceVars(queryStr), query.Data...)
	if err := row.Scan(&count); err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
		return 0, err
	}
	return count, nil
}
