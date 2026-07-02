package resolvers

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedVulnerability", []string{
			"id: ID!",
			"cve: String!",
			"cvss: Float!",
			"scoreVersion: String!",
			"vectors: EmbeddedVulnerabilityVectors",
			"link: String!",
			"summary: String!",
			"fixedByVersion: String!",
			"isFixable(query: String): Boolean!",
			"lastScanned: Time",
			"createdAt: Time", // Discovered At System
			"discoveredAtImage(query: String): Time",
			"images(query: String, pagination: Pagination): [Image!]!",
			"imageCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"deploymentCount(query: String): Int!",
			"nodes(query: String, pagination: Pagination): [Node!]!",
			"nodeCount(query: String): Int!",
			"envImpact: Float!",
			"severity: String!",
			"publishedOn: Time",
			"lastModified: Time",
			"impactScore: Float!",
			"vulnerabilityType: String!",
			"vulnerabilityTypes: [String!]!",
			"suppressed: Boolean!",
			"suppressActivation: Time",
			"suppressExpiry: Time",
			"vulnerabilityState: String!",
			"effectiveVulnerabilityRequest: VulnerabilityRequest",
		}),
	)
}

func tryUnsuppressedQuery(q *v1.Query) *v1.Query {
	var isSearchBySuppressed, isSearchByVulnState bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.CVESuppressed.String() && mfQ.MatchFieldQuery.GetValue() == "true" {
			isSearchBySuppressed = true
			return
		}
		if ok && mfQ.MatchFieldQuery.GetField() == search.VulnerabilityState.String() {
			isSearchByVulnState = true
			return
		}
	})
	// If search query is explicitly requesting vulns by its observed state using the legacy way or the new way,
	// do not override with only unsnoozed cves query.
	if isSearchBySuppressed || isSearchByVulnState {
		return q
	}

	local := q.CloneVT()
	pagination := local.GetPagination()
	local.Pagination = nil
	local = search.ConjunctionQuery(local, search.NewQueryBuilder().AddBools(search.CVESuppressed, false).ProtoQuery())
	local.Pagination = pagination
	return local
}
