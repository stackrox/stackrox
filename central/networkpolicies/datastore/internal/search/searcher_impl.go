package search

import (
	"context"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	sacHelper = sac.ForResource(resources.NetworkPolicy).MustCreatePgSearchHelper()
)

// searcherImpl provides a search implementation for network policies.
type searcherImpl struct {
	index index.Indexer
}

// Count returns the number of search results from the query.
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return sacHelper.FilteredSearcher(s.index).Count(ctx, q)
}
