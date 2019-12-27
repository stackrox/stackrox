package idspace

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// TransformIDs applies a transformation to all of the ids of the results before returning them.
func TransformIDs(searcher search.Searcher, idTransformer func(string) (string, error)) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		results, err := searcher.Search(ctx, q)
		if err != nil {
			return results, err
		}

		errorlist := errorhelpers.NewErrorList("error transforming some ids")
		seenIds := set.NewStringSet()
		outputResults := make([]search.Result, 0, len(results))
		for idx := range results {
			newID, err := idTransformer(results[idx].ID)
			if err != nil {
				errorlist.AddError(err)
				continue
			}
			if seenIds.Contains(newID) {
				continue
			}
			seenIds.Add(newID)
			newResult := results[idx]
			newResult.ID = newID
			outputResults = append(outputResults, newResult)
		}
		return outputResults, errorlist.ToError()
	})
}

// EdgeIDToParentID gets the parent id from the input edge id.
func EdgeIDToParentID(edgeID string) (string, error) {
	paresedEdgeID, err := edges.FromString(edgeID)
	if err != nil {
		return "", err
	}
	return paresedEdgeID.ParentID, nil
}

// EdgeIDToChildID gets the child id from the input edge id.
func EdgeIDToChildID(edgeID string) (string, error) {
	paresedEdgeID, err := edges.FromString(edgeID)
	if err != nil {
		return "", err
	}
	return paresedEdgeID.ChildID, nil
}
