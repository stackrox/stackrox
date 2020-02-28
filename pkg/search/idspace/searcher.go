package idspace

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/search"
)

// WithKeyTransformations applies the input key transformations to the returns before output.
func WithKeyTransformations(searcher search.Searcher, transformers ...transformation.OneToMany) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		results, err := searcher.Search(ctx, q)
		if err != nil {
			return results, err
		}

		// Need to track the transformed results generated.
		transformed := make([]search.Result, 0, len(results))
		transformedIndices := make(map[string]int)

		// For each untransformed result...
		for _, result := range results {

			// For each transformer we wish to apply to it...
			for _, transformer := range transformers {

				// Generated the list of transformed results the original result corresponds to.
				newKeys := transformer(ctx, []byte(result.ID))
				for _, newKey := range newKeys {
					newKeyStr := string(newKey)

					// If a different transformed result already generated a result for this id, reference it, otherwise
					// add a new result for the id.
					var transformedIndex int
					if index, exists := transformedIndices[newKeyStr]; exists {
						transformedIndex = index
					} else {
						transformedIndex = len(transformed)
						transformedIndices[newKeyStr] = transformedIndex
						transformed = append(transformed, search.Result{
							ID: newKeyStr,
						})
					}

					// Merge the match and field values from the original result into the transformed result.
					mregeFieldsAndMatches(&transformed[transformedIndex], &result)
				}
			}
		}
		return transformed, nil
	})
}

func mregeFieldsAndMatches(to, from *search.Result) {
	if to.Matches == nil && from.Matches != nil {
		to.Matches = make(map[string][]string)
	}
	for k, vs := range from.Matches {
		if _, toHas := to.Matches[k]; toHas {
			to.Matches[k] = append(to.Matches[k], vs...)
		} else {
			to.Matches[k] = append([]string{}, vs...)
		}
	}

	if to.Fields == nil && from.Fields != nil {
		to.Fields = make(map[string]interface{})
	}
	for k, vs := range from.Fields {
		if _, toHas := to.Fields[k]; !toHas {
			to.Fields[k] = vs
		}
	}
}
