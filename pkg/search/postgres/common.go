package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/walker"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/ternary"
)

var (
	log = logging.LoggerForModule()
)

// QueryType describe what type of query to execute
//go:generate stringer -type=QueryType
type QueryType int

// These are the currently supported query types
const (
	SEARCH QueryType = iota
	GET
	COUNT
	DELETE
)

func replaceVars(s string) string {
	varNum := 1
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '$' && s[i+1] == '$' {
			varStr := strconv.Itoa(varNum)
			s = s[:i+1] + varStr + s[i+2:]
			i += len(varStr)
			varNum++
		}
	}
	return s
}

type innerJoin struct {
	leftTable       string
	rightTable      string
	columnNamePairs []walker.ColumnNamePair
}

type query struct {
	Schema            *walker.Schema
	QueryType         QueryType
	SelectedFields    []pgsearch.SelectQueryField
	From              string
	Where             string
	Pagination        parsedPaginationQuery
	InnerJoins        []innerJoin
	Data              []interface{}
	GroupByPrimaryKey bool
}

// ExtraSelectedFieldPaths includes extra fields to add to the select clause.
// We don't care about actually reading the values of these fields, they're
// there to make SQL happy.
func (q *query) ExtraSelectedFieldPaths() []pgsearch.SelectQueryField {
	if !q.DistinctAppliedOnPrimaryKeySelect() {
		return nil
	}
	var out []pgsearch.SelectQueryField
	for _, orderByEntry := range q.Pagination.OrderBys {
		var alreadyExists bool
		for _, selectedField := range q.SelectedFields {
			if selectedField.SelectPath == orderByEntry.Field.SelectPath {
				alreadyExists = true
			}
		}
		if !alreadyExists {
			out = append(out, orderByEntry.Field)
		}
	}
	return out
}

func (q *query) getPortionBeforeFromClause() string {
	switch q.QueryType {
	case DELETE:
		return "delete"
	case COUNT:
		return "select count(*)"
	case GET:
		return fmt.Sprintf("select %q.serialized", q.From)
	case SEARCH:
		var primaryKeyPaths []string
		// Always select the primary keys first.
		for _, pk := range q.Schema.PrimaryKeys() {
			primaryKeyPaths = append(primaryKeyPaths, qualifyColumn(pk.Schema.Table, pk.ColumnName))
		}
		primaryKeyPortion := strings.Join(primaryKeyPaths, ", ")

		if q.DistinctAppliedOnPrimaryKeySelect() {
			primaryKeyPortion = fmt.Sprintf("distinct(%s)", primaryKeyPortion)
		}
		var remainingFieldPaths []string
		for _, selectedField := range q.SelectedFields {
			remainingFieldPaths = append(remainingFieldPaths, selectedField.SelectPath)
		}
		for _, field := range q.ExtraSelectedFieldPaths() {
			remainingFieldPaths = append(remainingFieldPaths, field.SelectPath)
		}
		remainingPortion := strings.Join(remainingFieldPaths, ", ")
		return "select " + stringutils.JoinNonEmpty(", ", primaryKeyPortion, remainingPortion)
	}
	panic(fmt.Sprintf("unhandled query type %s", q.QueryType))
}

func (q *query) DistinctAppliedOnPrimaryKeySelect() bool {
	// If this involves multiple tables, then we need to wrap the primary key portion in a distinct, because
	// otherwise there could be multiple rows with the same primary key in the join table.
	// TODO(viswa): we might be able to do this even more narrowly
	return len(q.InnerJoins) > 0 && !q.GroupByPrimaryKey
}

func (q *query) AsSQL() string {
	var querySB strings.Builder

	querySB.WriteString(q.getPortionBeforeFromClause())
	querySB.WriteString(" from ")
	querySB.WriteString(q.From)
	for _, innerJoin := range q.InnerJoins {
		querySB.WriteString(" inner join ")
		querySB.WriteString(innerJoin.rightTable)
		querySB.WriteString(" on")
		for i, columnNamePair := range innerJoin.columnNamePairs {
			if i > 0 {
				querySB.WriteString(" and")
			}
			querySB.WriteString(fmt.Sprintf(" %s.%s = %s.%s", innerJoin.leftTable, columnNamePair.ColumnNameInThisSchema, innerJoin.rightTable, columnNamePair.ColumnNameInOtherSchema))
		}
	}
	if q.Where != "" {
		querySB.WriteString(" where ")
		querySB.WriteString(replaceVars(q.Where))
	}
	if q.GroupByPrimaryKey {
		primaryKeys := q.Schema.PrimaryKeys()
		primaryKeyPaths := make([]string, 0, len(primaryKeys))
		for _, pk := range primaryKeys {
			primaryKeyPaths = append(primaryKeyPaths, qualifyColumn(pk.Schema.Table, pk.ColumnName))
		}
		querySB.WriteString(" group by ")
		querySB.WriteString(strings.Join(primaryKeyPaths, ", "))
	}
	if paginationSQL := q.Pagination.AsSQL(); paginationSQL != "" {
		querySB.WriteString(" ")
		querySB.WriteString(paginationSQL)
	}
	return querySB.String()
}

func qualifyColumn(table, column string) string {
	return table + "." + column
}

type parsedPaginationQuery struct {
	OrderBys []orderByEntry
	Limit    int
	Offset   int
}

type orderByEntry struct {
	Field       pgsearch.SelectQueryField
	Descending  bool
	SearchAfter string
}

func (p *parsedPaginationQuery) AsSQL() string {
	var paginationSB strings.Builder
	if len(p.OrderBys) > 0 {
		orderByClauses := make([]string, 0, len(p.OrderBys))
		for _, entry := range p.OrderBys {
			orderByClauses = append(orderByClauses, fmt.Sprintf("%s %s", entry.Field.SelectPath, ternary.String(entry.Descending, "desc", "asc")))
		}
		paginationSB.WriteString(fmt.Sprintf("order by %s", strings.Join(orderByClauses, ", ")))
	}
	if p.Limit > 0 {
		paginationSB.WriteString(fmt.Sprintf(" LIMIT %d", p.Limit))
	}
	if p.Offset > 0 {
		paginationSB.WriteString(fmt.Sprintf(" OFFSET %d", p.Offset))
	}
	return paginationSB.String()
}

func populatePagination(querySoFar *query, pagination *v1.QueryPagination, schema *walker.Schema, queryFields map[string]searchFieldMetadata) error {
	if pagination == nil {
		return nil
	}

	for idx, so := range pagination.GetSortOptions() {
		if idx != 0 && so.GetSearchAfter() != "" {
			return errors.New("search after for pagination must be defined for only the first sort option")
		}
		if so.GetField() == searchPkg.DocID.String() {
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field: pgsearch.SelectQueryField{
					SelectPath: qualifyColumn(schema.Table, schema.ID().ColumnName),
					FieldType:  walker.String,
				},
				Descending:  so.GetReversed(),
				SearchAfter: so.GetSearchAfter(),
			})
			continue
		}
		fieldMetadata := queryFields[so.GetField()]
		dbField := fieldMetadata.baseField
		if dbField == nil {
			return errors.Errorf("field %s does not exist in table %s or connected tables", so.GetField(), schema.Table)
		}
		if fieldMetadata.derivedMetadata == nil {
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field: pgsearch.SelectQueryField{
					SelectPath: qualifyColumn(dbField.Schema.Table, dbField.ColumnName),
					FieldType:  dbField.DataType,
				},
				Descending:  so.GetReversed(),
				SearchAfter: so.GetSearchAfter(),
			})
		} else {
			switch fieldMetadata.derivedMetadata.DerivationType {
			case searchPkg.CountDerivationType:
				querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
					Field: pgsearch.SelectQueryField{
						SelectPath: fmt.Sprintf("count(%s)", qualifyColumn(dbField.Schema.Table, dbField.ColumnName)),
						FieldType:  dbField.DataType,
					},
					Descending: so.GetReversed(),
				})
				// If we're ordering by a count, we will need to group by the primary key.
				querySoFar.GroupByPrimaryKey = true
			}
		}
	}
	querySoFar.Pagination.Limit = int(pagination.GetLimit())
	querySoFar.Pagination.Offset = int(pagination.GetOffset())
	return nil
}

func applyPaginationForSearchAfter(query *query) error {
	pagination := query.Pagination
	if len(pagination.OrderBys) == 0 {
		return nil
	}
	firstOrderBy := pagination.OrderBys[0]
	if firstOrderBy.SearchAfter == "" {
		return nil
	}
	if query.Where != "" {
		query.Where += " and "
	}
	operand := ">"
	if firstOrderBy.Descending {
		operand = "<"
	}
	query.Where += fmt.Sprintf("%s %s $$", firstOrderBy.Field.SelectPath, operand)
	query.Data = append(query.Data, firstOrderBy.SearchAfter)
	return nil
}

func standardizeQueryAndPopulatePath(q *v1.Query, schema *walker.Schema, queryType QueryType) (*query, error) {
	standardizeFieldNamesInQuery(q)
	innerJoins, dbFields := getJoinsAndFields(schema, q)

	queryEntry, err := compileQueryToPostgres(schema, q, dbFields)
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

	query := &query{
		Schema:     schema,
		QueryType:  queryType,
		InnerJoins: innerJoins,
		From:       schema.Table,
	}
	if queryEntry != nil {
		query.Where = queryEntry.Where.Query
		query.Data = queryEntry.Where.Values
		query.SelectedFields = queryEntry.SelectedFields
	}
	if err := populatePagination(query, q.GetPagination(), schema, dbFields); err != nil {
		return nil, err
	}
	if err := applyPaginationForSearchAfter(query); err != nil {
		return nil, err
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
	queryFields map[string]searchFieldMetadata,
) ([]*pgsearch.QueryEntry, error) {
	var entries []*pgsearch.QueryEntry
	for _, q := range queries {
		entry, err := compileQueryToPostgres(table, q, queryFields)
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

func compileQueryToPostgres(schema *walker.Schema, q *v1.Query, queryFields map[string]searchFieldMetadata) (*pgsearch.QueryEntry, error) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			return &pgsearch.QueryEntry{Where: pgsearch.WhereClause{
				Query:  fmt.Sprintf("%s.%s = ANY($$::text[])", schema.Table, schema.ID().ColumnName),
				Values: []interface{}{subBQ.DocIdQuery.GetIds()},
			}}, nil
		case *v1.BaseQuery_MatchFieldQuery:
			qe, err := pgsearch.MatchFieldQuery(
				queryFields[subBQ.MatchFieldQuery.GetField()].baseField,
				subBQ.MatchFieldQuery.GetValue(),
				subBQ.MatchFieldQuery.GetHighlight(),
			)
			if err != nil {
				return nil, err
			}
			return qe, nil
		case *v1.BaseQuery_MatchNoneQuery:
			return pgsearch.NewFalseQuery(), nil
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			var entries []*pgsearch.QueryEntry
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				qe, err := pgsearch.MatchFieldQuery(queryFields[q.GetField()].baseField, q.GetValue(), q.GetHighlight())
				if err != nil {
					return nil, err
				}
				if qe == nil {
					continue
				}

				entries = append(entries, qe)
			}
			return combineQueryEntries(entries, " and "), nil
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		entries, err := entriesFromQueries(schema, sub.Conjunction.Queries, queryFields)
		if err != nil {
			return nil, err
		}
		return combineQueryEntries(entries, " and "), nil
	case *v1.Query_Disjunction:
		entries, err := entriesFromQueries(schema, sub.Disjunction.Queries, queryFields)
		if err != nil {
			return nil, err
		}
		return combineQueryEntries(entries, " or "), nil
	case *v1.Query_BooleanQuery:
		entries, err := entriesFromQueries(schema, sub.BooleanQuery.Must.Queries, queryFields)
		if err != nil {
			return nil, err
		}
		cqe := combineQueryEntries(entries, " and ")
		if cqe == nil {
			cqe = pgsearch.NewTrueQuery()
		}

		entries, err = entriesFromQueries(schema, sub.BooleanQuery.MustNot.Queries, queryFields)
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

// RunSearchRequest executes a request against the database for given category
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool) ([]searchPkg.Result, error) {
	schema := mapping.GetTableFromCategory(category)
	return RunSearchRequestForSchema(schema, q, db)
}

// RunSearchRequestForSchema executes a request against the database for given schema
func RunSearchRequestForSchema(schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) (searchResults []searchPkg.Result, err error) {
	var query *query
	// Add this to be safe and convert panics to errors,
	// since we do a lot of casting and other operations that could potentially panic in this code.
	// Panics are expected ONLY in the event of a programming error, all foreseeable errors are handled
	// the usual way.
	defer func() {
		if r := recover(); r != nil {
			if query != nil {
				log.Errorf("Query issue: %s %+v: %v", query.AsSQL(), query.Data, r)
			} else {
				log.Errorf("Unexpected error running search request: %v", r)
			}
			debug.PrintStack()
			err = fmt.Errorf("unexpected error running search request: %v", r)
		}
	}()

	query, err = standardizeQueryAndPopulatePath(q, schema, SEARCH)
	if err != nil {
		return nil, err
	}
	// A nil-query implies no results.
	if query == nil {
		return nil, nil
	}

	queryStr := query.AsSQL()
	rows, err := db.Query(context.Background(), queryStr, query.Data...)
	if err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", queryStr, query.Data, err)
		return nil, err
	}
	defer rows.Close()
	log.Debugf("SEARCH: ran query %s; data %+v", queryStr, query.Data)

	// Assumes that ids are strings.
	numPrimaryKeys := len(schema.PrimaryKeys())
	extraSelectedFields := query.ExtraSelectedFieldPaths()
	var numFieldsForPrimaryKey int
	if query.DistinctAppliedOnPrimaryKeySelect() {
		numFieldsForPrimaryKey = 1
	} else {
		numFieldsForPrimaryKey = numPrimaryKeys
	}
	primaryKeysComposite := numPrimaryKeys > 1 && numFieldsForPrimaryKey == 1

	bufferToScanRowInto := make([]interface{}, numFieldsForPrimaryKey+len(query.SelectedFields)+len(extraSelectedFields))
	if primaryKeysComposite {
		var outputSlice []interface{}
		bufferToScanRowInto[0] = &outputSlice
	} else {
		for i := 0; i < numPrimaryKeys; i++ {
			bufferToScanRowInto[i] = pointers.String("")
		}
	}
	for i, field := range query.SelectedFields {
		bufferToScanRowInto[i+numFieldsForPrimaryKey] = mustAllocForDataType(field.FieldType)
	}
	for i, field := range extraSelectedFields {
		bufferToScanRowInto[i+len(query.SelectedFields)+numFieldsForPrimaryKey] = mustAllocForDataType(field.FieldType)
	}
	for rows.Next() {
		if err := rows.Scan(bufferToScanRowInto...); err != nil {
			return nil, err
		}

		idParts := make([]string, 0, numPrimaryKeys)
		if primaryKeysComposite {
			for _, elem := range *bufferToScanRowInto[0].(*[]interface{}) {
				idParts = append(idParts, elem.(string))
			}
		} else {
			for i := 0; i < numPrimaryKeys; i++ {
				idParts = append(idParts, valueFromStringPtrInterface(bufferToScanRowInto[i]))
			}
		}
		result := searchPkg.Result{
			ID: strings.Join(idParts, IDSeparator), // TODO: figure out what separator to use
		}
		if len(query.SelectedFields) > 0 {
			result.Matches = make(map[string][]string)
			for i, field := range query.SelectedFields {
				returnedValue := bufferToScanRowInto[i+numFieldsForPrimaryKey]
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

	queryStr := query.AsSQL()
	var count int
	row := db.QueryRow(context.Background(), queryStr, query.Data...)
	if err := row.Scan(&count); err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", queryStr, query.Data, err)
		return 0, err
	}
	return count, nil
}

// RunGetQueryForSchema executes a request for just the search against the database
func RunGetQueryForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) ([]byte, error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, errox.InvalidArgs.New("empty query")
	}

	queryStr := query.AsSQL()
	row := db.QueryRow(ctx, queryStr, query.Data...)

	var data []byte
	err = row.Scan(&data)
	return data, err
}

// RunGetManyQueryForSchema executes a request for just the search against the database
func RunGetManyQueryForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) ([][]byte, error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, errox.InvalidArgs.New("empty query")
	}

	queryStr := query.AsSQL()
	rows, err := db.Query(ctx, queryStr, query.Data...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results [][]byte
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		results = append(results, data)
	}
	return results, nil
}

// RunDeleteRequestForSchema executes a request for just the delete against the database
func RunDeleteRequestForSchema(schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) error {
	query, err := standardizeQueryAndPopulatePath(q, schema, DELETE)
	if err != nil || query == nil {
		return err
	}

	_, err = db.Exec(context.Background(), query.AsSQL(), query.Data...)
	if err != nil {
		return errors.Wrapf(err, "could not delete from %q", schema.Table)
	}
	return nil
}
