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
		schema.AddType("ImageVulnerability", []string{
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
			"envImpact: Float!",
			"severity: String!",
			"publishedOn: Time",
			"lastModified: Time",
			"impactScore: Float!",
			"suppressed: Boolean!",
			"suppressActivation: Time",
			"suppressExpiry: Time",
			"activeState(query: String): ActiveState",
			"vulnerabilityState: String!",
			"effectiveVulnerabilityRequest: VulnerabilityRequest",
			"unusedVarSink(query: String): Int",
		}),
		schema.AddQuery("imageVulnerability(id: ID): ImageVulnerability"),
		schema.AddQuery("imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability!]!"),
		schema.AddQuery("imageVulnerabilityCount(query: String): Int!"),
	)
}

// ImageVulnerabilityResolver represents the supported API on image vulnerabilities
type ImageVulnerabilityResolver interface {
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

// ImageVulnerability returns a vulnerability of the given id
func (resolver *Resolver) ImageVulnerability(ctx context.Context, args IDQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ImageVulnerability does not support postgres yet")
}

// ImageVulnerabilities resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilities(ctx context.Context, q PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withImageTypeFiltering(q.String())
		return resolver.imageVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ImageVulnerabilities does not support postgres yet")
}

// ImageVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withImageTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver ImageVulnerabilityCount does not support postgres yet")
}

// ImageVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withImageTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ImageVulnCounter does not support postgres yet")
}

// withImageTypeFiltering adds a conjunction as a raw query to filter vulnerability type by image
// this is needed to support pre postgres requests
func withImageTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_IMAGE_CVE.String()).Query())
}
