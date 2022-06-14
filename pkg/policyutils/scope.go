package policyutils

import (
	"fmt"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// ScopeToQuery generates a proto query for objects in the specified scopes.
func ScopeToQuery(scopes []*storage.Scope) *v1.Query {
	if len(scopes) == 0 {
		return search.EmptyQuery()
	}

	queries := make([]*v1.Query, 0, len(scopes))
	for _, s := range scopes {
		qb := search.NewQueryBuilder()
		if s.GetCluster() != "" {
			qb.AddExactMatches(search.ClusterID, s.GetCluster())
		}
		if s.GetNamespace() != "" {
			qb.AddRegexes(search.Namespace, s.GetNamespace())
		}
		if s.GetLabel() != nil {
			qb.AddMapQuery(search.Label, fmt.Sprintf("%s%s", search.RegexPrefix, s.GetLabel().GetKey()), fmt.Sprintf("%s%s", search.RegexPrefix, s.GetLabel().GetValue()))
		}
		queries = append(queries, qb.ProtoQuery())
	}

	return search.DisjunctionQuery(queries...)
}
