package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
)

var (
	categoryToTableMap = make(map[v1.SearchCategory]string)

	log = logging.LoggerForModule()
)

func RegisterCategoryToTable(category v1.SearchCategory, table string) {
	if val, ok := categoryToTableMap[category]; ok {
		log.Fatalf("Cannot register category %s with table %s, it is already registered with %s", category, table, val)
	}
	categoryToTableMap[category] = table
}

type queryTree struct {
	elem searchPkg.PathElem
	children map[string]*queryTree
}

func newQueryTree(elem searchPkg.PathElem) *queryTree  {
	return &queryTree{
		elem: elem,
		children: make(map[string]*queryTree),
	}
}

func (q *queryTree) addElems(elems []searchPkg.PathElem) {
	currTree := q
	for _, e := range elems {
		var ok bool
		childTree, ok := currTree.children[e.Name]
		if !ok {
			childTree = newQueryTree(e)
			currTree.children[e.Name] = childTree
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
	fmt.Println(indent, t.elem.Name, t.elem.Slice)
	for _, children := range t.children {
		printTree(children, indent+"  ")
	}
}

func createFromClauseRecursive(t *queryTree, parent string) []string {
	var results []string
	if parent == "" {
		results = append(results, t.elem.Name)
		parent = fmt.Sprintf("%s.value", t.elem.Name)
	}

	for _, childTree := range t.children {
		if len(childTree.children) == 0 {
			if childTree.elem.Slice {
				results = append(results, fmt.Sprintf("jsonb_array_elements_text(%s->'%s') %s", parent, childTree.elem.Name, childTree.elem.Name))
			}
			continue
		}
		localParent := parent
		if childTree.elem.Slice {
			results = append(results, fmt.Sprintf("jsonb_array_elements(%s->'%s') %s", parent, childTree.elem.Name, childTree.elem.Name))
			localParent = childTree.elem.Name
		} else {
			localParent = fmt.Sprintf("%s->'%s'", parent, childTree.elem.Name)
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

func populatePath(q *v1.Query, optionsMap searchPkg.OptionsMap, table string, count bool) (string, []interface{}, error) {
	tree := newQueryTree(searchPkg.PathElem{
		Name:  table,
	})
	populatePathRecursive(tree, q, optionsMap)
	fromClause := createFROMClause(tree)

	// Initial select, need to support highlights as well
	selectClause := "select distinct id"
	if count {
		selectClause = "select count(distinct id)"
	}

	// Building the where clause is the hardest part
	//printTree(tree, "")
	queryEntry, err := compileBaseQuery(table, q, optionsMap)
	if err != nil {
		return "", nil, err
	}
	if queryEntry == nil {
		return fmt.Sprintf("%s %s;", selectClause, fromClause), nil, nil
	}
	query := fmt.Sprintf("%s %s where %s;", selectClause, fromClause, queryEntry.Query)
	return replaceVars(query), queryEntry.Values, nil
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

func compileBaseQuery(table string, q *v1.Query, optionsMap searchPkg.OptionsMap) (*pgsearch.QueryEntry, error) {
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
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

func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *sql.DB, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
	query, data, err := populatePath(q, optionsMap, categoryToTableMap[category], false)
	if err != nil {
		return nil, err
	}
	t := time.Now()
	defer func() {
		log.Infof("Took %d milliseconds to run: %s %+v", time.Since(t).Milliseconds(), query, data)
	}()
	rows, err := db.Query(query, data...)
	if err != nil {
		log.Errorf("Query issue: %s %+v: %v", query, data, err)
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
			ID:      id,
		})
	}
	return searchResults, nil
}

func RunCountRequest(category v1.SearchCategory, q *v1.Query, db *sql.DB, optionsMap searchPkg.OptionsMap) (int, error) {
	query, data, err := populatePath(q, optionsMap, categoryToTableMap[category], true)
	if err != nil {
		return 0, err
	}
	t := time.Now()
	defer func() {
		log.Infof("Took %d milliseconds to run: %s %+v", time.Since(t).Milliseconds(), query, data)
	}()
	row := db.QueryRow(query, data...)
	if err := row.Err(); err != nil {
		log.Errorf("Query issue: %s %+v: %v", query, data, err)
		return 0, err
	}
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
