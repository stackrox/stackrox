package resolvers

import (
	"context"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
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
		schema.AddType("ComplianceControlCount", []string{"failingCount: Int!", "passingCount: Int!", "unknownCount: Int!"}),

		// NOTE: This list is and should remain alphabetically ordered
		schema.AddExtraResolvers("Node", []string{
			"cluster: Cluster!",
			"complianceResults(query: String): [ControlResult!]!",
			"controls(query: String): [ComplianceControl!]!",
			"controlStatus(query: String): String!",
			"failingControls(query: String): [ComplianceControl!]!",
			"nodeComplianceControlCount(query: String) : ComplianceControlCount!",
			"nodeComponentCount(query: String): Int!",
			"nodeComponents(query: String, pagination: Pagination): [NodeComponent!]!",
			"nodeStatus(query: String): String!",
			"nodeVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability]!",
			"nodeVulnerabilityCount(query: String): Int!",
			"nodeVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"passingControls(query: String): [ComplianceControl!]!",
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

func (resolver *nodeResolver) ComplianceResults(ctx context.Context, args RawQuery) ([]*controlResultResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "ComplianceResults")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nodeID := resolver.data.GetId()
	output.addNodeData(resolver.root, runResults, func(node *storage.ComplianceDomain_Node, _ *v1.ComplianceControl) bool {
		return node.GetId() == nodeID
	})
	return *output, nil
}

func (resolver *nodeResolver) NodeComplianceControlCount(ctx context.Context, args RawQuery) (*complianceControlCountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeComplianceControlCount")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	if results == nil {
		return &complianceControlCountResolver{}, nil
	}
	return getComplianceControlCountFromAggregationResults(results), nil
}

func (resolver *nodeResolver) ControlStatus(ctx context.Context, args RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "ControlStatus")
	if err := readCompliance(ctx); err != nil {
		return "Fail", err
	}
	r, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE}, args)
	if err != nil || r == nil {
		return "Fail", err
	}
	if len(r) != 1 {
		return "Fail", errors.Errorf("unexpected node aggregation results length: expected: 1, actual: %d", len(r))
	}
	return getControlStatusFromAggregationResult(r[0]), nil
}

func (resolver *nodeResolver) getNodeLastSuccessfulComplianceRunAggregatedResult(ctx context.Context, scope []storage.ComplianceAggregation_Scope, args RawQuery) ([]*storage.ComplianceAggregation_Result, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	standardIDs, err := getStandardIDs(ctx, resolver.root.ComplianceStandardStore)
	if err != nil {
		return nil, err
	}
	hasComplianceSuccessfullyRun, err := resolver.root.ComplianceDataStore.IsComplianceRunSuccessfulOnCluster(ctx, resolver.data.GetClusterId(), standardIDs)
	if err != nil || !hasComplianceSuccessfullyRun {
		return nil, err
	}
	query, err := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.NodeID, resolver.data.GetId()).RawQuery()
	if err != nil {
		return nil, err
	}
	if args.Query != nil {
		query = strings.Join([]string{query, args.String()}, "+")
	}
	r, _, _, err := resolver.root.ComplianceAggregator.Aggregate(ctx, query, scope, storage.ComplianceAggregation_CONTROL)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (resolver *nodeResolver) FailingControls(ctx context.Context, args RawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "FailingControls")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, failing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *nodeResolver) PassingControls(ctx context.Context, args RawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "PassingControls")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, passing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *nodeResolver) Controls(ctx context.Context, args RawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "Controls")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, all, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func getComplianceControlsFromAggregationResults(results []*storage.ComplianceAggregation_Result, controlType resultType, cs complianceStandards.Repository) ([]*v1.ComplianceControl, error) {
	if cs == nil {
		return nil, errors.New("empty compliance standards store encountered: argument cs is nil")
	}
	var controls []*v1.ComplianceControl
	for _, r := range results {
		if (controlType == passing && r.GetNumPassing() == 0) || (controlType == failing && r.GetNumFailing() == 0) {
			continue
		}
		controlID, err := getScopeIDFromAggregationResult(r, storage.ComplianceAggregation_CONTROL)
		if err != nil {
			continue
		}
		control := cs.Control(controlID)
		if control == nil {
			continue
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func getComplianceControlCountFromAggregationResults(results []*storage.ComplianceAggregation_Result) *complianceControlCountResolver {
	ret := &complianceControlCountResolver{}
	for _, r := range results {
		if r.GetNumFailing() != 0 {
			ret.failingCount++
		} else if r.GetNumPassing() != 0 {
			ret.passingCount++
		} else {
			ret.unknownCount++
		}
	}
	return ret
}

type complianceControlCountResolver struct {
	failingCount int32
	passingCount int32
	unknownCount int32
}

func (resolver *complianceControlCountResolver) FailingCount() int32 {
	return resolver.failingCount
}

func (resolver *complianceControlCountResolver) PassingCount() int32 {
	return resolver.passingCount
}

func (resolver *complianceControlCountResolver) UnknownCount() int32 {
	return resolver.unknownCount
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
		ID:    resolver.data.GetId(),
	})
}

//// Node scan-related fields pulled as direct sub-resolvers of node.

func (resolver *nodeResolver) ScanNotes(_ context.Context) []string {
	return stringSlice(resolver.data.GetScan().GetNotes())
}

func (resolver *nodeResolver) ScanTime(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetScan().GetScanTime())
}
