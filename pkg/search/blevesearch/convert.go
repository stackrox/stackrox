package blevesearch

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
)

// protoToBleveQuery converts our proto query format to the Bleve Query.
func protoToBleveQuery(q *v1.Query, category v1.SearchCategory, index bleve.Index, optionsMap map[string]*v1.SearchField) (bleveQuery query.Query, err error) {
	if q.GetQuery() == nil {
		return
	}
	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		return baseQueryToBleve(typedQ.BaseQuery, category, index, optionsMap)
	case *v1.Query_Conjunction:
		return conjunctionQueryToBleve(typedQ.Conjunction, category, index, optionsMap)
	case *v1.Query_Disjunction:
		return disjunctionQueryToBleve(typedQ.Disjunction, category, index, optionsMap)
	default:
		panic(fmt.Sprintf("Unhandled query type: %T", typedQ))
	}
}

func baseQueryToBleve(bq *v1.BaseQuery, category v1.SearchCategory, index bleve.Index, optionsMap map[string]*v1.SearchField) (bleveQuery query.Query, err error) {
	if bq.GetQuery() == nil {
		return
	}

	switch typedBQ := bq.GetQuery().(type) {
	case *v1.BaseQuery_StringQuery:
		if typedBQ.StringQuery.GetQuery() != "" {
			bleveQuery = NewMatchPhrasePrefixQuery("", typedBQ.StringQuery.GetQuery())
		}
		return
	case *v1.BaseQuery_MatchFieldQuery:
		return matchFieldQueryToBleve(typedBQ.MatchFieldQuery, category, index, optionsMap)
	default:
		panic(fmt.Sprintf("Unhandled base query type: %T", typedBQ))
	}
}

func matchFieldQueryToBleve(mq *v1.MatchFieldQuery, category v1.SearchCategory, index bleve.Index, optionsMap map[string]*v1.SearchField) (bleveQuery query.Query, err error) {
	if mq.GetField() == "" || mq.GetValue() == "" {
		return
	}
	searchField, found := optionsMap[mq.GetField()]
	if !found {
		return
	}
	bleveQuery, err = runSubQuery(index, category, searchField, mq.GetValue())
	return
}

func conjunctionQueryToBleve(cq *v1.ConjunctionQuery, category v1.SearchCategory, index bleve.Index, optionsMap map[string]*v1.SearchField) (query.Query, error) {
	if len(cq.GetQueries()) == 0 {
		return nil, nil
	}
	bleveSubQueries, err := getBleveSubQueries(cq.GetQueries(), category, index, optionsMap)
	if err != nil {
		return nil, err
	}
	if len(bleveSubQueries) == 0 {
		return nil, nil
	}
	return bleve.NewConjunctionQuery(bleveSubQueries...), nil
}

func disjunctionQueryToBleve(dq *v1.DisjunctionQuery, category v1.SearchCategory, index bleve.Index, optionsMap map[string]*v1.SearchField) (query.Query, error) {
	if len(dq.GetQueries()) == 0 {
		return nil, nil
	}

	bleveSubQueries, err := getBleveSubQueries(dq.GetQueries(), category, index, optionsMap)
	if err != nil {
		return nil, err
	}
	if len(bleveSubQueries) == 0 {
		return nil, nil
	}
	return bleve.NewDisjunctionQuery(bleveSubQueries...), nil
}

func getBleveSubQueries(qs []*v1.Query, category v1.SearchCategory, index bleve.Index, optionsMap map[string]*v1.SearchField) ([]query.Query, error) {
	bleveSubQueries := make([]query.Query, 0, len(qs))
	for _, q := range qs {
		bleveSubQuery, err := protoToBleveQuery(q, category, index, optionsMap)
		if err != nil {
			return nil, err
		}
		if bleveSubQuery == nil {
			continue
		}
		bleveSubQueries = append(bleveSubQueries, bleveSubQuery)
	}
	return bleveSubQueries, nil
}
