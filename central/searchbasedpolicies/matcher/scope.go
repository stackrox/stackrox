package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func scopeToQuery(scopes []*storage.Scope) *v1.Query {
	if len(scopes) == 0 {
		return nil
	}

	queries := make([]*v1.Query, 0, len(scopes))
	for _, s := range scopes {
		qb := search.NewQueryBuilder()
		if s.GetCluster() != "" {
			qb.AddExactMatches(search.ClusterID, s.GetCluster())
		}
		if s.GetNamespace() != "" {
			qb.AddExactMatches(search.Namespace, s.GetNamespace())
		}
		if s.GetLabel() != nil {
			qb.AddMapQuery(search.Label, s.GetLabel().GetKey(), s.GetLabel().GetValue())
		}
		queries = append(queries, qb.ProtoQuery())
	}

	return search.NewDisjunctionQuery(queries...)
}
