package idspace

import (
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

// NewEdgeToParentTransformer returns a new transformer that transforms results into the edge's parent's id space.
func NewEdgeToParentTransformer() Transformer {
	return &edgeTransformerImpl{parser: edgeIDToParentID}
}

// NewEdgeToChildTransformer returns a new transformer that transforms results into the edge's child's id space.
func NewEdgeToChildTransformer() Transformer {
	return &edgeTransformerImpl{parser: edgeIDToChildID}
}

type edgeTransformerImpl struct {
	parser func(edgeID string) (string, error)
}

func (et *edgeTransformerImpl) Transform(from ...string) ([]string, error) {
	errorsList := errorhelpers.NewErrorList("error transforming some ids")
	seenIds := set.NewStringSet()
	ret := make([]string, 0, len(from))
	for _, id := range from {
		newID, err := et.parser(id)
		if err != nil {
			errorsList.AddError(err)
			continue
		}
		if seenIds.Contains(newID) {
			continue
		}
		seenIds.Add(newID)
		ret = append(ret, newID)
	}
	return ret, errorsList.ToError()
}

func edgeIDToParentID(edgeID string) (string, error) {
	paresedEdgeID, err := edges.FromString(edgeID)
	if err != nil {
		return "", err
	}
	return paresedEdgeID.ParentID, nil
}

// EdgeIDToChildID gets the child id from the input edge id.
func edgeIDToChildID(edgeID string) (string, error) {
	paresedEdgeID, err := edges.FromString(edgeID)
	if err != nil {
		return "", err
	}
	return paresedEdgeID.ChildID, nil
}
