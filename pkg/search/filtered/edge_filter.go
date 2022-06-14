package filtered

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/dackbox/keys"
)

type edgeFilter struct {
	filter Filter
}

// NewEdgeSourceFilter creates a filter that filters the first id of the composite edge id.
func NewEdgeSourceFilter(filter Filter) *edgeFilter {
	return &edgeFilter{
		filter: filter,
	}
}

func (e *edgeFilter) Apply(ctx context.Context, from ...string) ([]int, bool, error) {
	firstIDs := make([]string, 0, len(from))
	for _, edgeID := range from {
		firstID, err := keys.PairKeySelect([]byte(edgeID), 0)
		if err != nil {
			return nil, false, errors.Wrap(err, "decoding edge key")
		}
		firstIDs = append(firstIDs, string(firstID))

	}
	return e.filter.Apply(ctx, firstIDs...)
}
