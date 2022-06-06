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
		schema.AddType("NodeComponent", []string{
			"fixedIn: String!", // is this needed?
			"id: ID!",
			"nodeComponentLastScanned: Time",
			"license: License",
			"location(query: String): String!",
			"name: String!",
			"nodes(query: String, scopeQuery: String, pagination: Pagination): [Node!]!",
			"nodeCount(query: String, scopeQuery: String): Int!",
			"nodeVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability]!",
			"nodeVulnerabilityCount(query: String, scopeQuery: String): Int!",
			"nodeVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"plottedNodeVulnerabilities(query: String): PlottedVulnerabilities!",
			"priority: Int!",
			"riskScore: Float!",
			"source: String!", // is this infrastructure ?
			"topNodeVulnerability: NodeVulnerability",
			"unusedVarSink(query: String): Int",
			"version: String!",
		}),

		schema.AddQuery("nodeComponent(id: ID): NodeComponent"),
		schema.AddQuery("nodeComponents(query: String, scopeQuery: String, pagination: Pagination): [NodeComponent!]!"),
		schema.AddQuery("nodeComponentCount(query: String): Int!"),
	)
}

// NodeComponentResolver represents a generic resolver of node component fields.
// Values may come from either an embedded component context, or a top level component context.
// NOTE: This list is and should remain alphabetically ordered
type NodeComponentResolver interface {
	FixedIn(ctx context.Context) string
	ID(ctx context.Context) graphql.ID
	NodeComponentLastScanned(ctx context.Context) (*graphql.Time, error)
	License(ctx context.Context) (*licenseResolver, error)
	Location(ctx context.Context, args RawQuery) (string, error)
	Name(ctx context.Context) string
	Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error)
	NodeCount(ctx context.Context, args RawQuery) (int32, error)
	NodeVulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error)
	NodeVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error)
	NodeVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)
	PlottedNodeVulnerabilities(ctx context.Context, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error)
	Priority(ctx context.Context) int32
	RiskScore(ctx context.Context) float64
	Source(ctx context.Context) string
	TopNodeVulnerability(ctx context.Context) (NodeVulnerabilityResolver, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Version(ctx context.Context) string
}

// NodeComponent returns a node component based on an input id (name:version)
func (resolver *Resolver) NodeComponent(ctx context.Context, args IDQuery) (NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeComponent")
	if !features.PostgresDatastore.Enabled() {
		return resolver.nodeComponentV2(ctx, args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Resolver NodeComponent does not support postgres yet")
}

// NodeComponents returns node components that match the input query.
func (resolver *Resolver) NodeComponents(ctx context.Context, q PaginatedQuery) ([]NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeComponents")
	if !features.PostgresDatastore.Enabled() {
		query := queryWithNodeIDRegexFilter(q.String())

		return resolver.nodeComponentsV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO : Add postgres support
	return nil, errors.New("Resolver NodeComponents does not support postgres yet")
}

// NodeComponentCount returns count of node components that match the input query
func (resolver *Resolver) NodeComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeComponentCount")
	if !features.PostgresDatastore.Enabled() {
		query := queryWithNodeIDRegexFilter(args.String())

		return resolver.componentCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO : Add postgres support
	return 0, errors.New("Resolver NodeComponentCount does not support postgres yet")
}

func queryWithNodeIDRegexFilter(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddRegexes(search.NodeID, ".+").Query())
}
