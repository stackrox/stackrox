package policyutils

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// DeploymentExclusionToQuery returns the proto query to get all excluded deployments
func DeploymentExclusionToQuery(exclusions []*storage.Exclusion) *v1.Query {
	var queries []*v1.Query
	for _, exclusion := range exclusions {
		subqueries := make([]*v1.Query, 0, 2)
		if exclusion.GetDeployment() != nil {
			if exclusion.GetDeployment().GetName() != "" {
				subqueries = append(subqueries, search.NewQueryBuilder().AddExactMatches(search.DeploymentName,
					exclusion.GetDeployment().GetName()).ProtoQuery())
			}
			if exclusion.GetDeployment().GetScope() != nil {
				subqueries = append(subqueries, ScopeToQuery([]*storage.Scope{exclusion.GetDeployment().GetScope()}))
			}

			if len(subqueries) == 0 {
				continue
			}

			queries = append(queries, search.ConjunctionQuery(subqueries...))
		}
	}

	if len(queries) == 0 {
		return search.MatchNoneQuery()
	}

	return search.DisjunctionQuery(queries...)
}
