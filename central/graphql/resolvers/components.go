package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	componentPredicateFactory = predicate.NewFactory("component", &storage.EmbeddedImageScanComponent{})
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedImageScanComponent", []string{
			"license: License",
			"id: ID!",
			"name: String!",
			"version: String!",
			"topVuln: EmbeddedVulnerability",
			"vulns(query: String, pagination: Pagination): [EmbeddedVulnerability]!",
			"vulnCount(query: String): Int!",
			"vulnCounter(query: String): VulnerabilityCounter!",
			"lastScanned: Time",
			"images(query: String, pagination: Pagination): [Image!]!",
			"imageCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"deploymentCount(query: String): Int!",
			"priority: Int!",
			"source: String!",
			"location(query: String): String!",
			"riskScore: Float!",
		}),
		schema.AddExtraResolver("ImageScan", `components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("ImageScan", `componentCount(query: String): Int!`),
		schema.AddQuery("component(id: ID): EmbeddedImageScanComponent"),
		schema.AddQuery("components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!"),
		schema.AddQuery("componentCount(query: String): Int!"),
		schema.AddExtraResolver("EmbeddedImageScanComponent", `unusedVarSink(query: String): Int`),
	)
}

// ComponentResolver represents a generic resolver of component fields.
// Values may come from either an embedded component context, or a top level component context.
type ComponentResolver interface {
	ID(ctx context.Context) graphql.ID
	Name(ctx context.Context) string
	Version(ctx context.Context) string
	Priority(ctx context.Context) int32
	Source(ctx context.Context) string
	Location(ctx context.Context, args RawQuery) (string, error)
	LayerIndex() *int32
	LastScanned(ctx context.Context) (*graphql.Time, error)
	License(ctx context.Context) (*licenseResolver, error)
	RiskScore(ctx context.Context) float64

	TopVuln(ctx context.Context) (VulnerabilityResolver, error)
	Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error)
	VulnCount(ctx context.Context, args RawQuery) (int32, error)
	VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)

	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)

	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
}

// Component returns an image scan component based on an input id (name:version)
func (resolver *Resolver) Component(ctx context.Context, args idQuery) (ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		return resolver.componentV2(ctx, args)
	}
	return resolver.componentV1(ctx, args)
}

// Components returns the image scan components that match the input query.
func (resolver *Resolver) Components(ctx context.Context, q PaginatedQuery) ([]ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		return resolver.componentsV2(ctx, q)
	}
	return resolver.componentsV1(ctx, q)
}

// ComponentCount returns count of all clusters across infrastructure
func (resolver *Resolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ComponentCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	if features.Dackbox.Enabled() {
		return resolver.componentCountV2(ctx, args)
	}
	return resolver.componentCountV1(ctx, args)
}
