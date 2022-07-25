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
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("EmbeddedImageScanComponent", []string{
			"activeState(query: String): ActiveState",
			"deploymentCount(query: String, scopeQuery: String): Int!",
			"deployments(query: String, scopeQuery: String, pagination: Pagination): [Deployment!]!",
			"fixedIn: String!",
			"id: ID!",
			"imageCount(query: String, scopeQuery: String): Int!",
			"images(query: String, scopeQuery: String, pagination: Pagination): [Image!]!",
			"lastScanned: Time",
			"layerIndex: Int!",
			"license: License",
			"location(query: String): String!",
			"name: String!",
			"nodeCount(query: String, scopeQuery: String): Int!",
			"nodes(query: String, scopeQuery: String, pagination: Pagination): [Node!]!",
			"priority: Int!",
			"riskScore: Float!",
			"source: String!",
			"topVuln: EmbeddedVulnerability",
			"version: String!",
			"vulnCount(query: String, scopeQuery: String): Int!",
			"vulnCounter(query: String): VulnerabilityCounter!",
			"vulns(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedVulnerability]!",
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
	LayerIndex() (int32, error)
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
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("Component is not supported with postgres, please use Image/NodeComponent")
	}
	return resolver.componentV2(ctx, args)
}

// Components returns the image scan components that match the input query.
func (resolver *Resolver) Components(ctx context.Context, q PaginatedQuery) ([]ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("Components is not supported with postgres, please use Image/NodeComponents")
	}
	return resolver.componentsV2(ctx, q)
}

// ComponentCount returns count of all clusters across infrastructure
func (resolver *Resolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ComponentCount")
	if features.PostgresDatastore.Enabled() {
		return 0, errors.New("ComponentCount is not supported with postgres, please use Image/NodeComponentCount")
	}
	return resolver.componentCountV2(ctx, args)
}
