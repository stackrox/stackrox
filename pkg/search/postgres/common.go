package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/random"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/ternary"
)

var (
	log = logging.LoggerForModule()

	emptyQueryErr = errox.InvalidArgs.New("empty query")
)

// QueryType describe what type of query to execute
//
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
	if len(s) == 0 {
		return ""
	}
	varNum := 1
	var newString strings.Builder
	newString.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i < len(s)-1 && s[i] == '$' && s[i+1] == '$' {
			newString.WriteRune('$')
			newString.WriteString(strconv.Itoa(varNum))
			varNum++
			i++
		} else {
			newString.WriteByte(s[i])
		}
	}
	return newString.String()
}

type innerJoin struct {
	leftTable       string
	rightTable      string
	columnNamePairs []walker.ColumnNamePair
}

type query struct {
	Schema           *walker.Schema
	QueryType        QueryType
	PrimaryKeyFields []pgsearch.SelectQueryField
	SelectedFields   []pgsearch.SelectQueryField
	From             string
	Where            string
	Data             []interface{}

	Having     string
	Pagination parsedPaginationQuery
	InnerJoins []innerJoin

	GroupBys []groupByEntry
}

type groupByEntry struct {
	Field pgsearch.SelectQueryField
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

func (q *query) populatePrimaryKeySelectFields() {
	// Note that DB framework version prior to adding this func assumes all primary key fields in Go to be string type.
	for _, pk := range q.Schema.PrimaryKeys() {
		q.PrimaryKeyFields = append(q.PrimaryKeyFields, pgsearch.SelectQueryField{
			SelectPath: qualifyColumn(pk.Schema.Table, pk.ColumnName, ternary.String(pk.SQLType == "uuid", "::text", "")),
			Alias:      strings.Join([]string{q.Schema.Table, pk.ColumnName}, "_"),
			FieldType:  pk.DataType,
		})
	}

	if len(q.PrimaryKeyFields) == 0 {
		return
	}
	// If we do not need to apply distinct clause to the primary keys, then we are done here.
	if !q.DistinctAppliedOnPrimaryKeySelect() {
		return
	}

	// Collect select paths and apply distinct clause.
	outStr := make([]string, 0, len(q.PrimaryKeyFields))
	for _, f := range q.PrimaryKeyFields {
		outStr = append(outStr, f.SelectPath)
	}
	columnName := ternary.String(len(q.PrimaryKeyFields) > 1, "pks", q.PrimaryKeyFields[0].Alias)
	q.PrimaryKeyFields = q.PrimaryKeyFields[:0]
	q.PrimaryKeyFields = append(q.PrimaryKeyFields, pgsearch.SelectQueryField{
		SelectPath: fmt.Sprintf("distinct(%s)", stringutils.JoinNonEmpty(",", outStr...)),
		Alias:      strings.Join([]string{"distinct", q.Schema.Table, columnName}, "_"),
	})
}

func (q *query) getPortionBeforeFromClause() string {
	switch q.QueryType {
	case DELETE:
		return "delete"
	case COUNT:
		countOn := "*"
		if q.DistinctAppliedOnPrimaryKeySelect() {
			var primaryKeyPaths []string
			// Always select the primary keys for count.
			for _, pk := range q.Schema.PrimaryKeys() {
				primaryKeyPaths = append(primaryKeyPaths, qualifyColumn(pk.Schema.Table, pk.ColumnName, ""))
			}
			countOn = fmt.Sprintf("distinct(%s)", strings.Join(primaryKeyPaths, ", "))
		}
		return fmt.Sprintf("select count(%s)", countOn)
	case GET:
		return fmt.Sprintf("select %q.serialized", q.From)
	case SEARCH:
		// Always select the primary keys first.
		var selectPaths []string
		for _, field := range q.PrimaryKeyFields {
			selectPaths = append(selectPaths, field.PathForSelectPortion())
		}
		for _, field := range q.SelectedFields {
			selectPaths = append(selectPaths, field.PathForSelectPortion())
		}

		// Extra fields are group by fields and order by fields.
		for _, field := range q.ExtraSelectedFieldPaths() {
			selectPaths = append(selectPaths, field.PathForSelectPortion())
		}
		return "select " + stringutils.JoinNonEmpty(", ", selectPaths...)
	}
	panic(fmt.Sprintf("unhandled query type %s", q.QueryType))
}

func (q *query) DistinctAppliedOnPrimaryKeySelect() bool {
	// If this involves multiple tables, then we need to wrap the primary key portion in a distinct, because
	// otherwise there could be multiple rows with the same primary key in the join table.
	// TODO(viswa): we might be able to do this even more narrowly
	return len(q.InnerJoins) > 0 && len(q.GroupBys) == 0
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

	if len(q.GroupBys) > 0 {
		groupByClauses := make([]string, 0, len(q.GroupBys))
		for _, entry := range q.GroupBys {
			groupByClauses = append(groupByClauses, entry.Field.SelectPath)
		}
		querySB.WriteString(" group by ")
		querySB.WriteString(strings.Join(groupByClauses, ", "))
	}
	if q.Having != "" {
		querySB.WriteString(" having ")
		querySB.WriteString(replaceVars(q.Having))
	}
	if paginationSQL := q.Pagination.AsSQL(); paginationSQL != "" {
		querySB.WriteString(" ")
		querySB.WriteString(paginationSQL)
	}
	return querySB.String()
}

func qualifyColumn(table, column, cast string) string {
	return table + "." + column + cast
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

func populateGroupBy(querySoFar *query, schema *walker.Schema, queryFields map[string]searchFieldMetadata) {
	// Explicit GROUP BY clauses are not supported yet. If a query field (in select or order by) is a derived field requiring a group by clause,
	// default to primary key grouping. Note that all fields in the query, including pagination, are in `queryFields`.
	for _, field := range queryFields {
		if field.derivedMetadata == nil {
			continue
		}
		switch field.derivedMetadata.DerivationType {
		case searchPkg.CountDerivationType:
			applyGroupByPrimaryKeys(querySoFar, schema)
			return
		}
	}
}

func applyGroupByPrimaryKeys(querySoFar *query, schema *walker.Schema) {
	for _, pk := range schema.PrimaryKeys() {
		var cast string
		if pk.SQLType == "uuid" {
			cast = "::text"
		}
		querySoFar.GroupBys = append(querySoFar.GroupBys, groupByEntry{
			Field: pgsearch.SelectQueryField{
				SelectPath: qualifyColumn(pk.Schema.Table, pk.ColumnName, cast),
				FieldType:  pk.DataType,
			},
		})
	}
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
			var cast string
			if schema.ID().SQLType == "uuid" {
				cast = "::text"
			}
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field: pgsearch.SelectQueryField{
					SelectPath: qualifyColumn(schema.Table, schema.ID().ColumnName, cast),
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

		var cast string
		if dbField.SQLType == "uuid" {
			cast = "::text"
		}

		if fieldMetadata.derivedMetadata == nil {
			querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
				Field: pgsearch.SelectQueryField{
					SelectPath: qualifyColumn(dbField.Schema.Table, dbField.ColumnName, cast),
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
						SelectPath: fmt.Sprintf("count(%s)", qualifyColumn(dbField.Schema.Table, dbField.ColumnName, "")),
						Alias:      strings.Join([]string{"count", dbField.Schema.Table, dbField.ColumnName}, "_"),
						FieldType:  walker.Integer,
					},
					Descending: so.GetReversed(),
				})
			case searchPkg.SimpleReverseSortDerivationType:
				querySoFar.Pagination.OrderBys = append(querySoFar.Pagination.OrderBys, orderByEntry{
					Field: pgsearch.SelectQueryField{
						SelectPath: qualifyColumn(dbField.Schema.Table, dbField.ColumnName, cast),
						FieldType:  dbField.DataType,
					},
					Descending: !so.GetReversed(),
				})
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
	nowForQuery := time.Now()
	standardizeFieldNamesInQuery(q)
	innerJoins, dbFields := getJoinsAndFields(schema, q)

	queryEntry, err := compileQueryToPostgres(schema, q, dbFields, nowForQuery)
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

	parsedQuery := &query{
		Schema:     schema,
		QueryType:  queryType,
		InnerJoins: innerJoins,
		From:       schema.Table,
	}
	if queryEntry != nil {
		parsedQuery.Where = queryEntry.Where.Query
		parsedQuery.Data = queryEntry.Where.Values
		parsedQuery.SelectedFields = queryEntry.SelectedFields
		if queryEntry.Having != nil {
			parsedQuery.Having = queryEntry.Having.Query
			parsedQuery.Data = append(parsedQuery.Data, queryEntry.Having.Values...)
		}
	}
	populateGroupBy(parsedQuery, schema, dbFields)
	if err := populatePagination(parsedQuery, q.GetPagination(), schema, dbFields); err != nil {
		return nil, err
	}
	if err := applyPaginationForSearchAfter(parsedQuery); err != nil {
		return nil, err
	}

	// Populate primary key select fields once so that we do not have to evaluate.
	parsedQuery.populatePrimaryKeySelectFields()

	// Nothing more to do. Currently, GROUP BY clause is only supported on primary keys.
	return parsedQuery, nil
}

func combineQueryEntries(entries []*pgsearch.QueryEntry, separator string) *pgsearch.QueryEntry {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) == 1 {
		return entries[0]
	}
	var whereQueryStrings []string
	var havingQueryStrings []string
	seenSelectFields := set.NewStringSet()
	newQE := &pgsearch.QueryEntry{}
	for _, entry := range entries {
		whereQueryStrings = append(whereQueryStrings, entry.Where.Query)
		newQE.Where.Values = append(newQE.Where.Values, entry.Where.Values...)
		for _, selectedField := range entry.SelectedFields {
			if seenSelectFields.Add(selectedField.SelectPath) {
				newQE.SelectedFields = append(newQE.SelectedFields, selectedField)
			}
		}
		if len(entry.GroupBy) > 0 {
			newQE.GroupBy = append(newQE.GroupBy, entry.GroupBy...)
		}
		if entry.Having != nil {
			if newQE.Having == nil {
				newQE.Having = &pgsearch.WhereClause{}
			}
			newQE.Having.Values = append(newQE.Having.Values, entry.Having.Values...)
			havingQueryStrings = append(havingQueryStrings, entry.Having.Query)
		}
	}
	newQE.Where.Query = fmt.Sprintf("(%s)", strings.Join(whereQueryStrings, separator))
	if newQE.Having != nil {
		newQE.Having.Query = fmt.Sprintf("(%s)", strings.Join(havingQueryStrings, separator))
	}

	return newQE
}

func entriesFromQueries(
	table *walker.Schema,
	queries []*v1.Query,
	queryFields map[string]searchFieldMetadata,
	nowForQuery time.Time,
) ([]*pgsearch.QueryEntry, error) {
	var entries []*pgsearch.QueryEntry
	for _, q := range queries {
		entry, err := compileQueryToPostgres(table, q, queryFields, nowForQuery)
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

func compileQueryToPostgres(schema *walker.Schema, q *v1.Query, queryFields map[string]searchFieldMetadata, nowForQuery time.Time) (*pgsearch.QueryEntry, error) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			cast := "::text[]"
			if schema.ID().SQLType == "uuid" {
				cast = "::uuid[]"
			}
			return &pgsearch.QueryEntry{Where: pgsearch.WhereClause{
				Query:  fmt.Sprintf("%s.%s = ANY($$%s)", schema.Table, schema.ID().ColumnName, cast),
				Values: []interface{}{subBQ.DocIdQuery.GetIds()},
			}}, nil
		case *v1.BaseQuery_MatchFieldQuery:
			queryFieldMetadata := queryFields[subBQ.MatchFieldQuery.GetField()]
			qe, err := pgsearch.MatchFieldQuery(
				queryFieldMetadata.baseField,
				queryFieldMetadata.derivedMetadata,
				subBQ.MatchFieldQuery.GetValue(),
				subBQ.MatchFieldQuery.GetHighlight(),
				nowForQuery,
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
				queryFieldMetadata := queryFields[q.GetField()]
				qe, err := pgsearch.MatchFieldQuery(queryFieldMetadata.baseField, queryFieldMetadata.derivedMetadata, q.GetValue(), q.GetHighlight(), nowForQuery)
				if err != nil {
					return nil, err
				}
				if qe != nil {
					entries = append(entries, qe)
				}
			}
			return combineQueryEntries(entries, " and "), nil
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		entries, err := entriesFromQueries(schema, sub.Conjunction.Queries, queryFields, nowForQuery)
		if err != nil {
			return nil, err
		}
		return combineQueryEntries(entries, " and "), nil
	case *v1.Query_Disjunction:
		entries, err := entriesFromQueries(schema, sub.Disjunction.Queries, queryFields, nowForQuery)
		if err != nil {
			return nil, err
		}
		return combineQueryEntries(entries, " or "), nil
	case *v1.Query_BooleanQuery:
		entries, err := entriesFromQueries(schema, sub.BooleanQuery.Must.Queries, queryFields, nowForQuery)
		if err != nil {
			return nil, err
		}
		cqe := combineQueryEntries(entries, " and ")
		if cqe == nil {
			cqe = pgsearch.NewTrueQuery()
		}

		entries, err = entriesFromQueries(schema, sub.BooleanQuery.MustNot.Queries, queryFields, nowForQuery)
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

type tracedRows struct {
	qe *postgres.QueryEvent
	pgx.Rows
	accessedRows int
}

func (t *tracedRows) Next() bool {
	if !t.Rows.Next() {
		return false
	}
	t.accessedRows++
	return true
}

func (t *tracedRows) Close() {
	t.Rows.Close()
	t.qe.SetRowsAccessed(t.accessedRows)
}

func tracedQuery(ctx context.Context, pool *pgxpool.Pool, sql string, args ...interface{}) (*tracedRows, error) {
	t := time.Now()
	rows, err := pool.Query(ctx, sql, args...)
	return &tracedRows{
		qe:   postgres.AddTracedQuery(ctx, t, sql, args),
		Rows: rows,
	}, err
}

func tracedQueryRow(ctx context.Context, pool *pgxpool.Pool, sql string, args ...interface{}) pgx.Row {
	t := time.Now()
	row := pool.QueryRow(ctx, sql, args...)
	postgres.AddTracedQuery(ctx, t, sql, args)
	return row
}

// RunSearchRequest executes a request against the database for given category
func RunSearchRequest(ctx context.Context, category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool) ([]searchPkg.Result, error) {
	schema := mapping.GetTableFromCategory(category)

	return pgutils.Retry2(func() ([]searchPkg.Result, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		return RunSearchRequestForSchema(ctx, schema, q, db)
	})
}

func retryableRunSearchRequestForSchema(ctx context.Context, query *query, schema *walker.Schema, db *pgxpool.Pool) ([]searchPkg.Result, error) {
	queryStr := query.AsSQL()

	// Assumes that ids are strings.
	numPrimaryKeys := len(schema.PrimaryKeys())
	extraSelectedFields := query.ExtraSelectedFieldPaths()
	numSelectFieldsForPrimaryKey := len(query.PrimaryKeyFields)
	primaryKeysComposite := numPrimaryKeys > 1 && len(query.PrimaryKeyFields) == 1
	bufferToScanRowInto := make([]interface{}, numSelectFieldsForPrimaryKey+len(query.SelectedFields)+len(extraSelectedFields))
	if primaryKeysComposite {
		var outputSlice []interface{}
		bufferToScanRowInto[0] = &outputSlice
	} else {
		for i := 0; i < numPrimaryKeys; i++ {
			bufferToScanRowInto[i] = pointers.String("")
		}
	}
	for i, field := range query.SelectedFields {
		bufferToScanRowInto[i+numSelectFieldsForPrimaryKey] = mustAllocForDataType(field.FieldType)
	}
	for i, field := range extraSelectedFields {
		bufferToScanRowInto[i+numSelectFieldsForPrimaryKey+len(query.SelectedFields)] = mustAllocForDataType(field.FieldType)
	}

	recordIDIdxMap := make(map[string]int)
	var searchResults []searchPkg.Result

	rows, err := tracedQuery(ctx, db, queryStr, query.Data...)
	if err != nil {
		if !pgutils.IsTransientError(err) {
			debug.PrintStack()
		} else {
			log.Debugf("%s", debug.Stack())
		}
		log.Errorf("Query issue: %s %+v: %v", queryStr, redactedQueryData(query), err)
		return nil, err
	}
	defer rows.Close()
	log.Debugf("SEARCH: ran query %s; data %+v", queryStr, redactedQueryData(query))

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

		id := IDFromPks(idParts)
		idx, ok := recordIDIdxMap[id]
		if !ok {
			idx = len(searchResults)
			recordIDIdxMap[id] = idx
			searchResults = append(searchResults, searchPkg.Result{
				ID:      IDFromPks(idParts), // TODO: figure out what separator to use
				Matches: make(map[string][]string),
			})
		}
		result := searchResults[idx]

		if len(query.SelectedFields) > 0 {
			for i, field := range query.SelectedFields {
				returnedValue := bufferToScanRowInto[i+numSelectFieldsForPrimaryKey]
				if field.PostTransform != nil {
					returnedValue = field.PostTransform(returnedValue)
				}
				if matches := mustPrintForDataType(field.FieldType, returnedValue); len(matches) > 0 {
					result.Matches[field.FieldPath] = matches
				}
			}
		}
		searchResults[idx] = result
	}
	return searchResults, nil
}

// RunSearchRequestForSchema executes a request against the database for given schema
func RunSearchRequestForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) ([]searchPkg.Result, error) {
	var query *query
	var err error
	// Add this to be safe and convert panics to errors,
	// since we do a lot of casting and other operations that could potentially panic in this code.
	// Panics are expected ONLY in the event of a programming error, all foreseeable errors are handled
	// the usual way.
	defer func() {
		if r := recover(); r != nil {
			if query != nil {
				log.Errorf("Query issue: %s %+v: %v", query.AsSQL(), redactedQueryData(query), r)
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
	return pgutils.Retry2(func() ([]searchPkg.Result, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		return retryableRunSearchRequestForSchema(ctx, query, schema, db)
	})
}

// RunCountRequest executes a request for just the count against the database
func RunCountRequest(ctx context.Context, category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool) (int, error) {
	schema := mapping.GetTableFromCategory(category)

	return pgutils.Retry2(func() (int, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		return RunCountRequestForSchema(ctx, schema, q, db)
	})
}

// RunCountRequestForSchema executes a request for just the count against the database
func RunCountRequestForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) (int, error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, COUNT)
	if err != nil || query == nil {
		return 0, err
	}
	queryStr := query.AsSQL()

	return pgutils.Retry2(func() (int, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		var count int
		row := tracedQueryRow(ctx, db, queryStr, query.Data...)
		if err := row.Scan(&count); err != nil {
			if !pgutils.IsTransientError(err) {
				debug.PrintStack()
			} else {
				log.Debugf("%s", debug.Stack())
			}
			log.Errorf("Query issue: %s %+v: %v", queryStr, redactedQueryData(query), err)
			return 0, err
		}
		return count, nil
	})
}

type unmarshaler[T any] interface {
	proto.Unmarshaler
	*T
}

// RunGetQueryForSchema executes a request for just the search against the database
func RunGetQueryForSchema[T any, PT unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) (*T, error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, emptyQueryErr
	}
	queryStr := query.AsSQL()

	return pgutils.Retry2(func() (*T, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		row := tracedQueryRow(ctx, db, queryStr, query.Data...)
		return unmarshal[T, PT](row)
	})
}

func retryableRunGetManyQueryForSchema[T any, PT unmarshaler[T]](ctx context.Context, query *query, db *pgxpool.Pool) ([]*T, error) {
	queryStr := query.AsSQL()
	rows, err := tracedQuery(ctx, db, queryStr, query.Data...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows[T, PT](rows)
}

// RunGetManyQueryForSchema executes a request for just the search against the database and unmarshal it to given type.
func RunGetManyQueryForSchema[T any, PT unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) ([]*T, error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, emptyQueryErr
	}

	return pgutils.Retry2(func() ([]*T, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		return retryableRunGetManyQueryForSchema[T, PT](ctx, query, db)
	})
}

// RunCursorQueryForSchema creates a cursor against the database
func RunCursorQueryForSchema[T any, PT unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) (fetcher func(n int) ([]*T, error), closer func(), err error) {
	query, err := standardizeQueryAndPopulatePath(q, schema, GET)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error creating query")
	}
	if query == nil {
		return nil, nil, emptyQueryErr
	}

	queryStr := query.AsSQL()

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating transaction")
	}
	closer = func() {
		if err := tx.Commit(ctx); err != nil {
			log.Errorf("error committing cursor transaction: %v", err)
		}
	}

	cursor, err := random.GenerateString(16, random.CaseInsensitiveAlpha)
	if err != nil {
		closer()
		return nil, nil, errors.Wrap(err, "creating cursor name")
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("DECLARE %s CURSOR FOR %s", cursor, queryStr), query.Data...)
	if err != nil {
		closer()
		return nil, nil, errors.Wrap(err, "creating cursor")
	}

	return func(n int) ([]*T, error) {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM %s", n, cursor))
		if err != nil {
			return nil, errors.Wrap(err, "advancing in cursor")
		}
		defer rows.Close()

		return scanRows[T, PT](rows)
	}, closer, nil
}

// RunDeleteRequestForSchema executes a request for just the delete against the database
func RunDeleteRequestForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db *pgxpool.Pool) error {
	query, err := standardizeQueryAndPopulatePath(q, schema, DELETE)
	if err != nil || query == nil {
		return err
	}

	return pgutils.Retry(func() error {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		_, err = db.Exec(ctx, query.AsSQL(), query.Data...)
		if err != nil {
			return errors.Wrapf(err, "could not delete from %q", schema.Table)
		}
		return err
	})
}

// redactedQueryData returns query.Data for logging purposes if this is a dev build. Otherwise, it redacts it as it
// may contain sensitive data.
func redactedQueryData(query *query) interface{} {
	if !devbuild.IsEnabled() || buildinfo.ReleaseBuild {
		// On a release build or non-dev build, redact query data
		return "[REDACTED]"
	}
	// Otherwise, allow the logging
	return query.Data
}

// helper functions
///////////////////

func scanRows[T any, PT unmarshaler[T]](rows pgx.Rows) ([]*T, error) {
	var results []*T
	for rows.Next() {
		msg, err := unmarshal[T, PT](rows)
		if err != nil {
			return nil, err
		}
		results = append(results, msg)
	}
	return results, nil
}

func unmarshal[T any, PT unmarshaler[T]](row pgx.Row) (*T, error) {
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, err
	}
	msg := new(T)
	if err := PT(msg).Unmarshal(data); err != nil {
		return nil, err
	}
	return msg, nil
}
