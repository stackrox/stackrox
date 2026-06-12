package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("node(id:ID!): Node"),
		schema.AddQuery("nodes(query: String, pagination: Pagination): [Node!]!"),
		schema.AddQuery("nodeCount(query: String): Int!"),

		// NOTE: This list is and should remain alphabetically ordered
		schema.AddExtraResolvers("Node", []string{
			"cluster: Cluster!",
			"nodeCVECountBySeverity(query: String): ResourceCountByCVESeverity!",
			"nodeComponentCount(query: String): Int!",
			"nodeComponents(query: String, pagination: Pagination): [NodeComponent!]!",
			"nodeStatus(query: String): String!",
			"nodeVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability]!",
			"nodeVulnerabilityCount(query: String): Int!",
			"nodeVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"plottedNodeVulnerabilities(query: String): PlottedNodeVulnerabilities!",
			"scan: NodeScan",
			"topNodeVulnerability(query: String): NodeVulnerability",
			"unusedVarSink(query: String): Int",

			// Node scan-related fields
			"scanNotes: [NodeScan_Note!]!",
			"scanTime: Time",
		}),
	)
}

// Node returns a resolver for a matching node, or nil if no node is found in any cluster
func (resolver *Resolver) Node(ctx context.Context, args struct{ graphql.ID }) (*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Node")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}
	node, err := nodeLoader.FromID(ctx, string(args.ID))
	return resolver.wrapNodeWithContext(ctx, node, node != nil, err)
}

// Nodes returns resolvers for a matching nodes, or nil if no node is found in any cluster
func (resolver *Resolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Nodes")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := nodeLoader.FromQuery(ctx, q)
	return resolver.wrapNodesWithContext(ctx, nodes, err)
}

// NodeCount returns count of nodes across clusters
func (resolver *Resolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeCount")
	if err := readNodes(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return 0, err
	}
	return nodeLoader.CountFromQuery(ctx, query)
}

func (resolver *nodeResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "Cluster")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}
	return resolver.root.Cluster(resolver.ctx, struct{ graphql.ID }{graphql.ID(resolver.data.GetClusterId())})
}

func (resolver *nodeResolver) NodeStatus(_ context.Context, _ RawQuery) (string, error) {
	return "active", nil
}

// NodeComponents returns the components in the node.
func (resolver *nodeResolver) NodeComponents(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeComponents")
	return resolver.root.NodeComponents(resolver.nodeScopeContext(ctx), args)
}

// NodeComponentCount returns the number of components in the node
func (resolver *nodeResolver) NodeComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeComponentCount")
	return resolver.root.NodeComponentCount(resolver.nodeScopeContext(ctx), args)
}

// TopNodeVulnerability returns the first node vulnerability with the top CVSS score.
func (resolver *nodeResolver) TopNodeVulnerability(ctx context.Context, args RawQuery) (NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "TopNodeVulnerability")

	return resolver.root.TopNodeVulnerability(resolver.nodeScopeContext(ctx), args)
}

func (resolver *nodeResolver) getNodeRawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.NodeID, resolver.data.GetId()).Query()
}

// NodeCVECountBySeverity returns the count of node cves by severity in the node.
func (resolver *nodeResolver) NodeCVECountBySeverity(ctx context.Context, args RawQuery) (*resourceCountBySeverityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeCVECountBySeverity")
	return resolver.root.NodeCVECountBySeverity(resolver.nodeScopeContext(ctx), args)
}

// NodeVulnerabilities returns the vulnerabilities in the node.
func (resolver *nodeResolver) NodeVulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeVulnerabilities")
	return resolver.root.NodeVulnerabilities(resolver.nodeScopeContext(ctx), args)
}

// NodeVulnerabilityCount returns the number of vulnerabilities the node has.
func (resolver *nodeResolver) NodeVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeVulnerabilityCount")
	return resolver.root.NodeVulnerabilityCount(resolver.nodeScopeContext(ctx), args)
}

// NodeVulnerabilityCounter resolves the number of different types of vulnerabilities contained in a node.
func (resolver *nodeResolver) NodeVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeVulnerabilityCounter")
	return resolver.root.NodeVulnerabilityCounter(resolver.nodeScopeContext(ctx), args)
}

// PlottedNodeVulnerabilities returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
func (resolver *nodeResolver) PlottedNodeVulnerabilities(ctx context.Context, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "PlottedNodeVulnerabilities")

	// (ROX-10911) Cluster scoping the context is not able to resolve node vulns when combined with 'Fixable:true/false' query
	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getNodeRawQuery())
	return resolver.root.PlottedNodeVulnerabilities(ctx, RawQuery{Query: &query})
}

func (resolver *nodeResolver) Scan(ctx context.Context) (*nodeScanResolver, error) {
	// If scan is pulled, it is most likely for the user to fetch all components and vulns contained in node.
	// Therefore, load the node again with full scan.
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}

	node, err := nodeLoader.FullNodeWithID(ctx, resolver.data.GetId())
	if err != nil {
		return nil, err
	}
	scan := node.GetScan()

	res, err := resolver.root.wrapNodeScan(scan, true, nil)
	if err != nil || res == nil {
		return nil, err
	}
	res.ctx = resolver.nodeScopeContext(ctx)
	return res, nil
}

func (resolver *nodeResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

func (resolver *nodeResolver) nodeScopeContext(ctx context.Context) context.Context {
	if ctx == nil {
		err := utils.ShouldErr(errors.New("argument 'ctx' is nil"))
		if err != nil {
			log.Error(err)
		}
	}
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}
	return scoped.Context(resolver.ctx, scoped.Scope{
		Level: v1.SearchCategory_NODES,
		IDs:   []string{resolver.data.GetId()},
	})
}

//// Node scan-related fields pulled as direct sub-resolvers of node.

func (resolver *nodeResolver) ScanNotes(_ context.Context) []string {
	return stringSlice(resolver.data.GetScan().GetNotes())
}

func (resolver *nodeResolver) ScanTime(_ context.Context) (*graphql.Time, error) {
	return protocompat.ConvertTimestampToGraphqlTimeOrError(resolver.data.GetScan().GetScanTime())
}
