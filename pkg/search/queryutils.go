package search

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
)

// ApplyFnToAllBaseQueries walks recursively over the query, applying fn to all the base queries.
func ApplyFnToAllBaseQueries(q *v1.Query, fn func(*v1.BaseQuery)) {
	if q.GetQuery() == nil {
		return
	}

	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_Disjunction:
		for _, subQ := range typedQ.Disjunction.GetQueries() {
			ApplyFnToAllBaseQueries(subQ, fn)
		}
	case *v1.Query_Conjunction:
		for _, subQ := range typedQ.Conjunction.GetQueries() {
			ApplyFnToAllBaseQueries(subQ, fn)
		}
	case *v1.Query_BooleanQuery:
		for _, subQ := range typedQ.BooleanQuery.GetMust().GetQueries() {
			ApplyFnToAllBaseQueries(subQ, fn)
		}
		for _, subQ := range typedQ.BooleanQuery.GetMustNot().GetQueries() {
			ApplyFnToAllBaseQueries(subQ, fn)
		}
	case *v1.Query_BaseQuery:
		fn(typedQ.BaseQuery)
	default:
		utils.Should(fmt.Errorf("unhandled query type: %T; query was %s", q, proto.MarshalTextString(q)))
	}
}

// FilterQueryWithMap removes match fields portions of the query that are not in the input options map.
func FilterQueryWithMap(q *v1.Query, optionsMap OptionsMap) (*v1.Query, bool) {
	var areFieldsFiltered bool
	filtered, _ := FilterQuery(q, func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok {
			if _, isValid := optionsMap.Get(matchFieldQuery.MatchFieldQuery.GetField()); isValid {
				return true
			}
		}
		areFieldsFiltered = true
		return false
	})
	return filtered, areFieldsFiltered
}

// InverseFilterQueryWithMap removes match fields portions of the query that are in the input options map.
func InverseFilterQueryWithMap(q *v1.Query, optionsMap OptionsMap) (*v1.Query, bool) {
	var areFieldsFiltered bool
	filtered, _ := FilterQuery(q, func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok {
			if _, isValid := optionsMap.Get(matchFieldQuery.MatchFieldQuery.GetField()); !isValid {
				areFieldsFiltered = true
				return true
			}
		}
		return false
	})
	return filtered, areFieldsFiltered
}

// AddAsConjunction adds the input toAdd query to the input addTo query at the top level, either by appending it to the
// conjunction list, or, if it is a base query, by making it a conjunction. Explicity disallows nested queries, as the
// resulting query is expected to be either a base query, or a flat query.
func AddAsConjunction(toAdd *v1.Query, addTo *v1.Query) (*v1.Query, error) {
	if addTo.Query == nil {
		return toAdd, nil
	}
	switch typedQ := addTo.GetQuery().(type) {
	case *v1.Query_Conjunction:
		typedQ.Conjunction.Queries = append(typedQ.Conjunction.Queries, toAdd)
		return addTo, nil
	case *v1.Query_BaseQuery, *v1.Query_Disjunction:
		return ConjunctionQuery(toAdd, addTo), nil
	default:
		return nil, errors.New("cannot add to a non-nil, non-conjunction/disjunction, non-base query")
	}
}

// FilterQuery applies the given function on every base query, and returns a new
// query that has only the sub-queries that the function returns true for.
// It will NOT mutate q unless the function passed mutates its argument.
func FilterQuery(q *v1.Query, fn func(*v1.BaseQuery) bool) (*v1.Query, bool) {
	if q.GetQuery() == nil {
		return nil, false
	}
	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_Disjunction:
		filteredQueries := filterQueriesByFunction(typedQ.Disjunction.GetQueries(), fn)
		if len(filteredQueries) == 0 {
			return nil, false
		}
		return DisjunctionQuery(filteredQueries...), true
	case *v1.Query_Conjunction:
		filteredQueries := filterQueriesByFunction(typedQ.Conjunction.GetQueries(), fn)
		if len(filteredQueries) == 0 {
			return nil, false
		}
		return ConjunctionQuery(filteredQueries...), true
	case *v1.Query_BaseQuery:
		if fn(typedQ.BaseQuery) {
			return q, true
		}
		return nil, false
	default:
		log.Errorf("Unhandled query type: %T; query was %s", q, proto.MarshalTextString(q))
		return nil, false
	}
}

// Helper function used by FilterQuery.
func filterQueriesByFunction(qs []*v1.Query, fn func(*v1.BaseQuery) bool) (filteredQueries []*v1.Query) {
	for _, q := range qs {
		filteredQuery, found := FilterQuery(q, fn)
		if found {
			filteredQueries = append(filteredQueries, filteredQuery)
		}
	}
	return
}

// AddRawQueriesAsConjunction adds the input toAdd raw query to the input addTo raw query
func AddRawQueriesAsConjunction(toAdd string, addTo string) string {
	if toAdd == "" && addTo == "" {
		return ""
	}

	if addTo == "" {
		return toAdd
	}

	if toAdd == "" {
		return addTo
	}

	return addTo + "+" + toAdd
}
