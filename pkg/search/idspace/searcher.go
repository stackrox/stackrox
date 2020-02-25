package idspace

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Transformer represents a process of converting from one id-space to another.
type Transformer interface {
	Transform(from ...string) ([]string, error)
}

// TransformIDs applies a transformation to all of the ids of the results before returning them.
func TransformIDs(searcher search.Searcher, transformer Transformer) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		results, err := searcher.Search(ctx, q)
		if err != nil {
			return results, err
		}

		transformed, err := transformer.Transform(search.ResultsToIDs(results)...)
		if err != nil {
			return results, err
		}

		transformedResults := make([]search.Result, len(transformed))
		for index, id := range transformed {
			transformedResults[index].ID = id
		}
		return transformedResults, nil
	})
}
