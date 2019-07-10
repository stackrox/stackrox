package policyutils

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// DeploymentWhitelistToQuery returns the proto query to get all whiteisted deployments
func DeploymentWhitelistToQuery(whitelists []*storage.Whitelist) *v1.Query {
	if len(whitelists) == 0 {
		return search.EmptyQuery()
	}

	queries := make([]*v1.Query, 0, len(whitelists))
	for _, wl := range whitelists {
		subqueries := make([]*v1.Query, 0, 2)
		if wl.GetDeployment() != nil {
			if wl.GetDeployment().GetName() != "" {
				subqueries = append(subqueries, search.NewQueryBuilder().AddExactMatches(search.DeploymentName,
					wl.GetDeployment().GetName()).ProtoQuery())
			}
			if wl.GetDeployment().GetScope() != nil {
				subqueries = append(subqueries, ScopeToQuery([]*storage.Scope{wl.GetDeployment().GetScope()}))
			}

			queries = append(queries, search.NewConjunctionQuery(subqueries...))
		}
	}

	return search.NewDisjunctionQuery(queries...)
}
