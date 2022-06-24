package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("NodeVulnerability",
			append(commonVulnerabilitySubResolvers,
				"nodeComponentCount(query: String): Int!",
				"nodeComponents(query: String, pagination: Pagination): [NodeComponent!]!",
				"nodeCount(query: String): Int!",
				"nodes(query: String, pagination: Pagination): [Node!]!",
			)),
		schema.AddQuery("nodeVulnerability(id: ID): NodeVulnerability"),
		schema.AddQuery("nodeVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability!]!"),
		schema.AddQuery("nodeVulnerabilityCount(query: String): Int!"),
	)
}

// NodeVulnerabilityResolver represents the supported API on node vulnerabilities
//  NOTE: This list is and should remain alphabetically ordered
type NodeVulnerabilityResolver interface {
	CommonVulnerabilityResolver

	NodeComponentCount(ctx context.Context, args RawQuery) (int32, error)
	NodeComponents(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error)
	NodeCount(ctx context.Context, args RawQuery) (int32, error)
	Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error)
}

// NodeVulnerability resolves a single vulnerability based on an id
func (resolver *Resolver) NodeVulnerability(ctx context.Context, args IDQuery) (NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.nodeVulnerabilityV2(ctx, args)
	}
	return resolver.nodeCveV2(ctx, args)
}

// NodeVulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) NodeVulnerabilities(ctx context.Context, q PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeCveTypeFiltering(q.String())
		return resolver.nodeVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	return resolver.nodeCvesV2(ctx, q)
}

// NodeVulnerabilityCount returns count of all clusters across infrastructure
func (resolver *Resolver) NodeVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeCveTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	return resolver.nodeCveCountV2(ctx, args)
}

// NodeVulnCounter returns a VulnerabilityCounterResolver for the input query.s
func (resolver *Resolver) NodeVulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeCveTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}
	return resolver.nodeCveCounterV2(ctx, args)
}

func (resolver *Resolver) nodeCveV2(ctx context.Context, args IDQuery) (NodeVulnerabilityResolver, error) {
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}
	vuln, exists, err := resolver.NodeCVEDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("node cve not found: %s", string(*args.ID))
	}
	vulnResolver, err := resolver.wrapNodeCVE(vuln, true, nil)
	if err != nil {
		return nil, err
	}
	vulnResolver.ctx = ctx
	return vulnResolver, nil
}

func (resolver *Resolver) nodeCvesV2(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	vulnResolvers, err := resolver.nodeCvesV2Query(ctx, query)
	if err != nil {
		return nil, err
	}

	ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, nil
}

func (resolver *Resolver) nodeCvesV2Query(ctx context.Context, query *v1.Query) ([]*nodeCVEResolver, error) {
	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	query = tryUnsuppressedQuery(query)

	vulns, err := vulnLoader.FromQuery(ctx, query)
	vulnResolvers, err := resolver.wrapNodeCVEs(vulns, err)
	return vulnResolvers, err
}

func (resolver *Resolver) nodeCveCountV2(ctx context.Context, args RawQuery) (int32, error) {
	if err := readCVEs(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return 0, err
	}

	query = tryUnsuppressedQuery(query)
	return vulnLoader.CountFromQuery(ctx, query)
}

func (resolver *Resolver) nodeCveCounterV2(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.nodeCveCounterV2Query(ctx, query)
}

func (resolver *Resolver) nodeCveCounterV2Query(ctx context.Context, query *v1.Query) (*VulnerabilityCounterResolver, error) {
	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	fixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := vulnLoader.FromQuery(ctx, fixableVulnsQuery)
	if err != nil {
		return nil, err
	}

	unFixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableCVEs, err := vulnLoader.FromQuery(ctx, unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}

	return mapNodeCVEsToVulnerabilityCounter(fixableVulns, unFixableCVEs), nil
}

// CreatedAt is the time a node CVE first seen in the system
func (resolver *nodeCVEResolver) CreatedAt(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetCveBaseInfo().GetCreatedAt()
	return timestamp(value)
}

// CVE name of the node CVE
func (resolver *nodeCVEResolver) CVE(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetCve()
}

// EnvImpact is the fraction of nodes that contain the nodeCVE
func (resolver *nodeCVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	n, d, err := resolver.getEnvImpactComponentsForNodeCVE(ctx)
	if err != nil {
		return 0, err
	}
	if d == 0 {
		return 0, nil
	}
	return float64(n) / float64(d), nil
}

func (resolver *nodeCVEResolver) getEnvImpactComponentsForNodeCVE(ctx context.Context) (numerator, denominator int, err error) {
	allNodesCount, err := resolver.root.NodeGlobalDataStore.CountAllNodes(ctx)
	if err != nil {
		return 0, 0, err
	}
	if allNodesCount == 0 {
		return 0, 0, nil
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return 0, 0, err
	}
	withThisCVECount, err := nodeLoader.CountFromQuery(resolver.withNodeVulnerabilityScope(ctx), search.EmptyQuery())
	if err != nil {
		return 0, 0, err
	}
	return int(withThisCVECount), allNodesCount, nil
}

func (resolver *nodeCVEResolver) withNodeVulnerabilityScope(ctx context.Context) context.Context {
	return scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
}

// FixedByVersion returns the version of the parent component that removes this CVE
func (resolver *nodeCVEResolver) FixedByVersion(ctx context.Context) (string, error) {
	return resolver.getNodeComponentFixedByVersion(ctx)
}

func (resolver *nodeCVEResolver) getNodeComponentFixedByVersion(_ context.Context) (string, error) {
	scope, hasScope := scoped.GetScope(resolver.ctx)
	if !hasScope {
		return "", nil
	}
	if scope.Level != v1.SearchCategory_NODE_COMPONENTS {
		return "", nil
	}
	edgeID := edges.EdgeID{ParentID: scope.ID, ChildID: resolver.data.GetId()}.ToString()
	edge, found, err := resolver.root.NodeComponentCVEEdgeDataStore.Get(resolver.ctx, edgeID)
	if err != nil || !found {
		return "", err
	}
	return edge.GetFixedBy(), nil
}

// ID of the node CVE
func (resolver *nodeCVEResolver) ID(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

// IsFixable returns whether node CVE is fixable by any component
func (resolver *nodeCVEResolver) IsFixable(ctx context.Context, args RawQuery) (bool, error) {
	// TODO : Why do we remove this field query and then add it back again in addScopeContextOrNodeCVEQuery ?
	q, err := args.AsV1QueryOrEmpty(search.ExcludeFieldLabel(search.CVE))
	if err != nil {
		return false, err
	}

	ctx, query := resolver.addScopeContextOrNodeCVEQuery(q)
	query = search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	count, err := resolver.root.NodeComponentCVEEdgeDataStore.Count(ctx, query)
	if err != nil {
		return false, err
	}
	return count != 0, nil
}

func (resolver *nodeCVEResolver) addScopeContextOrNodeCVEQuery(query *v1.Query) (context.Context, *v1.Query) {
	ctx := resolver.ctx
	scope, ok := scoped.GetScope(ctx)
	if !ok {
		return resolver.withNodeVulnerabilityScope(ctx), query
	}
	// If the scope is not set to node vulnerabilities then
	// we need to add a query to scope the search to the current nodeCVE
	if scope.Level != v1.SearchCategory_NODE_VULNERABILITIES {
		return ctx, search.ConjunctionQuery(query, resolver.getNodeCVEQuery())
	}

	return ctx, query
}

func (resolver *nodeCVEResolver) addScopeContextOrNodeCVERawQuery(query string) (context.Context, string) {
	ctx := resolver.ctx
	scope, ok := scoped.GetScope(ctx)
	if !ok {
		return resolver.withNodeVulnerabilityScope(ctx), query
	}
	// If the scope is not set to node vulnerabilities then
	// we need to add a query to scope the search to the current nodeCVE
	if scope.Level != v1.SearchCategory_NODE_VULNERABILITIES {
		return ctx, search.AddRawQueriesAsConjunction(query, resolver.getNodeCVERawQuery())
	}

	return ctx, query
}

func (resolver *nodeCVEResolver) getNodeCVEQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()
}

func (resolver *nodeCVEResolver) getNodeCVERawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).Query()
}

// LastModified is the time this node CVE was last modified in the system
func (resolver *nodeCVEResolver) LastModified(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetCveBaseInfo().GetLastModified()
	return timestamp(value)
}

// LastScanned is the last time this node CVE was last scanned in a node
func (resolver *nodeCVEResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit:  1,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.NodeScanTime.String(),
				Reversed: true,
			},
		},
	}

	nodes, err := nodeLoader.FromQuery(resolver.withNodeVulnerabilityScope(ctx), q)
	if err != nil || len(nodes) == 0 {
		return nil, err
	} else if len(nodes) > 1 {
		return nil, errors.New("multiple nodes matched for last scanned node vulnerability query")
	}

	return timestamp(nodes[0].GetScan().GetScanTime())
}

// Link to the node CVE
func (resolver *nodeCVEResolver) Link(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetLink()
}

// PublishedOn is date and time when this node CVE was first published in the cve feeds
func (resolver *nodeCVEResolver) PublishedOn(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetCveBaseInfo().GetPublishedOn()
	return timestamp(value)
}

// ScoreVersion of the node CVE
func (resolver *nodeCVEResolver) ScoreVersion(ctx context.Context) string {
	value := resolver.data.GetCveBaseInfo().GetScoreVersion()
	return value.String()
}

// Summary of the node CVE
func (resolver *nodeCVEResolver) Summary(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetSummary()
}

// SuppressActivation returns the snooze start timestamp of the node CVE
func (resolver *nodeCVEResolver) SuppressActivation(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetSnoozeStart()
	return timestamp(value)
}

// SuppressExpiry returns the snooze expiration timestamp of the node CVE
func (resolver *nodeCVEResolver) SuppressExpiry(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetSnoozeExpiry()
	return timestamp(value)
}

// Suppressed returns true if the node CVE is snoozed
func (resolver *nodeCVEResolver) Suppressed(ctx context.Context) bool {
	return resolver.data.GetSnoozed()
}

// UnusedVarSink represents a query sink
func (resolver *nodeCVEResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

// Vectors returns the CVSSV3 or CVSSV2 associated with the node CVE
func (resolver *nodeCVEResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	if val := resolver.data.GetCveBaseInfo().GetCvssV3(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV3Resolver{resolver.ctx, resolver.root, val},
		}
	}
	if val := resolver.data.GetCveBaseInfo().GetCvssV2(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV2Resolver{resolver.ctx, resolver.root, val},
		}
	}
	return nil
}

// VulnerabilityState returns the effective state of the node CVE (observed, deferred or marked as false positive).
func (resolver *nodeCVEResolver) VulnerabilityState(ctx context.Context) string {
	//TODO Should this be removed from nodeVulnerabilities graphQL ?
	return ""
}

// NodeComponentCount is the number of node components that contain the node CVE.
func (resolver *nodeCVEResolver) NodeComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	//TODO implement me (ROX-11299)
	panic("implement me")
}

// NodeComponents are the node components that contain the node CVE.
func (resolver *nodeCVEResolver) NodeComponents(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	//TODO implement me (ROX-11299)
	panic("implement me")
}

// NodeCount is the number of nodes that contain the node CVE
func (resolver *nodeCVEResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	ctx, query := resolver.addScopeContextOrNodeCVERawQuery(args.String())
	return resolver.root.NodeCount(ctx, RawQuery{Query: &query})
}

// Nodes are the nodes that contain the node CVE
func (resolver *nodeCVEResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	ctx, query := resolver.addScopeContextOrNodeCVERawQuery(args.String())
	return resolver.root.Nodes(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// withNodeCveTypeFiltering adds a conjunction as a raw query to filter vulns by CVEType Node
func withNodeCveTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_NODE_CVE.String()).Query())
}
