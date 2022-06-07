package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddExtraResolvers("ImageComponent", []string{
			"activeState(query: String): ActiveState",
			"deploymentCount(query: String, scopeQuery: String): Int!",
			"deployments(query: String, scopeQuery: String, pagination: Pagination): [Deployment!]!",
			"fixedIn: String!",
			"imageCount(query: String, scopeQuery: String): Int!",
			"images(query: String, scopeQuery: String, pagination: Pagination): [Image!]!",
			"imageVulnerabilityCount(query: String, scopeQuery: String): Int!",
			"imageVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability]!",
			"lastScanned: Time",
			"location(query: String): String!",
			"topImageVulnerability: ImageVulnerability",
			"unusedVarSink(query: String): Int",

			"plottedVulns(query: String): PlottedVulnerabilities!", // TODO
		}),
		schema.AddQuery("imageComponent(id: ID): ImageComponent"),
		schema.AddQuery("imageComponents(query: String, scopeQuery: String, pagination: Pagination): [ImageComponent!]!"),
		schema.AddQuery("imageComponentCount(query: String): Int!"),

		// TODO
		schema.AddExtraResolver("ImageScan", `components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("ImageScan", `componentCount(query: String): Int!`),
	)
}

// ImageComponentResolver represents a generic resolver of image component fields.
// Values may come from either an embedded component context, or a top level component context.
// NOTE: This list is and should remain alphabetically ordered
type ImageComponentResolver interface {
	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	FixedIn(ctx context.Context) string
	FixedBy(ctx context.Context) string
	ID(ctx context.Context) graphql.ID
	ImageCount(ctx context.Context, args RawQuery) (int32, error)
	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error)
	ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)
	ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	LayerIndex() *int32
	License(ctx context.Context) (*licenseResolver, error)
	Location(ctx context.Context, args RawQuery) (string, error)
	Name(ctx context.Context) string
	OperatingSystem(ctx context.Context) string
	Priority(ctx context.Context) int32
	RiskScore(ctx context.Context) float64
	Source(ctx context.Context) string
	TopImageVulnerability(ctx context.Context) (ImageVulnerabilityResolver, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Version(ctx context.Context) string

	// TODO
	//PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error)
}

// ImageComponent returns an image component based on an input id (name:version)
func (resolver *Resolver) ImageComponent(ctx context.Context, args IDQuery) (ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")
	if !features.PostgresDatastore.Enabled() {
		return resolver.imageComponentV2(ctx, args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Resolver ImageComponent does not support postgres yet")
}

// ImageComponents returns image components that match the input query.
func (resolver *Resolver) ImageComponents(ctx context.Context, q PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	if !features.PostgresDatastore.Enabled() {
		query := queryWithImageIDRegexFilter(q.String())

		return resolver.imageComponentsV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO : Add postgres support
	return nil, errors.New("Resolver ImageComponents does not support postgres yet")
}

// ImageComponentCount returns count of image components that match the input query
func (resolver *Resolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponentCount")
	if !features.PostgresDatastore.Enabled() {
		query := queryWithImageIDRegexFilter(args.String())

		return resolver.componentCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO : Add postgres support
	return 0, errors.New("Resolver ImageComponentCount does not support postgres yet")
}

func queryWithImageIDRegexFilter(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddRegexes(search.ImageLabel, ".+").Query())
}
