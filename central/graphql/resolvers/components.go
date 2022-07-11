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
			"vulns(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedVulnerability]!",
			"vulnCount(query: String, scopeQuery: String): Int!",
			"vulnCounter(query: String): VulnerabilityCounter!",
			"lastScanned: Time",
			"images(query: String, scopeQuery: String, pagination: Pagination): [Image!]!",
			"imageCount(query: String, scopeQuery: String): Int!",
			"deployments(query: String, scopeQuery: String, pagination: Pagination): [Deployment!]!",
			"deploymentCount(query: String, scopeQuery: String): Int!",
			"activeState(query: String): ActiveState",
			"nodes(query: String, scopeQuery: String, pagination: Pagination): [Node!]!",
			"nodeCount(query: String, scopeQuery: String): Int!",
			"priority: Int!",
			"source: String!",
			"location(query: String): String!",
			"riskScore: Float!",
			"fixedIn: String!",
		}),
		schema.AddExtraResolver("ImageScan", `components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("ImageScan", `componentCount(query: String): Int!`),
		schema.AddQuery("component(id: ID): EmbeddedImageScanComponent"+
			"@deprecated(reason: \"use 'imageComponent' or 'nodeComponent'\")"),
		schema.AddQuery("components(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedImageScanComponent!]!"+
			"@deprecated(reason: \"use 'imageComponents' or 'nodeComponents'\")"),
		schema.AddQuery("componentCount(query: String): Int!"+
			"@deprecated(reason: \"use 'imageComponentCount' or 'nodeComponentCount'\")"),
		schema.AddExtraResolver("EmbeddedImageScanComponent", `unusedVarSink(query: String): Int`),
		schema.AddExtraResolver("EmbeddedImageScanComponent", "plottedVulns(query: String): PlottedVulnerabilities!"),
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
	FixedIn(ctx context.Context) string

	TopVuln(ctx context.Context) (VulnerabilityResolver, error)
	Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error)
	VulnCount(ctx context.Context, args RawQuery) (int32, error)
	VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)

	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)

	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)

	Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error)
	NodeCount(ctx context.Context, args RawQuery) (int32, error)

	PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error)

	UnusedVarSink(ctx context.Context, args RawQuery) *int32
}

// Component returns an image scan component based on an input id (name:version)
func (resolver *Resolver) Component(ctx context.Context, args IDQuery) (ComponentResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("Component is not supported with postgres, please use Image/NodeComponent")
	}
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")
	return resolver.componentV2(ctx, args)
}

// Components returns the image scan components that match the input query.
func (resolver *Resolver) Components(ctx context.Context, q PaginatedQuery) ([]ComponentResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("Components is not supported with postgres, please use Image/NodeComponents")
	}
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	return resolver.componentsV2(ctx, q)
}

// ComponentCount returns count of all clusters across infrastructure
func (resolver *Resolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	if features.PostgresDatastore.Enabled() {
		return 0, errors.New("ComponentCount is not supported with postgres, please use Image/NodeComponentCount")
	}
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ComponentCount")
	return resolver.componentCountV2(ctx, args)
}
