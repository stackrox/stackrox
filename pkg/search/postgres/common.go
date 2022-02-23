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
)

var (
	log = logging.LoggerForModule()
)

// QueryType describe what type of query to execute
type QueryType int

// These are the currently supported query types
const (
	GET    QueryType = 0
	COUNT  QueryType = 1
	VALUE  QueryType = 2
	DELETE QueryType = 3
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
	query := q.Select.Query + " " + q.From
	if q.Where != "" {
		query += " where " + q.Where
	}
	if q.Pagination != "" {
		query += " " + q.Pagination
	}
	return query
}

func getPaginationQuery(pagination *v1.QueryPagination, schema *walker.Schema, optionsMap searchPkg.OptionsMap) (string, error) {
	if pagination == nil {
		return "", nil
	}

	var orderByClauses []string
	for _, so := range pagination.GetSortOptions() {
		direction := "asc"
		if so.GetReversed() {
			direction = "desc"
		}
		dbField := schema.FieldsBySearchLabel()[so.GetField()]
		if dbField == nil {
			return "", errors.Errorf("field %s does not exist in table %s", so.GetField(), schema.Table)
		}
		orderByClauses = append(orderByClauses, dbField.ColumnName+" "+direction)
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

func generateSelectFieldsRecursive(schema *walker.Schema, added set.StringSet, q *v1.Query, optionsMap searchPkg.OptionsMap) ([]string, []*searchPkg.Field) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			// nothing to do here
		case *v1.BaseQuery_MatchFieldQuery:
			// Need to find base value
			field, ok := optionsMap.Get(subBQ.MatchFieldQuery.GetField())
			if !ok {
				return nil, nil
			}
			if subBQ.MatchFieldQuery.Highlight && added.Add(field.FieldPath) {
				dbField := schema.FieldsBySearchLabel()[subBQ.MatchFieldQuery.GetField()]
				if dbField == nil {
					log.Errorf("Missing field %s in table %s", subBQ.MatchFieldQuery.GetField(), schema.Table)
					return nil, nil
				}
				return []string{dbField.ColumnName}, []*searchPkg.Field{field}
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
				dbField := schema.FieldsBySearchLabel()[q.GetField()]
				if dbField == nil {
					log.Errorf("Missing field %s in table %s", q.GetField(), schema.Table)
					return nil, nil
				}

				if q.Highlight && added.Add(field.FieldPath) {
					paths = append(paths, dbField.ColumnName)
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
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, cq, optionsMap)
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
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, dq, optionsMap)
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
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, cq, optionsMap)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		for _, dq := range sub.BooleanQuery.MustNot.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(schema, added, dq, optionsMap)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		return paths, fields
	}
	return nil, nil
}

func generateSelectFields(schema *walker.Schema, q *v1.Query, optionsMap searchPkg.OptionsMap, selectType QueryType) selectQuery {
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
	paths, fields := generateSelectFieldsRecursive(schema, added, q, optionsMap)

	var values []string
	for _, pk := range schema.LocalPrimaryKeys() {
		values = append(values, pk.ColumnName)
	}
	if len(values) > 1 {
		values = []string{
			fmt.Sprintf("distinct(%s)", strings.Join(values, ", ")),
		}
	}
	if selectType == VALUE {
		paths = append(values, "serialized")
	} else {
		paths = append(values, paths...)
	}
	sel.Query = fmt.Sprintf("select %s", strings.Join(paths, ","))
	sel.Fields = fields
	return sel
}

func populatePath(q *v1.Query, optionsMap searchPkg.OptionsMap, schema *walker.Schema, selectType QueryType) (*query, error) {
	fromClause := fmt.Sprintf("from %s", schema.Table)

	selQuery := generateSelectFields(schema, q, optionsMap, selectType)
	queryEntry, err := compileBaseQuery(schema, q, optionsMap)
	if err != nil {
		return nil, err
	}
	pagination, err := getPaginationQuery(q.Pagination, schema, optionsMap)
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

func entriesFromQueries(table *walker.Schema, queries []*v1.Query, optionsMap searchPkg.OptionsMap) ([]*pgsearch.QueryEntry, error) {
	var entries []*pgsearch.QueryEntry
	for _, q := range queries {
		entry, err := compileBaseQuery(table, q, optionsMap)
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

func compileBaseQuery(schema *walker.Schema, q *v1.Query, optionsMap searchPkg.OptionsMap) (*pgsearch.QueryEntry, error) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			return &pgsearch.QueryEntry{
				Query:  fmt.Sprintf("%s.id = ANY($$::text[])", schema.Table),
				Values: []interface{}{subBQ.DocIdQuery.GetIds()},
			}, nil
		case *v1.BaseQuery_MatchFieldQuery:
			return pgsearch.MatchFieldQuery(schema, subBQ.MatchFieldQuery, optionsMap)
		case *v1.BaseQuery_MatchNoneQuery:
			return nil, nil
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			var entries []*pgsearch.QueryEntry
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				qe, err := pgsearch.MatchFieldQuery(schema, q, optionsMap)
				if err != nil {
					return nil, err
				}
				if qe == nil {
					continue
				}
				entries = append(entries, qe)
			}
			return multiQueryFromQueryEntries(entries, " and "), nil
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		entries, err := entriesFromQueries(schema, sub.Conjunction.Queries, optionsMap)
		if err != nil {
			return nil, err
		}
		return multiQueryFromQueryEntries(entries, " and "), nil
	case *v1.Query_Disjunction:
		entries, err := entriesFromQueries(schema, sub.Disjunction.Queries, optionsMap)
		if err != nil {
			return nil, err
		}
		return multiQueryFromQueryEntries(entries, " or "), nil
	case *v1.Query_BooleanQuery:
		entries, err := entriesFromQueries(schema, sub.BooleanQuery.Must.Queries, optionsMap)
		if err != nil {
			return nil, err
		}
		cqe := multiQueryFromQueryEntries(entries, " and ")
		if cqe == nil {
			cqe = pgsearch.NewTrueQuery()
		}

		entries, err = entriesFromQueries(schema, sub.BooleanQuery.MustNot.Queries, optionsMap)
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

	// only support fields for now
	highlightedResults := make([]interface{}, len(query.Select.Fields)+len(schema.LocalPrimaryKeys()))
	for i := range highlightedResults {
		highlightedResults[i] = pointers.String("")
	}
	for rows.Next() {
		if err := rows.Scan(highlightedResults...); err != nil {
			return nil, err
		}
		result := searchPkg.Result{
			ID: valueFromStringPtrInterface(highlightedResults[0]),
		}
		if len(query.Select.Fields) > 0 {
			result.Matches = make(map[string][]string)
			for i, field := range query.Select.Fields {
				result.Matches[field.FieldPath] = []string{valueFromStringPtrInterface(highlightedResults[i+1])}
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
