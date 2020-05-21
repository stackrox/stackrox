package policyutils

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// DeploymentWhitelistToQuery returns the proto query to get all whiteisted deployments
func DeploymentWhitelistToQuery(whitelists []*storage.Whitelist) *v1.Query {
	var queries []*v1.Query
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

			if len(subqueries) == 0 {
				continue
			}

			queries = append(queries, search.NewConjunctionQuery(subqueries...))
		}
	}

	if len(queries) == 0 {
		return search.MatchNoneQuery()
	}

	return search.NewDisjunctionQuery(queries...)
}
