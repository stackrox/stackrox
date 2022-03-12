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
	VALUE
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
	Fields []*searchPkg.Field
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
	var orderBy string
	if len(orderByClauses) != 0 {
		orderBy = fmt.Sprintf("order by %s", strings.Join(orderByClauses, ", "))
	}
	if pagination.GetLimit() == 0 {
		return orderBy, nil
	}
	orderBy += fmt.Sprintf(" LIMIT %d OFFSET %d", pagination.GetLimit(), pagination.GetOffset())
	return orderBy, nil
}

func generateSelectFieldsRecursive(schema *walker.Schema, added set.StringSet, q *v1.Query, optionsMap searchPkg.OptionsMap, queryFields map[string]*walker.Field) ([]string, []*searchPkg.Field) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			// nothing to do here
		case *v1.BaseQuery_MatchFieldQuery:
			// Need to find base value.
			field, ok := optionsMap.Get(subBQ.MatchFieldQuery.GetField())
			if !ok {
				return nil, nil
			}
			if subBQ.MatchFieldQuery.Highlight && added.Add(field.FieldPath) {
				dbField := queryFields[subBQ.MatchFieldQuery.GetField()]
				if dbField == nil {
					log.Errorf("Missing field %s in table %s", subBQ.MatchFieldQuery.GetField(), schema.Table)
					return nil, nil
				}
				return []string{qualifyColumn(dbField.Schema.Table, dbField.ColumnName)}, []*searchPkg.Field{field}
			}
		case *v1.BaseQuery_MatchNoneQuery:
			// nothing to here either
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			// Need to split this
			var (
				paths  []string
				fields []*searchPkg.Field
			)
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				field, ok := optionsMap.Get(q.GetField())
				if !ok {
					return nil, nil
				}
				dbField := queryFields[q.GetField()]
				if dbField == nil {
					log.Errorf("Missing field %s in table %s", q.GetField(), schema.Table)
					return nil, nil
				}

				if q.Highlight && added.Add(field.FieldPath) {
					paths = append(paths, qualifyColumn(dbField.Schema.Table, dbField.ColumnName))
					fields = append(fields, field)
				}
			}
			return paths, fields
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		var (
			paths  []string
			fields []*searchPkg.Field
		)
		for _, cq := range sub.Conjunction.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, cq, optionsMap, queryFields)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		return paths, fields
	case *v1.Query_Disjunction:
		var (
			paths  []string
			fields []*searchPkg.Field
		)
		for _, dq := range sub.Disjunction.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, dq, optionsMap, queryFields)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		return paths, fields
	case *v1.Query_BooleanQuery:
		var (
			paths  []string
			fields []*searchPkg.Field
		)
		for _, cq := range sub.BooleanQuery.Must.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, cq, optionsMap, queryFields)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		for _, dq := range sub.BooleanQuery.MustNot.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, dq, optionsMap, queryFields)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		return paths, fields
	}
	return nil, nil
}

func generateSelectFields(schema *walker.Schema, q *v1.Query, optionsMap searchPkg.OptionsMap, selectType QueryType, queryFields map[string]*walker.Field) selectQuery {
	var sel selectQuery
	if selectType == DELETE {
		sel.Query = "delete"
		return sel
	}

	if selectType == COUNT {
		sel.Query = "select count(*)"
		return sel
	}
	added := set.NewStringSet()
	paths, fields := generateSelectFieldsRecursive(schema, added, q, optionsMap, queryFields)

	var values []string
	for _, pk := range schema.LocalPrimaryKeys() {
		values = append(values, qualifyColumn(pk.Schema.Table, pk.ColumnName))
	}
	if selectType == VALUE {
		// TODO: Tackle request of serialized values that reside in multiple tables.
		paths = append(values, qualifyColumn(schema.Table, "serialized"))
	} else {
		paths = append(values, paths...)
	}
	sel.Query = fmt.Sprintf("select %s", strings.Join(paths, ","))
	sel.Fields = fields
	return sel
}

func populatePath(q *v1.Query, optionsMap searchPkg.OptionsMap, schema *walker.Schema, selectType QueryType) (*query, error) {
	// Field can belong to multiple tables. Therefore, find all the tables reachable from starting table, that contain
	// query fields.
	dbFields := getTableFieldsForQuery(schema, q)
	tables := make([]*walker.Schema, 0, len(dbFields))
	for _, f := range dbFields {
		tables = append(tables, f.Schema)
	}
	froms, joinsMap := getJoins(schema, tables...)

	fromClause := stringutils.JoinNonEmpty(", ", froms...)
	selQuery := generateSelectFields(schema, q, optionsMap, selectType, dbFields)
	queryEntry, err := compileBaseQuery(schema, q, optionsMap, dbFields, joinsMap)
	if err != nil {
		return nil, err
	}
	pagination, err := getPaginationQuery(q.Pagination, schema, dbFields)
	if err != nil {
		return nil, err
	}
	if queryEntry == nil {
		return &query{
			Select:     selQuery,
			From:       fromClause,
			Pagination: pagination,
		}, nil
	}

	return &query{
		Select:     selQuery,
		From:       fromClause,
		Where:      queryEntry.Query,
		Pagination: pagination,
		Data:       queryEntry.Values,
	}, nil
}

func multiQueryFromQueryEntries(entries []*pgsearch.QueryEntry, separator string) *pgsearch.QueryEntry {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) == 1 {
		return entries[0]
	}
	var queryStrings []string
	var data []interface{}
	for _, entry := range entries {
		queryStrings = append(queryStrings, entry.Query)
		data = append(data, entry.Values...)
	}
	return &pgsearch.QueryEntry{
		Query:  fmt.Sprintf("(%s)", strings.Join(queryStrings, separator)),
		Values: data,
	}
}

func entriesFromQueries(
	table *walker.Schema,
	queries []*v1.Query,
	optionsMap searchPkg.OptionsMap,
	queryFields map[string]*walker.Field,
	joinMap map[string]string,
) ([]*pgsearch.QueryEntry, error) {
	var entries []*pgsearch.QueryEntry
	for _, q := range queries {
		entry, err := compileBaseQuery(table, q, optionsMap, queryFields, joinMap)
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
	reachableFields := make(map[string]*walker.Field)
	recursiveSearchForFields([]*walker.Schema{schema}, searchFields, reachableFields, set.NewStringSet())
	return reachableFields
}

func recursiveSearchForFields(schemaQ []*walker.Schema, searchFields set.StringSet, reachableFields map[string]*walker.Field, visitedTables set.StringSet) {
	if len(schemaQ) == 0 || len(searchFields) == 0 {
		return
	}

	curr, schemaQ := schemaQ[0], schemaQ[1:]
	if !visitedTables.Add(curr.Table) {
		return
	}

	for _, f := range curr.Fields {
		field := f
		if searchFields.Remove(f.Search.FieldName) {
			reachableFields[f.Search.FieldName] = &field
		}
	}

	if len(searchFields) == 0 {
		return
	}
	if len(curr.Parents) == 0 && len(curr.Children) == 0 {
		return
	}

	// We want to traverse shortest length from current schema to find the tables containing the getDBFieldsForSearchFields fields.
	// Therefore, perform BFS.
	schemaQ = append(schemaQ, curr.Parents...)
	schemaQ = append(schemaQ, curr.Children...)
	recursiveSearchForFields(schemaQ, searchFields, reachableFields, visitedTables)
}

func withJoinClause(queryEntry *pgsearch.QueryEntry, dbField *walker.Field, joinMap map[string]string) {
	if queryEntry == nil {
		return
	}
	queryEntry.Query = fmt.Sprintf("(%s)", stringutils.JoinNonEmpty(" and ", queryEntry.Query, joinMap[dbField.Schema.Table]))
}

func compileBaseQuery(
	schema *walker.Schema,
	q *v1.Query,
	optionsMap searchPkg.OptionsMap,
	queryFields map[string]*walker.Field,
	joinMap map[string]string,
) (*pgsearch.QueryEntry, error) {

	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			// TODO: Tackle selection of children.
			return &pgsearch.QueryEntry{
				Query:  fmt.Sprintf("%s.id = ANY($$::text[])", schema.Table),
				Values: []interface{}{subBQ.DocIdQuery.GetIds()},
			}, nil
		case *v1.BaseQuery_MatchFieldQuery:
			qe, err := pgsearch.MatchFieldQueryFromField(
				queryFields[subBQ.MatchFieldQuery.GetField()],
				subBQ.MatchFieldQuery.GetValue(), optionsMap,
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
				qe, err := pgsearch.MatchFieldQueryFromField(queryFields[q.GetField()], q.GetValue(), optionsMap)
				if err != nil {
					return nil, err
				}
				if qe == nil {
					continue
				}

				withJoinClause(qe, queryFields[q.GetField()], joinMap)
				entries = append(entries, qe)
			}
			return multiQueryFromQueryEntries(entries, " and "), nil
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		entries, err := entriesFromQueries(schema, sub.Conjunction.Queries, optionsMap, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		return multiQueryFromQueryEntries(entries, " and "), nil
	case *v1.Query_Disjunction:
		entries, err := entriesFromQueries(schema, sub.Disjunction.Queries, optionsMap, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		return multiQueryFromQueryEntries(entries, " or "), nil
	case *v1.Query_BooleanQuery:
		entries, err := entriesFromQueries(schema, sub.BooleanQuery.Must.Queries, optionsMap, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		cqe := multiQueryFromQueryEntries(entries, " and ")
		if cqe == nil {
			cqe = pgsearch.NewTrueQuery()
		}

		entries, err = entriesFromQueries(schema, sub.BooleanQuery.MustNot.Queries, optionsMap, queryFields, joinMap)
		if err != nil {
			return nil, err
		}
		dqe := multiQueryFromQueryEntries(entries, " or ")
		if dqe == nil {
			dqe = pgsearch.NewFalseQuery()
		}
		return &pgsearch.QueryEntry{
			Query:  fmt.Sprintf("(%s and not (%s))", cqe.Query, dqe.Query),
			Values: append(cqe.Values, dqe.Values...),
		}, nil
	}
	return nil, nil
}

func valueFromStringPtrInterface(value interface{}) string {
	return *(value.(*string))
}

// RunSearchRequest executes a request again the database
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
	schema := mapping.GetTableFromCategory(category)
	query, err := populatePath(q, optionsMap, schema, GET)
	if err != nil {
		return nil, err
	}

	queryStr := query.String()
	rows, err := db.Query(context.Background(), replaceVars(queryStr), query.Data...)
	if err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
		return nil, err
	}
	defer rows.Close()

	var searchResults []searchPkg.Result

	numPrimaryKeys := len(schema.LocalPrimaryKeys())
	// only support fields for now
	highlightedResults := make([]interface{}, len(query.Select.Fields)+numPrimaryKeys)
	for i := range highlightedResults {
		highlightedResults[i] = pointers.String("")
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
			ID: strings.Join(idParts, "+"), // TODO: figure out what separator to use
		}
		if len(query.Select.Fields) > 0 {
			result.Matches = make(map[string][]string)
			for i, field := range query.Select.Fields {
				result.Matches[field.FieldPath] = []string{valueFromStringPtrInterface(highlightedResults[i+numPrimaryKeys])}
			}
		}
		searchResults = append(searchResults, result)
	}
	return searchResults, nil
}

// RunCountRequest executes a request for just the count against the database
func RunCountRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) (int, error) {
	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), COUNT)
	if err != nil {
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
