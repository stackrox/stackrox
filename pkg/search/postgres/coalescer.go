package postgres

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/pointers"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/walker"
)

type tableQuery struct {
	schema *walker.Schema

	and []*v1.Query
	or []*v1.Query
}


type coalescer struct {
	queriesByTable map[string]*tableQuery
}

func newCoalescer() *coalescer {
	return &coalescer{
		queriesByTable: make(map[string]*tableQuery),
	}
}

func (c *coalescer) schemaFromField(q *v1.MatchFieldQuery, optionsMap searchPkg.OptionsMap) *tableQuery {
	// Need to find base value
	field, ok := optionsMap.Get(q.GetField())
	if !ok {
		return nil
	}
	schema := field.DatabaseField.Schema
	table, ok := c.queriesByTable[schema.Table]
	if !ok {
		table = &tableQuery {
			schema: schema,
		}
		c.queriesByTable[schema.Table] = table
	}
	return table
}

func (c *coalescer) getTableQueryFromBase(q *v1.Query, optionsMap searchPkg.OptionsMap) (*tableQuery, error) {
	if q.GetBaseQuery() == nil {
		return nil, nil
	}
	switch subBQ := q.GetBaseQuery().Query.(type) {
	case *v1.BaseQuery_DocIdQuery:
		// nothing to do here
	case *v1.BaseQuery_MatchFieldQuery:
		return c.schemaFromField(subBQ.MatchFieldQuery, optionsMap), nil
	case *v1.BaseQuery_MatchNoneQuery:
		// nothing to here either
	case *v1.BaseQuery_MatchLinkedFieldsQuery:
		// Need to split this
		for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
			return c.schemaFromField(q.GetField(), optionsMap)
		}
	default:
		panic("unsupported")
	}
	return nil, nil
}

func (c *coalescer) populatePathRecursive(q *v1.Query, optionsMap searchPkg.OptionsMap) (*tableQuery, error) {
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
			schema := field.DatabaseField.Schema
			table, ok := c.queriesByTable[schema.Table]
			if !ok {
				table = &tableQuery {
					schema: schema,
				}
				c.queriesByTable[schema.Table] = table
			}
			return table, nil
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
		log.Fatalf("Boolean query not implemented: %+v", sub)
		for _, cq := range sub.BooleanQuery.Must.Queries {
			populatePathRecursive(tree, cq, optionsMap)
		}
		for _, dq := range sub.BooleanQuery.MustNot.Queries {
			populatePathRecursive(tree, dq, optionsMap)
		}
	}
}

func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *pgxpool.Pool, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
	// Validate search query doesn't break any join rules


	query, err := populatePath(q, optionsMap, mapping.GetTableFromCategory(category), GET)
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

	highlightedResults := make([]interface{}, len(query.Select.Fields)+1)
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
