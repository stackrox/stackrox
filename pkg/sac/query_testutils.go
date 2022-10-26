package sac

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func clusterVerboseMatch(_ *testing.T, clusterID string) *v1.Query {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).MarkHighlighted(search.ClusterID)
	return query.ProtoQuery()
}

func clusterNonVerboseMatch(_ *testing.T, clusterID string) *v1.Query {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID)
	return query.ProtoQuery()
}

func namespaceVerboseMatch(_ *testing.T, namespace string) *v1.Query {
	query := search.NewQueryBuilder().AddExactMatches(search.Namespace, namespace).MarkHighlighted(search.Namespace)
	return query.ProtoQuery()
}

func namespaceNonVerboseMatch(_ *testing.T, namespace string) *v1.Query {
	query := search.NewQueryBuilder().AddExactMatches(search.Namespace, namespace)
	return query.ProtoQuery()
}

func isSameQuery(expected, actual *v1.Query) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	if !isSameQueryPagination(expected.GetPagination(), actual.GetPagination()) {
		return false
	}
	if expected.Query == nil && actual.Query == nil {
		return true
	}
	if expected.Query != nil && actual.Query == nil {
		return false
	}
	if expected.Query == nil && actual.Query != nil {
		return false
	}
	switch expected.Query.(type) {
	case *v1.Query_BaseQuery:
		switch actual.Query.(type) {
		case *v1.Query_BaseQuery:
			return isSameBaseQuery(expected.GetBaseQuery(), actual.GetBaseQuery())
		default:
			return false
		}
	case *v1.Query_BooleanQuery:
		switch actual.Query.(type) {
		case *v1.Query_BooleanQuery:
			return isSameBooleanQuery(expected.GetBooleanQuery(), actual.GetBooleanQuery())
		default:
			return false
		}
	case *v1.Query_Conjunction:
		switch actual.Query.(type) {
		case *v1.Query_Conjunction:
			return isSameConjunctionQuery(expected.GetConjunction(), actual.GetConjunction())
		default:
			return false
		}
	case *v1.Query_Disjunction:
		switch actual.Query.(type) {
		case *v1.Query_Disjunction:
			return isSameDisjunctionQuery(expected.GetDisjunction(), actual.GetDisjunction())
		default:
			return false
		}
	default:
		utils.Must(fmt.Errorf("Unexpected query type %T", expected.Query))
		return false
	}
}

func isSameBaseQuery(expected, actual *v1.BaseQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	switch expected.Query.(type) {
	case *v1.BaseQuery_DocIdQuery:
		switch actual.Query.(type) {
		case *v1.BaseQuery_DocIdQuery:
			return isSameDocIDQuery(expected.GetDocIdQuery(), actual.GetDocIdQuery())
		default:
			return false
		}
	case *v1.BaseQuery_MatchFieldQuery:
		switch actual.Query.(type) {
		case *v1.BaseQuery_MatchFieldQuery:
			return isSameMatchFieldQuery(expected.GetMatchFieldQuery(), actual.GetMatchFieldQuery())
		default:
			return false
		}
	case *v1.BaseQuery_MatchLinkedFieldsQuery:
		switch actual.Query.(type) {
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			return isSameMatchLinkedFieldsQuery(
				expected.GetMatchLinkedFieldsQuery(),
				actual.GetMatchLinkedFieldsQuery())
		default:
			return false
		}
	case *v1.BaseQuery_MatchNoneQuery:
		switch actual.Query.(type) {
		case *v1.BaseQuery_MatchNoneQuery:
			if expected.Query == nil && actual.Query == nil {
				return true
			}
			if expected.Query != nil && actual.Query != nil {
				return true
			}
			return false
		default:
			return false
		}
	default:
		utils.Must(fmt.Errorf("Unexpected base query type %T", expected.Query))
		return false
	}
}

func isSameDocIDQuery(expected, actual *v1.DocIDQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	expectedIDs := make(map[string]struct{})
	actualIDs := make(map[string]struct{})
	for _, id := range expected.GetIds() {
		expectedIDs[id] = struct{}{}
	}
	for _, id := range actual.GetIds() {
		actualIDs[id] = struct{}{}
	}
	if len(expectedIDs) != len(actualIDs) {
		return false
	}
	for id := range expectedIDs {
		if _, ok := actualIDs[id]; !ok {
			return false
		}
	}
	return true
}

func isSameMatchFieldQuery(expected, actual *v1.MatchFieldQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	if expected.GetField() != actual.GetField() {
		return false
	}
	if expected.GetValue() != actual.GetValue() {
		return false
	}
	if expected.GetHighlight() != actual.GetHighlight() {
		return false
	}
	return true
}

func isSameMatchLinkedFieldsQuery(expected, actual *v1.MatchLinkedFieldsQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	expectedSubQueries := expected.GetQuery()
	actualSubQueries := actual.GetQuery()
	if len(expectedSubQueries) != len(actualSubQueries) {
		return false
	}
	matchedActual := make([]bool, len(actualSubQueries))
	for _, expectedSubQuery := range expectedSubQueries {
		matched := false
		for ix, actualSubQuery := range actualSubQueries {
			if matchedActual[ix] {
				continue
			}
			if isSameMatchFieldQuery(expectedSubQuery, actualSubQuery) {
				matched = true
				matchedActual[ix] = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func isSameBooleanQuery(expected, actual *v1.BooleanQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	if !isSameConjunctionQuery(expected.GetMust(), actual.GetMust()) {
		return false
	}
	if !isSameDisjunctionQuery(expected.GetMustNot(), actual.GetMustNot()) {
		return false
	}
	return true
}

func isSameConjunctionQuery(expected, actual *v1.ConjunctionQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	expectedSubQueries := expected.GetQueries()
	actualSubQueries := actual.GetQueries()
	if len(expectedSubQueries) != len(actualSubQueries) {
		return false
	}
	matchedActual := make([]bool, len(actualSubQueries))
	for _, expectedSubQuery := range expectedSubQueries {
		matched := false
		for ix, actualSubQuery := range actualSubQueries {
			if matchedActual[ix] {
				continue
			}
			if isSameQuery(expectedSubQuery, actualSubQuery) {
				matched = true
				matchedActual[ix] = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func isSameDisjunctionQuery(expected, actual *v1.DisjunctionQuery) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	expectedSubQueries := expected.GetQueries()
	actualSubQueries := actual.GetQueries()
	if len(expectedSubQueries) != len(actualSubQueries) {
		return false
	}
	matchedActual := make([]bool, len(actualSubQueries))
	for _, expectedSubQuery := range expectedSubQueries {
		matched := false
		for ix, actualSubQuery := range actualSubQueries {
			if matchedActual[ix] {
				continue
			}
			if isSameQuery(expectedSubQuery, actualSubQuery) {
				matched = true
				matchedActual[ix] = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func isSameQueryPagination(expected, actual *v1.QueryPagination) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	if expected.GetLimit() != actual.GetLimit() {
		return false
	}
	if expected.GetOffset() != actual.GetOffset() {
		return false
	}
	expectedSortOptions := expected.GetSortOptions()
	actualSortOptions := actual.GetSortOptions()
	if len(expectedSortOptions) != len(actualSortOptions) {
		return false
	}
	// Order of sort options does matter.
	for i := range expectedSortOptions {
		if !isSameSortOption(expectedSortOptions[i], actualSortOptions[i]) {
			return false
		}
	}
	return true
}

func isSameSortOption(expected, actual *v1.QuerySortOption) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected != nil && actual == nil {
		return false
	}
	if expected == nil && actual != nil {
		return false
	}
	if expected.GetField() != actual.GetField() {
		return false
	}
	if expected.GetReversed() != actual.GetReversed() {
		return false
	}
	if expected.GetSearchAfter() != actual.GetSearchAfter() {
		return false
	}
	return true
}
