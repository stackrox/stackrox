package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	printerOnce sync.Once

	queryLock   sync.Mutex
	queryCounts = make(map[string]*queryStats)
)

type queryStats struct {
	query  string
	counts int
	nanos  int64
}

func incQueryCount(query string, t time.Time) {
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
	elem     searchPkg.PathElem
	children map[string]*queryTree
}

func newQueryTree(elem searchPkg.PathElem) *queryTree {
	return &queryTree{
		elem:     elem,
		children: make(map[string]*queryTree),
	}
}

func (q *queryTree) addElems(elems []searchPkg.PathElem) {
	currTree := q
	for _, e := range elems {
		var ok bool
		childTree, ok := currTree.children[e.ProtoJSONName]
		if !ok {
			childTree = newQueryTree(e)
			currTree.children[e.ProtoJSONName] = childTree
		}
		currTree = childTree
	}
}

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
			tree.addElems(field.Elems)
		case *v1.BaseQuery_MatchNoneQuery:
			// nothing to here either
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			// Need to split this
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				field, ok := optionsMap.Get(q.GetField())
				if !ok {
					return
				}
				tree.addElems(field.Elems)
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

func printTree(t *queryTree, indent string) {
	fmt.Println(indent, t.elem.ProtoJSONName, t.elem.Slice)
	for _, children := range t.children {
		printTree(children, indent+"  ")
	}
}

func needsDistinct(t *queryTree) bool {
	for _, childTree := range t.children {
		if childTree.elem.Slice && len(childTree.children) != 0 {
			return true
		}
		if needsDistinct(childTree) {
			return true
		}
	}
	return false
}

func createFromClauseRecursive(t *queryTree, parent string) []string {
	var results []string
	if parent == "" {
		results = append(results, t.elem.ProtoJSONName)
		parent = fmt.Sprintf("%s.value", t.elem.ProtoJSONName)
	}

	for _, childTree := range t.children {
		if len(childTree.children) == 0 {
			if childTree.elem.Slice {
				results = append(results, fmt.Sprintf("jsonb_array_elements_text(%s->'%s') %s", parent, childTree.elem.ProtoJSONName, childTree.elem.ProtoJSONName))
			}
			continue
		}
		localParent := parent
		if childTree.elem.Slice {
			results = append(results, fmt.Sprintf("jsonb_array_elements(%s->'%s') %s", parent, childTree.elem.ProtoJSONName, childTree.elem.ProtoJSONName))
			localParent = childTree.elem.ProtoJSONName
		} else {
			localParent = fmt.Sprintf("%s->'%s'", parent, childTree.elem.ProtoJSONName)
		}
		subRes := createFromClauseRecursive(childTree, localParent)
		results = append(results, subRes...)
	}
	return results
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

type Query struct {
	Select     string
	From       string
	Where      string
	Pagination string
	Data       []interface{}
}

func (q *Query) String() string {
	query := q.Select + " " + q.From
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
		field, ok := optionsMap.Get(so.GetField())
		if !ok {
			return "", fmt.Errorf("cannot sort by field %s on table %s", so.GetField(), table)
		}

		root := field.TopLevelValue()
		if root == "" {
			elemPath := pgsearch.GenerateShortestElemPath(table, field.Elems)
			switch field.Type {
			case v1.SearchDataType_SEARCH_STRING:
				root = pgsearch.RenderFinalPath(elemPath, field.LastElem().ProtoJSONName)
			case v1.SearchDataType_SEARCH_NUMERIC, v1.SearchDataType_SEARCH_ENUM:
				root = fmt.Sprintf("(%s)::numeric", pgsearch.RenderFinalPath(elemPath, field.LastElem().ProtoJSONName))
			case v1.SearchDataType_SEARCH_DATETIME:
				root = fmt.Sprintf("(%s)::timestamp", pgsearch.RenderFinalPath(elemPath, field.LastElem().ProtoJSONName))
			}
		}
		direction := "asc"
		if so.GetReversed() {
			direction = "desc"
		}
		orderByClauses = append(orderByClauses, root+" "+direction)
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

func populatePath(q *v1.Query, optionsMap searchPkg.OptionsMap, table string, count bool) (*Query, error) {
	tree := newQueryTree(searchPkg.PathElem{
		ProtoJSONName: table,
	})
	populatePathRecursive(tree, q, optionsMap)
	fromClause := createFROMClause(tree)

	var distinct string
	if needsDistinct(tree) {
		distinct = "distinct"
	}

	// Initial select, need to support highlights as well
	selectClause := fmt.Sprintf("select %s id", distinct)
	if count {
		if distinct == "" {
			selectClause = "select count(*)"
		} else {
			selectClause = fmt.Sprintf("select count(%s id)", distinct)
		}
	}

	// Building the where clause is the hardest part
	//printTree(tree, "")
	queryEntry, err := compileBaseQuery(table, q, optionsMap)
	if err != nil {
		return nil, err
	}
	if queryEntry == nil {
		return &Query{
			Select: selectClause,
			From:   fromClause,
		}, nil
	}

	pagination, err := getPaginationQuery(q.Pagination, table, optionsMap)
	if err != nil {
		return nil, err
	}

	return &Query{
		Select:     selectClause,
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

func joinWrap(baseTable, joinTable string, query *Query) string {
	//GetTableToTablePath
	pathElems := searchPkg.GetTableToTablePath(baseTable, joinTable)
	if len(pathElems) == 0 {
		// This means that there is only pointers from joinTable to baseTable (which for now is just ID)
		pathElems = searchPkg.GetTableToTablePath(joinTable, baseTable)
		if len(pathElems) == 0 {
			log.Fatalf("No existing path between table %s to %s", baseTable, joinTable)
		}
		path := pgsearch.GenerateShortestElemPath(joinTable, pathElems)
		query.Select = fmt.Sprintf("select distinct(%s)", pgsearch.RenderFinalPath(path, pathElems[len(pathElems)-1].ProtoJSONName))
		return fmt.Sprintf("%s.id in (%s)", baseTable, query.String())
	}
	path := pgsearch.GenerateShortestElemPath(baseTable, pathElems)
	return fmt.Sprintf("%s in (%s)", pgsearch.RenderFinalPath(path, pathElems[len(pathElems)-1].ProtoJSONName), query.String())
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
		if ok && queryTable != table {
			// Need to regen the whole query and join it
			query, err := populatePath(q, optionsMap, queryTable, false)
			if err != nil {
				return nil, err
			}
			return &pgsearch.QueryEntry{
				Query:  joinWrap(table, queryTable, query),
				Values: query.Data,
			}, nil
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

func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), false)
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
	defer rows.Close()

	var searchResults []searchPkg.Result
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		searchResults = append(searchResults, searchPkg.Result{
			ID: id,
		})
	}
	return searchResults, nil
}

func RunCountRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) (int, error) {
	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), true)
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
