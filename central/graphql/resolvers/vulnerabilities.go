package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
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
			"activeState(query: String): ActiveState",
			"vulnerabilityState: String!",
			"effectiveVulnerabilityRequest: VulnerabilityRequest",
		}),
	)
}

// VulnerabilityResolver represents a generic resolver of vulnerability fields.
// Values may come from either an embedded vulnerability context, or a top level vulnerability context.
type VulnerabilityResolver interface {
	ID(ctx context.Context) graphql.ID
	CVE(ctx context.Context) string
	Cvss(ctx context.Context) float64
	Link(ctx context.Context) string
	Summary(ctx context.Context) string
	EnvImpact(ctx context.Context) (float64, error)
	ImpactScore(ctx context.Context) float64
	ScoreVersion(ctx context.Context) string
	FixedByVersion(ctx context.Context) (string, error)
	IsFixable(ctx context.Context, args RawQuery) (bool, error)
	PublishedOn(ctx context.Context) (*graphql.Time, error)
	CreatedAt(ctx context.Context) (*graphql.Time, error)
	DiscoveredAtImage(ctx context.Context, args RawQuery) (*graphql.Time, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	LastModified(ctx context.Context) (*graphql.Time, error)
	Vectors() *EmbeddedVulnerabilityVectorsResolver
	Severity(ctx context.Context) string
	VulnerabilityType() string
	VulnerabilityTypes() []string

	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)

	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)

	Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error)
	NodeCount(ctx context.Context, args RawQuery) (int32, error)

	ClusterCount(ctx context.Context, args RawQuery) (int32, error)
	Clusters(ctx context.Context, args PaginatedQuery) ([]*clusterResolver, error)

	UnusedVarSink(ctx context.Context, args RawQuery) *int32

	Suppressed(ctx context.Context) bool
	SuppressActivation(ctx context.Context) (*graphql.Time, error)
	SuppressExpiry(ctx context.Context) (*graphql.Time, error)

	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)

	VulnerabilityState(ctx context.Context) string
	EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error)
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

	local := q.Clone()
	pagination := local.GetPagination()
	local.Pagination = nil
	local = search.ConjunctionQuery(local, search.NewQueryBuilder().AddBools(search.CVESuppressed, false).ProtoQuery())
	local.Pagination = pagination
	return local
}
