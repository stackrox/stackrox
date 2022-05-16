package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ClusterVulnerability", []string{ // TODO pruning
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
			"components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!",
			"componentCount(query: String): Int!",
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
		schema.AddQuery("clusterVulnerability(id: ID): ClusterVulnerability"),
		schema.AddQuery("clusterVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ClusterVulnerability!]!"),
		schema.AddQuery("clusterVulnerabilityCount(query: String): Int!"),
	)
}

// ClusterVulnerabilityResolver represents the supported API on image vulnerabilities TODO pruning
type ClusterVulnerabilityResolver interface {
	ID(ctx context.Context) graphql.ID
	CVE(ctx context.Context) string
	Cvss(ctx context.Context) float64
	ScoreVersion(ctx context.Context) string
	Vectors() *EmbeddedVulnerabilityVectorsResolver
	Link(ctx context.Context) string
	Summary(ctx context.Context) string
	FixedByVersion(ctx context.Context) (string, error)
	IsFixable(ctx context.Context, args RawQuery) (bool, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	CreatedAt(ctx context.Context) (*graphql.Time, error)
	DiscoveredAtImage(ctx context.Context, args RawQuery) (*graphql.Time, error)
	Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error)
	ComponentCount(ctx context.Context, args RawQuery) (int32, error)
	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)
	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	EnvImpact(ctx context.Context) (float64, error)
	Severity(ctx context.Context) string
	PublishedOn(ctx context.Context) (*graphql.Time, error)
	LastModified(ctx context.Context) (*graphql.Time, error)
	ImpactScore(ctx context.Context) float64
	Suppressed(ctx context.Context) bool
	SuppressActivation(ctx context.Context) (*graphql.Time, error)
	SuppressExpiry(ctx context.Context) (*graphql.Time, error)
	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	VulnerabilityState(ctx context.Context) string
	EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
}

// ClusterVulnerability returns a vulnerability of the given id
func (resolver *Resolver) ClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ClusterVulnerability does not support postgres yet")
}

// ClusterVulnerabilities resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ClusterVulnerabilities(ctx context.Context, q PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withClusterTypeFiltering(q.String())
		return resolver.clusterVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ClusterVulnerabilities does not support postgres yet")
}

// ClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withClusterTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver ClusterVulnerabilityCount does not support postgres yet")
}

// ClusterVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ClusterVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withClusterTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ClusterVulnerabilityCounter does not support postgres yet")
}

// withClusterTypeFiltering adds a conjunction as a raw query to filter vulnerability type by cluster
// this is needed to support pre postgres requests
func withClusterTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType,
			storage.CVE_ISTIO_CVE.String(),
			storage.CVE_OPENSHIFT_CVE.String(),
			storage.CVE_K8S_CVE.String()).Query())
}
