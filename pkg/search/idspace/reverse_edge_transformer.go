package idspace

import (
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

// NewChildToEdgeTransformer returns a new transformer that transforms results from child idspace to edge id space.
func NewChildToEdgeTransformer(graphProvider GraphProvider, prefixPath [][]byte) Transformer {
	return &reverseEdgeTransformerImpl{
		parentToEdge:     false,
		graphTransformer: NewBackwardGraphTransformer(graphProvider, prefixPath),
	}
}

// NewParentToEdgeTransformer returns a new transformer that transforms results from parent idspace to edge id space.
func NewParentToEdgeTransformer(graphProvider GraphProvider, prefixPath [][]byte) Transformer {
	return &reverseEdgeTransformerImpl{
		parentToEdge:     true,
		graphTransformer: NewForwardGraphTransformer(graphProvider, prefixPath),
	}
}

type reverseEdgeTransformerImpl struct {
	parentToEdge     bool
	graphTransformer Transformer
}

func (et *reverseEdgeTransformerImpl) Transform(from ...string) ([]string, error) {
	errorsList := errorhelpers.NewErrorList("error transforming some ids")
	seenIds := set.NewStringSet()
	ret := make([]string, 0, len(from))
	for _, id := range from {
		newIDs, err := et.graphTransformer.Transform(id)
		if err != nil {
			errorsList.AddError(err)
			continue
		}

		for _, newID := range newIDs {
			var edge string
			if et.parentToEdge {
				edge = edges.EdgeID{
					ParentID: id,
					ChildID:  newID,
				}.ToString()
			} else {
				edge = edges.EdgeID{
					ParentID: newID,
					ChildID:  id,
				}.ToString()
			}

			if seenIds.Contains(edge) {
				continue
			}
			seenIds.Add(newID)
			ret = append(ret, edge)
		}
	}
	return ret, errorsList.ToError()
}
