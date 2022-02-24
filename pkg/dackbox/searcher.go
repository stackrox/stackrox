package dackbox

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/dackbox/keys"
)

// Searcher is an interface to performing a search on a DackBox graph.
type Searcher interface {
	Search(ctx context.Context, id string) (bool, error)
}

type funcSearcher func(id string) (bool, error)

func (s funcSearcher) Search(_ context.Context, id string) (bool, error) {
	return s(id)
}

// EdgeSearcher returns a searcher that starts at an edge, taking the given component as the start.
// edgeComponentIndex must be 0 or 1.
func EdgeSearcher(ctx context.Context, searcher Searcher, edgeComponentIndex int) Searcher {
	if edgeComponentIndex < 0 || edgeComponentIndex > 1 {
		panic(errors.Errorf("invalid edge component index %d", edgeComponentIndex))
	}

	return funcSearcher(func(edgeID string) (bool, error) {
		component, err := keys.PairKeySelect([]byte(edgeID), edgeComponentIndex)
		if err != nil {
			return false, errors.Wrap(err, "decoding edge key")
		}
		return searcher.Search(ctx, string(component))
	})
}
