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
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ImageVulnerability",
			append(commonVulnerabilitySubResolvers,
				"activeState(query: String): ActiveState",
				"imageComponentCount(query: String): Int!",
				"imageComponents(query: String, pagination: Pagination): [ImageComponent!]!",
				"effectiveVulnerabilityRequest: VulnerabilityRequest",
				"deploymentCount(query: String): Int!",
				"deployments(query: String, pagination: Pagination): [Deployment!]!",
				"discoveredAtImage(query: String): Time",
				"imageCount(query: String): Int!",
				"images(query: String, pagination: Pagination): [Image!]!",
			)),
		schema.AddQuery("imageVulnerability(id: ID): ImageVulnerability"),
		schema.AddQuery("imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability!]!"),
		schema.AddQuery("imageVulnerabilityCount(query: String): Int!"),
	)
}

// ImageVulnerabilityResolver represents the supported API on image vulnerabilities
//  NOTE: This list is and should remain alphabetically ordered
type ImageVulnerabilityResolver interface {
	CommonVulnerabilityResolver

	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error)
	ImageComponentCount(ctx context.Context, args RawQuery) (int32, error)
	EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DiscoveredAtImage(ctx context.Context, args RawQuery) (*graphql.Time, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)
	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
}

// ImageVulnerability returns a vulnerability of the given id
func (resolver *Resolver) ImageVulnerability(ctx context.Context, args IDQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.imageVulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ImageVulnerability does not support postgres yet")
}

// ImageVulnerabilities resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilities(ctx context.Context, q PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withImageCveTypeFiltering(q.String())
		return resolver.imageVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ImageVulnerabilities does not support postgres yet")
}

// ImageVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withImageCveTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver ImageVulnerabilityCount does not support postgres yet")
}

// ImageVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withImageCveTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ImageVulnCounter does not support postgres yet")
}

// withImageCveTypeFiltering adds a conjunction as a raw query to filter vulnerability type by image
// this is needed to support pre postgres requests
func withImageCveTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_IMAGE_CVE.String()).Query())
}
