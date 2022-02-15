package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	printerOnce sync.Once

	queryLock   sync.Mutex
	queryCounts = make(map[string]*queryStats)
)

type SelectType int

const (
	GET    SelectType = 0
	COUNT  SelectType = 1
	VALUE  SelectType = 2
	DELETE SelectType = 3
)

type queryStats struct {
	query  string
	counts int
	nanos  int64
}

func incQueryCount(query string, t time.Time) {
	if strings.Contains(query, "select id from alerts where (alerts.value->'deployment' ->>'id'") {
		debug.PrintStack()
	}
	took := time.Since(t)
	queryLock.Lock()
	defer queryLock.Unlock()
	val, ok := queryCounts[query]
	if !ok {
		queryCounts[query] = &queryStats{
			query:  query,
			counts: 1,
			nanos:  int64(took),
		}
		return
	}
	val.counts++
	val.nanos += int64(took)
}

func printCounts() {
	queryLock.Lock()
	defer queryLock.Unlock()

	var stats []*queryStats
	for _, v := range queryCounts {
		stats = append(stats, v)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].counts > stats[j].counts
	})
	for _, stat := range stats {
		fmt.Printf("%s %d ms avg (%d/%d)\n", stat.query, time.Duration(float64(stat.nanos)/float64(stat.counts)).Milliseconds(), stat.nanos, stat.counts)
	}
}

func runQueryPrinter() {
	printerOnce.Do(func() {
		go func() {
			t := time.NewTicker(30 * time.Second)
			for range t.C {
				printCounts()
			}
		}()
	})
}

type queryTree struct {
	table  string
	tables set.StringSet
}

func (q *queryTree) AddTable(table string) {
	q.tables.Add(strings.ToLower(table))
}

func newQueryTree(table string) *queryTree {
	qt := &queryTree{
		table: table,
	}
	qt.AddTable(table)
	return qt
}

//func (q *queryTree) addElems(elems []searchPkg.PathElem) {
//	currTree := q
//	for _, e := range elems {
//		var ok bool
//		childTree, ok := currTree.children[e.ProtoJSONName]
//		if !ok {
//			childTree = newQueryTree(e)
//			currTree.children[e.ProtoJSONName] = childTree
//		}
//		currTree = childTree
//	}
//}

func populatePathRecursive(tree *queryTree, q *v1.Query, optionsMap searchPkg.OptionsMap) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			// nothing to do here
		case *v1.BaseQuery_MatchFieldQuery:
			// Need to find base value
			field, ok := optionsMap.Get(subBQ.MatchFieldQuery.GetField())
			if !ok {
				return
			}
			tree.AddTable(field.FlatElem.TableName())
		case *v1.BaseQuery_MatchNoneQuery:
			// nothing to here either
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			// Need to split this
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				field, ok := optionsMap.Get(q.GetField())
				if !ok {
					return
				}
				tree.AddTable(field.FlatElem.TableName())
			}
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		for _, cq := range sub.Conjunction.Queries {
			populatePathRecursive(tree, cq, optionsMap)
		}
	case *v1.Query_Disjunction:
		for _, dq := range sub.Disjunction.Queries {
			populatePathRecursive(tree, dq, optionsMap)
		}
	case *v1.Query_BooleanQuery:
		for _, cq := range sub.BooleanQuery.Must.Queries {
			populatePathRecursive(tree, cq, optionsMap)
		}
		for _, dq := range sub.BooleanQuery.MustNot.Queries {
			populatePathRecursive(tree, dq, optionsMap)
		}
	}
}

func needsDistinct(t *queryTree) bool {
	return len(t.tables) > 1
}

func createFromClauseRecursive(t *queryTree, parent string) []string {
	return t.tables.AsSlice()
}

// This function does not currently solve naming collisions, and we'll need to eventually solve for that
func createFROMClause(t *queryTree) string {
	results := createFromClauseRecursive(t, "")
	return "from " + strings.Join(results, ", ")
}

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

type Select struct {
	Query  string
	Fields []*searchPkg.Field
}

type Query struct {
	Select     Select
	From       string
	Where      string
	Pagination string
	Data       []interface{}
}

func (q *Query) String() string {
	query := q.Select.Query + " " + q.From
	if q.Where != "" {
		query += " where " + q.Where
	}
	if q.Pagination != "" {
		query += " " + q.Pagination
	}
	return query
}

func getPaginationQuery(pagination *v1.QueryPagination, table string, optionsMap searchPkg.OptionsMap) (string, error) {
	if pagination == nil {
		return "", nil
	}

	var orderByClauses []string
	for _, so := range pagination.GetSortOptions() {
		direction := "asc"
		if so.GetReversed() {
			direction = "desc"
		}
		field, ok := optionsMap.Get(so.GetField())
		if !ok {
			return "", fmt.Errorf("cannot sort by field %s on table %s", so.GetField(), table)
		}
		orderByClauses = append(orderByClauses, field.FlatElem.TablePrefixed()+" "+direction)
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

func generateSelectFieldsRecursive(table string, added set.StringSet, q *v1.Query, optionsMap searchPkg.OptionsMap) ([]string, []*searchPkg.Field) {
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
				return []string{field.FlatElem.TablePrefixed()}, []*searchPkg.Field{field}
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
				if q.Highlight && added.Add(field.FieldPath) {
					paths = append(paths, field.FlatElem.TablePrefixed())
					fields = append(fields, field)
				}
			}
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		var (
			paths  []string
			fields []*searchPkg.Field
		)
		for _, cq := range sub.Conjunction.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(table, added, cq, optionsMap)
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
			localPaths, localFields := generateSelectFieldsRecursive(table, added, dq, optionsMap)
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
			localPaths, localFields := generateSelectFieldsRecursive(table, added, cq, optionsMap)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		for _, dq := range sub.BooleanQuery.MustNot.Queries {
			localPaths, localFields := generateSelectFieldsRecursive(table, added, dq, optionsMap)
			paths = append(paths, localPaths...)
			fields = append(fields, localFields...)
		}
		return paths, fields
	}
	return nil, nil
}

func generateSelectFields(table string, tree *queryTree, q *v1.Query, optionsMap searchPkg.OptionsMap, selectType SelectType) Select {
	var sel Select
	if selectType == DELETE {
		sel.Query = "delete"
		return sel
	}

	distinct := needsDistinct(tree)

	if selectType == COUNT {
		if distinct {
			sel.Query = "select count(distinct id)"
		} else {
			sel.Query = "select count(*)"
		}
		return sel
	}
	added := set.NewStringSet()
	paths, fields := generateSelectFieldsRecursive(table, added, q, optionsMap)

	if distinct {
		if len(paths) > 0 {
			log.Errorf("UNEXPECTED: Highlights on nested JSONB field: %+v", paths)
		}
		sel.Query = "select distinct id"
		return sel
	}
	values := []string{"id"}
	if selectType == VALUE {
		paths = append(values, "serialized")
	} else {
		paths = append(values, paths...)
	}
	sel.Query = fmt.Sprintf("select %s", strings.Join(paths, ","))
	sel.Fields = fields
	return sel
}

func populatePath(q *v1.Query, optionsMap searchPkg.OptionsMap, table string, selectType SelectType) (*Query, error) {
	tree := newQueryTree(table)
	populatePathRecursive(tree, q, optionsMap)
	fromClause := createFROMClause(tree)

	selQuery := generateSelectFields(table, tree, q, optionsMap, selectType)

	// Building the where clause is the hardest part
	//printTree(tree, "")
	queryEntry, err := compileBaseQuery(table, q, optionsMap)
	if err != nil {
		return nil, err
	}
	pagination, err := getPaginationQuery(q.Pagination, table, optionsMap)
	if err != nil {
		return nil, err
	}
	if queryEntry == nil {
		return &Query{
			Select:     selQuery,
			From:       fromClause,
			Pagination: pagination,
		}, nil
	}

	// This is hack to try and fix the subjects test
	if needsDistinct(tree) {
		var joins []string
		for t := range tree.tables {
			if strings.EqualFold(t, tree.table) {
				continue
			}
			joins = append(joins, fmt.Sprintf("%s.id = %s.parent_id", tree.table, t))
		}
		queryEntry.Query += " and " + strings.Join(joins, " and ")
	}

	return &Query{
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

func entriesFromQueries(table string, queries []*v1.Query, optionsMap searchPkg.OptionsMap) ([]*pgsearch.QueryEntry, error) {
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

func tableFromBaseQuery(bq *v1.BaseQuery, optionsMap searchPkg.OptionsMap) (string, bool) {
	switch subBQ := bq.Query.(type) {
	case *v1.BaseQuery_DocIdQuery:
		return "", false
	case *v1.BaseQuery_MatchFieldQuery:
		field, ok := optionsMap.Get(subBQ.MatchFieldQuery.GetField())
		if !ok {
			return "", false
		}
		return mapping.GetTableFromCategory(field.Category), true
	case *v1.BaseQuery_MatchNoneQuery:
		return "", false
	case *v1.BaseQuery_MatchLinkedFieldsQuery:
		if queries := subBQ.MatchLinkedFieldsQuery.Query; len(queries) != 0 {
			field, ok := optionsMap.Get(queries[0].GetField())
			if !ok {
				return "", false
			}
			return mapping.GetTableFromCategory(field.Category), true
		}
	default:
		panic("unsupported")
	}
	return "", false
}

func compileBaseQuery(table string, q *v1.Query, optionsMap searchPkg.OptionsMap) (*pgsearch.QueryEntry, error) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		queryTable, ok := tableFromBaseQuery(sub.BaseQuery, optionsMap)
		if ok && !strings.EqualFold(queryTable, table) {
			log.Infof("Can't handle queryTable=%s, but table=%s", queryTable, table)
			return nil, nil
		}
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			return &pgsearch.QueryEntry{
				Query:  fmt.Sprintf("%s.id = ANY($$::text[])", table),
				Values: []interface{}{pq.Array(subBQ.DocIdQuery.GetIds())},
			}, nil
		case *v1.BaseQuery_MatchFieldQuery:
			return pgsearch.MatchFieldQuery(table, subBQ.MatchFieldQuery, optionsMap)
		case *v1.BaseQuery_MatchNoneQuery:
			return nil, nil
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			var entries []*pgsearch.QueryEntry
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				qe, err := pgsearch.MatchFieldQuery(table, q, optionsMap)
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
		entries, err := entriesFromQueries(table, sub.Conjunction.Queries, optionsMap)
		if err != nil {
			return nil, err
		}
		return multiQueryFromQueryEntries(entries, " and "), nil
	case *v1.Query_Disjunction:
		entries, err := entriesFromQueries(table, sub.Disjunction.Queries, optionsMap)
		if err != nil {
			return nil, err
		}
		return multiQueryFromQueryEntries(entries, " or "), nil
	case *v1.Query_BooleanQuery:
		entries, err := entriesFromQueries(table, sub.BooleanQuery.Must.Queries, optionsMap)
		if err != nil {
			return nil, err
		}
		cqe := multiQueryFromQueryEntries(entries, " and ")
		if cqe == nil {
			cqe = pgsearch.NewTrueQuery()
		}

		entries, err = entriesFromQueries(table, sub.BooleanQuery.MustNot.Queries, optionsMap)
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

func RunSearchRequestValue(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) (pgx.Rows, error) {
	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), VALUE)
	if err != nil {
		return nil, err
	}

	queryStr := query.String()

	runQueryPrinter()
	t := time.Now()
	defer func() {
		incQueryCount(queryStr, t)
	}()

	rows, err := db.Query(context.Background(), replaceVars(queryStr), query.Data...)
	if err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
		return nil, err
	}
	return rows, err
}

func RunSearchRequestDelete(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) error {
	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), DELETE)
	if err != nil {
		return err
	}
	// No pagination for deletes
	query.Pagination = ""

	queryStr := query.String()

	runQueryPrinter()
	t := time.Now()
	defer func() {
		incQueryCount(queryStr, t)
	}()

	_, err = db.Exec(context.Background(), replaceVars(queryStr), query.Data...)
	if err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
		return err
	}
	return nil
}
//
//func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
//	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), GET)
//	if err != nil {
//		return nil, err
//	}
//
//	queryStr := query.String()
//
//	runQueryPrinter()
//	t := time.Now()
//	defer func() {
//		incQueryCount(queryStr, t)
//	}()
//
//	rows, err := db.Query(context.Background(), replaceVars(queryStr), query.Data...)
//	if err != nil {
//		debug.PrintStack()
//		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
//		return nil, err
//	}
//	defer rows.Close()
//
//	var searchResults []searchPkg.Result
//
//	highlightedResults := make([]interface{}, len(query.Select.Fields)+1)
//	for i := range highlightedResults {
//		highlightedResults[i] = pointers.String("")
//	}
//	for rows.Next() {
//		if err := rows.Scan(highlightedResults...); err != nil {
//			return nil, err
//		}
//		result := searchPkg.Result{
//			ID: valueFromStringPtrInterface(highlightedResults[0]),
//		}
//		if len(query.Select.Fields) > 0 {
//			result.Matches = make(map[string][]string)
//			for i, field := range query.Select.Fields {
//				result.Matches[field.FieldPath] = []string{valueFromStringPtrInterface(highlightedResults[i+1])}
//			}
//		}
//		searchResults = append(searchResults, result)
//	}
//	return searchResults, nil
//}

func RunCountRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) (int, error) {
	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), COUNT)
	if err != nil {
		return 0, err
	}

	queryStr := query.String()
	runQueryPrinter()
	t := time.Now()
	defer func() {
		incQueryCount(queryStr, t)
	}()

	var count int
	row := db.QueryRow(context.Background(), replaceVars(queryStr), query.Data...)
	if err := row.Scan(&count); err != nil {
		debug.PrintStack()
		log.Errorf("Query issue: %s %+v: %v", query, query.Data, err)
		return 0, err
	}
	return count, nil
}
