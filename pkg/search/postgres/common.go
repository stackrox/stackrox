package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/random"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	emptyQueryErr = errox.InvalidArgs.New("empty query")

	cursorDefaultTimeout = env.PostgresDefaultCursorTimeout.DurationSetting()

	tableWithImageIDToField = map[string]string{
		pkgSchema.ImagesTableName:              "Id",
		pkgSchema.ImageComponentEdgesTableName: "ImageId",
	}

	tableWithImageCVEIDToField = map[string]string{
		pkgSchema.ImageCvesTableName:              "Id",
		pkgSchema.ImageComponentCveEdgesTableName: "ImageCveId",
	}
)

const cursorBatchSize = 1000

type cursorSession struct {
	id string
	tx *postgres.Tx

	close func()
}

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
	SELECT
	DELETERETURNINGIDS
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

type Join struct {
	leftTable       string
	rightTable      string
	joinType        JoinType
	columnNamePairs []walker.ColumnNamePair
}

type query struct {
	Schema           *walker.Schema
	QueryType        QueryType
	PrimaryKeyFields []pgsearch.SelectQueryField
	SelectedFields   []pgsearch.SelectQueryField
	ReturningFields  []pgsearch.SelectQueryField
	From             string
	Where            string
	Data             []interface{}

	Having     string
	Pagination parsedPaginationQuery
	Joins      []Join

	// This indicates if a primary key is present in the group by clause. Unless GROUP BY clause is explicitly provided,
	// we order the results by the primary key of the schema.
	GroupByPrimaryKey bool
	GroupBys          []groupByEntry

	// This field indicates if 'Distinct' is applied in the select portion of the query
	DistinctAppliedToSelects bool
}

type groupByEntry struct {
	Field pgsearch.SelectQueryField
}

// ExtraSelectedFieldPaths includes extra fields to add to the select clause.
// We don't care about actually reading the values of these fields, they're
// there to make SQL happy.
func (q *query) ExtraSelectedFieldPaths() []pgsearch.SelectQueryField {
	if !q.isDistinctAppliedToSelects() && !q.groupByNonPKFields() {
		return nil
	}

	seenSelectPathsToIndex := make(map[string]int)
	for idx, f := range q.SelectedFields {
		seenSelectPathsToIndex[f.SelectPath] = idx
		seenSelectPathsToIndex[f.Alias] = idx
	}

	var out []pgsearch.SelectQueryField
	for _, groupBy := range q.GroupBys {
		// Make sure we do not add duplicate select paths to the query.
		idx, found := seenSelectPathsToIndex[groupBy.Field.Alias]
		if !found {
			idx, found = seenSelectPathsToIndex[groupBy.Field.SelectPath]
			if !found {
				out = append(out, groupBy.Field)
			}
		}

		if found {
			// `FromGroupBy` property determines whether we want to apply json_agg() to the field later when
			// generating the SQL string.
			field := &q.SelectedFields[idx]
			field.FromGroupBy = true
		}

		// -1 because the select path was added to `out` and not to `q.SelectedFields`.
		seenSelectPathsToIndex[groupBy.Field.SelectPath] = -1
	}

	for _, orderByEntry := range q.Pagination.OrderBys {
		// Make sure we do not add duplicate select paths to the query.
		_, found := seenSelectPathsToIndex[orderByEntry.Field.Alias]
		if !found {
			_, found = seenSelectPathsToIndex[orderByEntry.Field.SelectPath]
			if !found {
				out = append(out, orderByEntry.Field)
			}
		}

		// -1 because the select path was added to `out` and not to `q.SelectedFields`.
		seenSelectPathsToIndex[orderByEntry.Field.SelectPath] = -1
	}
	return out
}

func (q *query) populatePrimaryKeySelectFields() {
	// Note that db framework version prior to adding this func assumes all primary key fields in Go to be string type.
	pks := q.Schema.PrimaryKeys()
	for idx := range pks {
		pk := &pks[idx]
		q.PrimaryKeyFields = append(q.PrimaryKeyFields, selectQueryField(pk.Search.FieldName, pk, false, aggregatefunc.Unset, ""))

		if len(q.PrimaryKeyFields) == 0 {
			return
		}
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

	alias := q.PrimaryKeyFields[0].Alias
	if len(q.PrimaryKeyFields) > 1 {
		alias = q.Schema.Table + "pks" // this will result in distinct(id, name) as tablepks
	}

	q.PrimaryKeyFields = q.PrimaryKeyFields[:0]
	q.PrimaryKeyFields = append(q.PrimaryKeyFields, pgsearch.SelectQueryField{
		SelectPath: fmt.Sprintf("distinct(%s)", stringutils.JoinNonEmpty(",", outStr...)),
		Alias:      alias,
	})
	q.DistinctAppliedToSelects = true
}

func (q *query) getPortionBeforeFromClause() string {
	switch q.QueryType {
	case DELETE, DELETERETURNINGIDS:
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
		var selectStrs []string
		// Always select the primary keys first.
		for _, f := range q.PrimaryKeyFields {
			selectStrs = append(selectStrs, f.PathForSelectPortion())
		}
		for _, f := range q.SelectedFields {
			selectStrs = append(selectStrs, f.PathForSelectPortion())
		}
		for _, f := range q.ExtraSelectedFieldPaths() {
			selectStrs = append(selectStrs, f.PathForSelectPortion())
		}
		return "select " + stringutils.JoinNonEmpty(", ", selectStrs...)
	case SELECT:
		allSelectFields := q.SelectedFields
		allSelectFields = append(allSelectFields, q.ExtraSelectedFieldPaths()...)

		selectStrs := make([]string, 0, len(allSelectFields))
		for _, field := range allSelectFields {
			if q.groupByNonPKFields() && !field.FromGroupBy && !field.DerivedField {
				selectStrs = append(selectStrs, fmt.Sprintf("jsonb_agg(%s) as %s", field.SelectPath, field.Alias))
			} else {
				selectStrs = append(selectStrs, field.PathForSelectPortion())
			}
		}
		return "select " + stringutils.JoinNonEmpty(", ", selectStrs...)
	}
	panic(fmt.Sprintf("unhandled query type %s", q.QueryType))
}

func (q *query) DistinctAppliedOnPrimaryKeySelect() bool {
	// If this involves multiple tables, then we need to wrap the primary key portion in a distinct, because
	// otherwise there could be multiple rows with the same primary key in the join table.
	// TODO(viswa): we might be able to do this even more narrowly
	return len(q.Joins) > 0 && len(q.GroupBys) == 0
}

// groupByNonPKFields returns true if a group by clause based on fields other than primary keys is present in the query.
func (q *query) groupByNonPKFields() bool {
	return len(q.GroupBys) > 0 && !q.GroupByPrimaryKey
}

func (q *query) isDistinctAppliedToSelects() bool {
	return q != nil && q.DistinctAppliedToSelects
}

func (q *query) AsSQL() string {
	if q == nil {
		return ""
	}

	var querySB strings.Builder

	querySB.WriteString(q.getPortionBeforeFromClause())
	querySB.WriteString(" from ")
	querySB.WriteString(q.From)

	for i, join := range q.Joins {
		if join.joinType == Inner {
			querySB.WriteString(" inner join ")
		} else {
			querySB.WriteString(" left join ")
		}
		querySB.WriteString(join.rightTable)
		querySB.WriteString(" on")

		if env.ImageCVEEdgeCustomJoin.BooleanSetting() && !features.FlattenCVEData.Enabled() {
			if (i == len(q.Joins)-1) && (join.rightTable == pkgSchema.ImageCveEdgesTableName) {
				// Step 4: Join image_cve_edges table such that both its ImageID and ImageCveId columns are matched with the joins so far
				imageIDTable := findImageIDTableAndField(q.Joins)
				imageCVEIDTable := findImageCVEIDTableAndField(q.Joins)
				if imageIDTable != "" && imageCVEIDTable != "" {
					imageIDField := tableWithImageIDToField[imageIDTable]
					imageCVEIDField := tableWithImageCVEIDToField[imageCVEIDTable]
					querySB.WriteString(fmt.Sprintf("(%s.%s = %s.%s and %s.%s = %s.%s)",
						imageIDTable, imageIDField, pkgSchema.ImageCveEdgesTableName, "ImageId",
						imageCVEIDTable, imageCVEIDField, pkgSchema.ImageCveEdgesTableName, "ImageCveId"))
					continue
				} else {
					log.Error("Could not find tables to match both ImageId and ImageCveId columns on image_cve_edges table. " +
						"Continuing with incomplete join")
				}
			}
		}

		for i, columnNamePair := range join.columnNamePairs {
			if i > 0 {
				querySB.WriteString(" and")
			}
			querySB.WriteString(fmt.Sprintf(" %s.%s = %s.%s", join.leftTable, columnNamePair.ColumnNameInThisSchema, join.rightTable, columnNamePair.ColumnNameInOtherSchema))
		}
	}
	if q.Where != "" {
		querySB.WriteString(" where ")
		querySB.WriteString(q.Where)
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
		querySB.WriteString(q.Having)
	}
	if len(q.ReturningFields) > 0 {
		querySB.WriteString(" returning ")
		returnedColumnPaths := make([]string, 0, len(q.ReturningFields))
		for _, f := range q.ReturningFields {
			returnedColumnPaths = append(returnedColumnPaths, f.PathForSelectPortion())
		}
		querySB.WriteString(strings.Join(returnedColumnPaths, ", "))
	}
	if paginationSQL := q.Pagination.AsSQL(); paginationSQL != "" {
		querySB.WriteString(" ")
		querySB.WriteString(paginationSQL)
	}
	// Performing this operation on full query is safe since table names and column names
	// can only contain alphanumeric and underscore character.
	queryString := replaceVars(querySB.String())
	if env.PostgresQueryLogger.BooleanSetting() {
		log.Info(queryString)
	}
	return queryString
}

func findImageIDTableAndField(joins []Join) string {
	for _, join := range joins {
		_, found := tableWithImageIDToField[join.leftTable]
		if found {
			return join.leftTable
		}
		_, found = tableWithImageIDToField[join.rightTable]
		if found {
			return join.rightTable
		}
	}
	return ""
}

func findImageCVEIDTableAndField(joins []Join) string {
	for _, join := range joins {
		_, found := tableWithImageCVEIDToField[join.leftTable]
		if found {
			return join.leftTable
		}
		_, found = tableWithImageCVEIDToField[join.rightTable]
		if found {
			return join.rightTable
		}
	}
	return ""
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
			orderByClauses = append(orderByClauses, fmt.Sprintf("%s %s", entry.Field.SelectPath, pkgUtils.IfThenElse(entry.Descending, "desc", "asc")))
		}
		paginationSB.WriteString(fmt.Sprintf("order by %s nulls last", strings.Join(orderByClauses, ", ")))
	}
	if p.Limit > 0 {
		paginationSB.WriteString(fmt.Sprintf(" LIMIT %d", p.Limit))
	}
	if p.Offset > 0 {
		paginationSB.WriteString(fmt.Sprintf(" OFFSET %d", p.Offset))
	}
	return paginationSB.String()
}

func standardizeQueryAndPopulatePath(ctx context.Context, q *v1.Query, schema *walker.Schema, queryType QueryType) (*query, error) {
	nowForQuery := time.Now()
	q, sacErr := enrichQueryWithSACFilter(ctx, q, schema, queryType)
	if sacErr != nil {
		return nil, sacErr
	}
	standardizeFieldNamesInQuery(q)
	joins, dbFields := getJoinsAndFields(schema, q)

	var err error
	if env.ImageCVEEdgeCustomJoin.BooleanSetting() && !features.FlattenCVEData.Enabled() {
		joins, err = handleImageCveEdgesTableInJoins(schema, joins)
		if err != nil {
			return nil, err
		}
	}

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
		Schema:    schema,
		QueryType: queryType,
		Joins:     joins,
		From:      schema.Table,
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

	if err := populateGroupBy(parsedQuery, q.GetGroupBy(), schema, dbFields); err != nil {
		return nil, err
	}
	if err := populatePagination(parsedQuery, q.GetPagination(), schema, dbFields); err != nil {
		return nil, err
	}
	if err := applyPaginationForSearchAfter(parsedQuery); err != nil {
		return nil, err
	}

	// Populate primary key select fields once so that we do not have to evaluate multiple times.
	parsedQuery.populatePrimaryKeySelectFields()

	if queryType == DELETERETURNINGIDS {
		parsedQuery.ReturningFields = parsedQuery.PrimaryKeyFields
	}

	return parsedQuery, nil
}

func handleImageCveEdgesTableInJoins(schema *walker.Schema, joins []Join) ([]Join, error) {
	// By avoiding ImageCveEdgesSchema as long as possible in getJoinsAndFields, we should have ensured that
	// unless ImageCveEdgesSchema is the src schema, it is not a leftTable in any of the inner joins. This means that
	// we have found an alternative route (via image_components) to join image and image_cves tables and if present,
	// image_cve_edges table is only there because of its required fields. In other words, it is not being used to join
	// any two distant tables.
	// But we validate the same just to be safe here
	if schema != pkgSchema.ImageCveEdgesSchema {
		idx, isLeftTable := findTableInJoins(joins, func(join Join) bool {
			return join.leftTable == pkgSchema.ImageCveEdgesTableName
		})

		if isLeftTable {
			return nil, errors.Wrapf(errox.InvariantViolation,
				"Even though '%s' is not the root table in the query, it is the left table in inner join '%v'",
				pkgSchema.ImageCveEdgesTableName, joins[idx])
		}
	}

	// Step 3: If image_cve_edges table is the right table of any inner join, move that join to the end of the list.
	// When building SQL query, this will ensure that we have already joined tables needed to match both CVEId and
	// ImageId columns from image_cve_edges table.
	idx, isRightTable := findTableInJoins(joins, func(join Join) bool {
		return join.rightTable == pkgSchema.ImageCveEdgesTableName
	})
	if isRightTable {
		elem := joins[idx]
		joins = append(joins[:idx], joins[idx+1:]...)
		joins = append(joins, elem)
	}
	return joins, nil
}

func findTableInJoins(innerJoins []Join, matchTables func(join Join) bool) (int, bool) {
	for i, join := range innerJoins {
		if matchTables(join) {
			return i, true
		}
	}
	return -1, false
}

// combineDisjunction tries to optimize disjunction queries with `IN` operator when possible.
// If not it fallbacks to combineQueryEntries
func combineDisjunction(entries []*pgsearch.QueryEntry) *pgsearch.QueryEntry {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) == 1 {
		return entries[0]
	}

	exactQuerySuffix := " = $$"

	seenQueries := set.StringSet{}
	seenSelectFields := set.StringSet{}
	values := make([]any, 0, len(entries))
	// skip for complex queries (having, groupby, multiple values and selects)
	// here we support only simple cases of multiple exact match statements
	// TODO(ROX-27944): add support for complex queries as well
	for _, entry := range entries {
		if entry.Having != nil ||
			len(entry.GroupBy) != 0 ||
			len(entry.Where.Values) != 1 ||
			!strings.HasSuffix(entry.Where.Query, exactQuerySuffix) {
			return combineQueryEntries(entries, " or ")
		}
		for _, selectedField := range entry.SelectedFields {
			seenSelectFields.Add(selectedField.SelectPath)
		}
		seenQueries.Add(entry.Where.Query)
		values = append(values, fmt.Sprintf("%s", entry.Where.Values[0]))
	}

	// if we've seen more than a single exact query this means we have multiple
	// columns there and we cannot apply IN operator there
	// TODO(ROX-27944): handle multiple selected fields
	if len(seenQueries) != 1 || len(seenSelectFields) > 1 {
		return combineQueryEntries(entries, " or ")
	}

	where := seenQueries.GetArbitraryElem()
	where = strings.TrimSuffix(where, exactQuerySuffix)

	return &pgsearch.QueryEntry{
		Where: pgsearch.WhereClause{
			Query:  fmt.Sprintf("%s IN (%s$$)", where, strings.Join(make([]string, len(entries)), "$$, ")),
			Values: values,
		},
		SelectedFields: entries[0].SelectedFields,
		GroupBy:        nil,
	}

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
		// It is possible to have a Having clause and an empty Where.  In that case
		// we need to not add the where strings.  Otherwise, it will add an empty one
		// at the end and a dangling separator.
		if entry.Where.Query != "" {
			whereQueryStrings = append(whereQueryStrings, entry.Where.Query)
			newQE.Where.Values = append(newQE.Where.Values, entry.Where.Values...)
		}
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
	if err := validateDerivedFieldDataType(queryFields); err != nil {
		return nil, err
	}

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
		return combineDisjunction(entries), nil
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
	for idx, s := range q.GetSelects() {
		q.Selects[idx].Field.Name = strings.ToLower(s.GetField().GetName())
		standardizeFieldNamesInQuery(s.GetFilter().GetQuery())
	}

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

	for idx, field := range q.GetGroupBy().GetFields() {
		q.GroupBy.Fields[idx] = strings.ToLower(field)
	}

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

func (t *tracedRows) CommandTag() pgconn.CommandTag {
	return t.Rows.CommandTag()
}

func (t *tracedRows) Err() error {
	return t.Rows.Err()
}

func tracedQuery(ctx context.Context, pool postgres.DB, sql string, args ...interface{}) (*tracedRows, error) {
	t := time.Now()
	rows, err := pool.Query(ctx, sql, args...)
	return &tracedRows{
		qe:   postgres.AddTracedQuery(ctx, t, sql, args),
		Rows: rows,
	}, err
}

func tracedQueryRow(ctx context.Context, pool postgres.DB, sql string, args ...interface{}) pgx.Row {
	t := time.Now()
	row := pool.QueryRow(ctx, sql, args...)
	postgres.AddTracedQuery(ctx, t, sql, args)
	return row
}

func retryableRunSearchRequestForSchema(ctx context.Context, query *query, schema *walker.Schema, db postgres.DB) ([]searchPkg.Result, error) {
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
		log.Errorf("Query issue: %s: %v", queryStr, err)
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(bufferToScanRowInto...); err != nil {
			return nil, errors.Wrap(err, "could not scan row")
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
					result.Matches[field.FieldPath] = append(result.Matches[field.FieldPath], matches...)
				}
			}
		}
		searchResults[idx] = result
	}
	return searchResults, rows.Err()
}

// RunSearchRequestForSchema executes a request against the database for given schema
func RunSearchRequestForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) ([]searchPkg.Result, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}

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

	query, err = standardizeQueryAndPopulatePath(ctx, q, schema, SEARCH)
	if err != nil {
		return nil, err
	}
	// A nil-query implies no results.
	if query == nil {
		return nil, nil
	}
	return pgutils.Retry2(ctx, func() ([]searchPkg.Result, error) {

		return retryableRunSearchRequestForSchema(ctx, query, schema, db)
	})
}

// RunCountRequestForSchema executes a request for just the count against the database
func RunCountRequestForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) (int, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}

	query, err := standardizeQueryAndPopulatePath(ctx, q, schema, COUNT)
	if err != nil || query == nil {
		return 0, err
	}
	queryStr := query.AsSQL()

	return pgutils.Retry2(ctx, func() (int, error) {
		var count int
		row := tracedQueryRow(ctx, db, queryStr, query.Data...)
		if err := row.Scan(&count); err != nil {
			log.Errorf("Query issue: %s: %v", queryStr, err)
			return 0, errors.Wrap(err, "error executing query")
		}
		return count, nil
	})
}

// RunGetQueryForSchema executes a request for just the search against the database
func RunGetQueryForSchema[T any, PT pgutils.Unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) (*T, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}

	query, err := standardizeQueryAndPopulatePath(ctx, q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, emptyQueryErr
	}
	queryStr := query.AsSQL()

	return pgutils.Retry2(ctx, func() (*T, error) {

		row := tracedQueryRow(ctx, db, queryStr, query.Data...)
		return pgutils.Unmarshal[T, PT](row)
	})
}

func retryableRunGetManyQueryForSchema[T any, PT pgutils.Unmarshaler[T]](ctx context.Context, query *query, db postgres.DB) ([]*T, error) {
	queryStr := query.AsSQL()
	rows, err := tracedQuery(ctx, db, queryStr, query.Data...)
	if err != nil {
		return nil, err
	}

	return pgutils.ScanRows[T, PT](rows)
}

// RunGetManyQueryForSchema executes a request for just the search against the database and unmarshal it to given type.
//
// Deprecated: use RunQueryForSchemaFn instead
func RunGetManyQueryForSchema[T any, PT pgutils.Unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) ([]*T, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}

	query, err := standardizeQueryAndPopulatePath(ctx, q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, emptyQueryErr
	}

	return pgutils.Retry2(ctx, func() ([]*T, error) {

		return retryableRunGetManyQueryForSchema[T, PT](ctx, query, db)
	})
}

func prepareQuery(ctx context.Context, schema *walker.Schema, q *v1.Query) (*query, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}

	preparedQuery, err := standardizeQueryAndPopulatePath(ctx, q, schema, GET)
	if err != nil {
		return nil, errors.Wrap(err, "error creating query")
	}
	if preparedQuery == nil {
		return nil, emptyQueryErr
	}

	return preparedQuery, nil
}

func handleRowsWithCallback[T any, PT pgutils.Unmarshaler[T]](ctx context.Context, rows pgx.Rows, callback func(obj PT) error) (int64, error) {
	var data []byte
	tag, err := pgx.ForEachRow(rows, []any{&data}, func() error {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "iterating over rows")
		}

		msg := new(T)
		if errUnmarshal := PT(msg).UnmarshalVTUnsafe(data); errUnmarshal != nil {
			return errUnmarshal
		}
		return callback(msg)
	})

	return tag.RowsAffected(), err
}

func retryableGetRows(ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) (*tracedRows, error) {
	preparedQuery, err := prepareQuery(ctx, schema, q)
	if err != nil {
		return nil, err
	}

	queryStr := preparedQuery.AsSQL()
	return tracedQuery(ctx, db, queryStr, preparedQuery.Data...)
}

func RunQueryForSchemaFn[T any, PT pgutils.Unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB, callback func(obj PT) error) error {
	rows, err := pgutils.Retry2(ctx, func() (*tracedRows, error) {
		return retryableGetRows(ctx, schema, q, db)
	})
	if err != nil {
		return err
	}

	_, err = handleRowsWithCallback(ctx, rows, callback)
	if err != nil {
		return errors.Wrap(err, "processing rows")
	}

	return nil
}

func retryableGetCursorSession(ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) (*cursorSession, error) {
	preparedQuery, err := prepareQuery(ctx, schema, q)
	if err != nil {
		return nil, err
	}

	queryStr := preparedQuery.AsSQL()

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "creating transaction")
	}

	// We have to ensure that cleanup function is called if exit early.
	cleanupFunc := func() {
		if err := tx.Commit(ctx); err != nil {
			log.Errorf("error committing cursor transaction: %v", err)
		}
	}

	cursorSuffix := random.GenerateString(16, random.CaseInsensitiveAlpha)
	cursorId := stringutils.JoinNonEmpty("_", preparedQuery.From, cursorSuffix)

	_, err = tx.Exec(ctx, fmt.Sprintf("DECLARE %s CURSOR FOR %s", cursorId, queryStr), preparedQuery.Data...)
	if err != nil {
		cleanupFunc()
		return nil, errors.Wrap(err, "creating cursor")
	}

	cursor := cursorSession{
		id:    cursorId,
		tx:    tx,
		close: cleanupFunc,
	}

	return &cursor, nil
}

func RunCursorQueryForSchemaFn[T any, PT pgutils.Unmarshaler[T]](ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB, callback func(obj PT) error) error {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, cursorDefaultTimeout)
	defer cancel()

	cursor, err := pgutils.Retry2(ctx, func() (*cursorSession, error) {
		return retryableGetCursorSession(ctx, schema, q, db)
	})
	if err != nil {
		return errors.Wrap(err, "prepare cursor")
	}
	defer cursor.close()

	for {
		rows, err := cursor.tx.Query(ctx, fmt.Sprintf("FETCH %d FROM %s", cursorBatchSize, cursor.id))
		if err != nil {
			return errors.Wrap(err, "advancing in cursor")
		}

		rowsAffected, err := handleRowsWithCallback(ctx, rows, callback)
		if err != nil {
			return errors.Wrap(err, "processing rows")
		}

		if rowsAffected != cursorBatchSize {
			return nil
		}
	}
}

// RunDeleteRequestForSchema executes a request for just the delete against the database
func RunDeleteRequestForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) error {
	if q == nil {
		return nil
	}

	query, err := standardizeQueryAndPopulatePath(ctx, q, schema, DELETE)
	if err != nil || query == nil {
		return err
	}

	queryStr := query.AsSQL()
	return pgutils.Retry(ctx, func() error {
		_, err := db.Exec(ctx, queryStr, query.Data...)
		if err != nil {
			log.Errorf("Query issue: %s: %v", queryStr, err)
			return errors.Wrap(err, "could not delete from database")
		}
		return err
	})
}

// RunDeleteRequestReturningIDsForSchema executes a request for the delete query against the database returning IDs.
func RunDeleteRequestReturningIDsForSchema(ctx context.Context, schema *walker.Schema, q *v1.Query, db postgres.DB) ([]string, error) {
	if q == nil {
		return nil, nil
	}

	query, err := standardizeQueryAndPopulatePath(ctx, q, schema, DELETERETURNINGIDS)
	if err != nil || query == nil {
		return nil, err
	}

	queryStr := query.AsSQL()
	// Assumes that ids are strings.
	numPrimaryKeys := len(schema.PrimaryKeys())
	numSelectFieldsForPrimaryKey := len(query.PrimaryKeyFields)
	primaryKeysComposite := numPrimaryKeys > 1 && len(query.PrimaryKeyFields) == 1
	bufferToScanRowInto := make([]interface{}, numSelectFieldsForPrimaryKey)
	if primaryKeysComposite {
		var outputSlice []interface{}
		bufferToScanRowInto[0] = &outputSlice
	} else {
		for i := 0; i < numPrimaryKeys; i++ {
			bufferToScanRowInto[i] = pointers.String("")
		}
	}
	returnedIDs := make([]string, 0)
	dbErr := pgutils.Retry(ctx, func() error {
		rows, err := tracedQuery(ctx, db, queryStr, query.Data...)
		if err != nil {
			log.Errorf("Query issue: %s: %v", queryStr, err)
			return errors.Wrap(err, "could not delete from database")
		}
		defer rows.Close()
		for rows.Next() {
			if err := rows.Scan(bufferToScanRowInto...); err != nil {
				return errors.Wrap(err, "could not scan row")
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
			returnedIDs = append(returnedIDs, id)
		}
		return err
	})
	if dbErr != nil {
		return nil, err
	}
	return returnedIDs, nil
}

// helper functions
///////////////////

func qualifyColumn(table, column, cast string) string {
	return table + "." + column + cast
}

func validateDerivedFieldDataType(queryFields map[string]searchFieldMetadata) error {
	errList := errorhelpers.NewErrorList("validating supported derived field datatype")
	for _, queryField := range queryFields {
		if queryField.derivedMetadata == nil {
			continue
		}
		dbField := queryField.baseField
		if dbField.Schema.OptionsMap == nil {
			continue
		}

		dataType := dbField.DataType
		if postgres.UnsupportedDerivedFieldDataTypes.Contains(dataType) {
			errList.AddError(errors.Errorf("datatype %s is not supported in aggregation", string(dataType)))
		}
	}
	return errList.ToError()
}
