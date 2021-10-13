package pgsearch
//
//import (
//	"fmt"
//
//	"github.com/blevesearch/bleve"
//	"github.com/blevesearch/bleve/search/query"
//	v1 "github.com/stackrox/rox/generated/api/v1"
//	"github.com/stackrox/rox/pkg/search"
//)
//
//type queryConverter struct {
//	category     v1.SearchCategory
//	index        bleve.Index
//	optionsMap   search.OptionsMap
//	highlightCtx highlightContext
//}
//
//func (c *queryConverter) booleanQueryToBleve(ctx bleveContext, bq *v1.BooleanQuery) (query.Query, error) {
//	cq, err := c.conjunctionQueryToBleve(ctx, bq.Must)
//	if err != nil {
//		return nil, err
//	}
//	dq, err := c.disjunctionQueryToBleve(ctx, bq.MustNot)
//	if err != nil {
//		return nil, err
//	}
//	boolQuery := bleve.NewBooleanQuery()
//	boolQuery.AddMust(typeQuery(c.category))
//	boolQuery.AddMust(cq.(*query.ConjunctionQuery).Conjuncts...)
//
//	boolQuery.MustNot = dq
//	return boolQuery, nil
//}
