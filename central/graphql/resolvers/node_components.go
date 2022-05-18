package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list should be alphabetically ordered
		schema.AddType("NodeComponent", []string{
			"fixedIn: String!", // is this needed?
			"id: ID!",
			"lastScanned: Time",
			"license: License",
			"location(query: String): String!",
			"name: String!",
			"nodes(query: String, scopeQuery: String, pagination: Pagination): [Node!]!",
			"nodeCount(query: String, scopeQuery: String): Int!",
			"plottedVulnerabilities(query: String): PlottedVulnerabilities!",
			"priority: Int!",
			"riskScore: Float!",
			"source: String!", // is this infrastructure ?
			"topVulnerability: NodeVulnerability",
			"unusedVarSink(query: String): Int",
			"version: String!",
			"vulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability]!",
			"vulnerabilityCount(query: String, scopeQuery: String): Int!",
			"vulnerabilityCounter(query: String): VulnerabilityCounter!",
		}),
		// TODO : Tie this to NodeScan resolvers
		// schema.AddExtraResolver("NodeScan", `nodeComponents(query: String, pagination: Pagination): [NodeComponent!]!`),
		// schema.AddExtraResolver("NodeScan", `componentCount(query: String): Int!`),
		schema.AddQuery("nodeComponent(id: ID): NodeComponent"),
		schema.AddQuery("nodeComponents(query: String, scopeQuery: String, pagination: Pagination): [NodeComponent!]!"),
		schema.AddQuery("nodeComponentCount(query: String): Int!"),
	)
}

// NodeComponentResolver represents a generic resolver of node component fields.
// Values may come from either an embedded component context, or a top level component context.
// NOTE: This list should be alphabetically ordered
type NodeComponentResolver interface {
	FixedIn(ctx context.Context) string
	ID(ctx context.Context) graphql.ID
	LastScanned(ctx context.Context) (*graphql.Time, error)
	License(ctx context.Context) (*licenseResolver, error)
	Location(ctx context.Context, args RawQuery) (string, error)
	Name(ctx context.Context) string
	Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error)
	NodeCount(ctx context.Context, args RawQuery) (int32, error)
	PlottedVulnerabilities(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error)
	Priority(ctx context.Context) int32
	RiskScore(ctx context.Context) float64
	Source(ctx context.Context) string
	TopVulnerability(ctx context.Context) (NodeVulnerabilityResolver, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Version(ctx context.Context) string
	Vulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error)
	VulnerabilityCount(ctx context.Context, args RawQuery) (int32, error)
	VulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)
}

// NodeComponent returns a node component based on an input id (name:version)
func (resolver *Resolver) NodeComponent(ctx context.Context, args IDQuery) (NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeComponent")
	return resolver.nodeComponentV2(ctx, args)
}

func (resolver *Resolver) NodeComponents(ctx context.Context, q PaginatedQuery) ([]NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeComponents")
	query := search.AddRawQueriesAsConjunction(q.String(),
		search.NewQueryBuilder().AddRegexes(search.NodeID, ".+", ".*").Query())

	return resolver.nodeComponentsV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
}

func (resolver *Resolver) NodeComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeComponentCount")
	query := search.AddRawQueriesAsConjunction(args.String(),
		search.NewQueryBuilder().AddRegexes(search.NodeID, ".+", ".*").Query())

	return resolver.componentCountV2(ctx, RawQuery{Query: &query})
}

// nodeComponentResolverImpl resolves data about a node component.
type nodeComponentResolverImpl struct {
	parent *imageComponentResolver
}

func (resolver *Resolver) wrapIntoNodeComponentResolver(imgCompRes *imageComponentResolver) *nodeComponentResolverImpl {
	return &nodeComponentResolverImpl{parent: imgCompRes}
}

func (ncr *nodeComponentResolverImpl) ID(ctx context.Context) graphql.ID {
	return ncr.parent.ID(ctx)
}

func (ncr *nodeComponentResolverImpl) Name(ctx context.Context) string {
	return ncr.parent.Name(ctx)
}

func (ncr *nodeComponentResolverImpl) Version(ctx context.Context) string {
	return ncr.parent.Version(ctx)
}

func (ncr *nodeComponentResolverImpl) Priority(ctx context.Context) int32 {
	return ncr.parent.Priority(ctx)
}

func (ncr *nodeComponentResolverImpl) Source(ctx context.Context) string {
	return ncr.parent.Source(ctx)
}

// Does Location always return empty string for node components?
func (ncr *nodeComponentResolverImpl) Location(ctx context.Context, args RawQuery) (string, error) {
	return ncr.parent.Location(ctx, args)
}

// TODO : LastScanned should be resolved by replicating imageComponentResolver's LastScanned
func (ncr *nodeComponentResolverImpl) LastScanned(ctx context.Context) (*graphql.Time, error) {
	return ncr.parent.LastScanned(ctx)
}

func (ncr *nodeComponentResolverImpl) License(ctx context.Context) (*licenseResolver, error) {
	return ncr.parent.License(ctx)
}

func (ncr *nodeComponentResolverImpl) RiskScore(ctx context.Context) float64 {
	return ncr.parent.RiskScore(ctx)
}

func (ncr *nodeComponentResolverImpl) FixedIn(ctx context.Context) string {
	return ncr.parent.FixedIn(ctx)
}

func (ncr *nodeComponentResolverImpl) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	return ncr.parent.Nodes(ctx, args)
}

func (ncr *nodeComponentResolverImpl) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	return ncr.parent.NodeCount(ctx, args)
}

func (ncr *nodeComponentResolverImpl) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

func (ncr *nodeComponentResolverImpl) PlottedVulnerabilities(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	// TODO : implement this method
	return nil, errors.New("Resolver not implemented")
}

func (ncr *nodeComponentResolverImpl) TopVulnerability(ctx context.Context) (NodeVulnerabilityResolver, error) {
	// TODO : implement this method
	return nil, errors.New("Resolver not implemented")
}

func (ncr *nodeComponentResolverImpl) Vulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	// TODO : implement this method
	return nil, errors.New("Resolver not implemented")
}

func (ncr *nodeComponentResolverImpl) VulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	// TODO : implement this method
	return 0, errors.New("Resolver not implemented")
}

func (ncr *nodeComponentResolverImpl) VulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	// TODO : implement this method
	return nil, errors.New("Resolver not implemented")
}
