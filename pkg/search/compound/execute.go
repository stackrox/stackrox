package compound

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/search"
)

func execute(ctx context.Context, tree *searchRequestSpec) ([]search.Result, error) {
	rs, err := executeRec(ctx, tree)
	if err != nil {
		return nil, err
	}
	return rs.asResultSlice(), nil
}

func executeRec(ctx context.Context, tree *searchRequestSpec) (resultSet, error) {
	if len(tree.and) > 0 {
		return executeAndRec(ctx, tree.and)
	} else if len(tree.or) > 0 {
		return executeOrRec(ctx, tree.or)
	} else if tree.boolean != nil {
		return executeBooleanRec(ctx, tree.boolean)
	} else if tree.base != nil {
		return executeBase(ctx, tree.base)
	}
	return resultSet{}, errors.New("search request tree empty")
}

func executeOrRec(ctx context.Context, ors []*searchRequestSpec) (resultSet, error) {
	var results resultSet
	for _, child := range ors {
		other, err := executeRec(ctx, child)
		if err != nil {
			return resultSet{}, err
		}
		if results.results == nil {
			results = other
		} else {
			results = results.union(other)
		}
	}
	return results, nil
}

func executeAndRec(ctx context.Context, ands []*searchRequestSpec) (resultSet, error) {
	var results resultSet
	for _, child := range ands {
		other, err := executeRec(ctx, child)
		if err != nil {
			return resultSet{}, err
		}
		if results.results == nil {
			results = other
		} else {
			results = results.intersect(other)
		}
	}
	return results, nil
}

func executeBooleanRec(ctx context.Context, boolean *booleanRequestSpec) (resultSet, error) {
	mustNots, err := executeRec(ctx, boolean.mustNot)
	if err != nil || len(mustNots.results) == 0 {
		return mustNots, err
	}
	musts, err := executeRec(ctx, boolean.must)
	if err != nil {
		return resultSet{}, err
	}
	return musts.subtract(mustNots), nil
}

func executeBase(ctx context.Context, base *baseRequestSpec) (resultSet, error) {
	results, err := base.Spec.Searcher.Search(ctx, base.Query)
	if err != nil {
		return resultSet{}, err
	}
	ordered := len(base.Query.GetPagination().GetSortOptions()) > 0
	return newResultSet(results, ordered), nil
}
