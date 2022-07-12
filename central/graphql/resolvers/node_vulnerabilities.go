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

	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	vuln, err := vulnLoader.FromID(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	}
	vulnResolver, err := resolver.wrapNodeCVE(vuln, true, nil)
	if err != nil {
		return nil, err
	}
	vulnResolver.ctx = ctx
	return vulnResolver, nil
}

// NodeVulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) NodeVulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeCveTypeFiltering(args.String())
		return resolver.nodeVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
	}

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	query = tryUnsuppressedQuery(query)

	vulns, err := vulnLoader.FromQuery(ctx, query)
	vulnResolvers, err := resolver.wrapNodeCVEs(vulns, err)

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

// NodeVulnerabilityCount returns count of all clusters across infrastructure
func (resolver *Resolver) NodeVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeCveTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}

	if err := readNodes(ctx); err != nil {
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

// NodeVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query.s
func (resolver *Resolver) NodeVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeCveTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	// check for Fixable fields in args
	ErrorOnQueryContainingField(query, search.Fixable, "Unexpected `Fixable` field in NodeVulnerabilityCounter resolver")

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	fixableQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := vulnLoader.FromQuery(ctx, fixableQuery)
	if err != nil {
		return nil, err
	}
	fixable := nodeCveToVulnerabilityWithSeverity(fixableVulns)

	unFixableQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableVulns, err := vulnLoader.FromQuery(ctx, unFixableQuery)
	if err != nil {
		return nil, err
	}
	unfixable := nodeCveToVulnerabilityWithSeverity(unFixableVulns)

	return mapCVEsToVulnerabilityCounter(fixable, unfixable), nil
}

/*
Utility Functions
*/

func (resolver *nodeCVEResolver) getNodeCVEQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *nodeCVEResolver) getNodeCVERawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).Query()
}

func nodeCveToVulnerabilityWithSeverity(in []*storage.NodeCVE) []VulnerabilityWithSeverity {
	ret := make([]VulnerabilityWithSeverity, 0, len(in))
	for _, vuln := range in {
		ret = append(ret, vuln)
	}
	return ret
}

// withNodeCveTypeFiltering adds a conjunction as a raw query to filter vulns by CVEType Node
func withNodeCveTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_NODE_CVE.String()).Query())
}

func (resolver *nodeCVEResolver) withNodeVulnerabilityScope(ctx context.Context) context.Context {
	return scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
}

/*
Sub Resolver Functions
*/

// EnvImpact is the fraction of nodes that contain the nodeCVE
func (resolver *nodeCVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return 0, err
	}
	numNodes, err := nodeLoader.CountAll(ctx)
	if err != nil || numNodes == 0 {
		return 0, err
	}

	query := resolver.getNodeCVEQuery()
	numNodesWithCVE, err := nodeLoader.CountFromQuery(ctx, query)
	if err != nil || numNodesWithCVE == 0 {
		return 0, err
	}
	return float64(numNodesWithCVE) / float64(numNodes), nil
}

// FixedByVersion returns the version of the parent component that removes this CVE
func (resolver *nodeCVEResolver) FixedByVersion(_ context.Context) (string, error) {
	scope, hasScope := scoped.GetScope(resolver.ctx)
	if !hasScope {
		return "", nil
	}
	if scope.Level != v1.SearchCategory_NODE_COMPONENTS {
		return "", nil
	}
	query := search.NewQueryBuilder().AddExactMatches(search.ComponentID, scope.ID).AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
	edges, err := resolver.root.NodeComponentCVEEdgeDataStore.SearchRawEdges(resolver.ctx, query)
	if err != nil || len(edges) == 0 {
		return "", err
	}
	return edges[0].GetFixedBy(), nil
}

// IsFixable returns whether node CVE is fixable by any component
func (resolver *nodeCVEResolver) IsFixable(ctx context.Context, args RawQuery) (bool, error) {
	query, err := args.AsV1QueryOrEmpty(search.ExcludeFieldLabel(search.CVEID))
	if err != nil {
		return false, err
	}

	// check for Fixable fields in args
	ErrorOnQueryContainingField(query, search.Fixable, "Unexpected `Fixable` field in IsFixable sub resolver")

	conjuncts := []*v1.Query{query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()}

	// check scoping, add as conjunction if needed
	if scope, ok := scoped.GetScope(resolver.ctx); !ok || scope.Level != v1.SearchCategory_NODE_VULNERABILITIES {
		conjuncts = append(conjuncts, resolver.getNodeCVEQuery())
	}

	query = search.ConjunctionQuery(conjuncts...)
	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return false, err
	}
	count, err := vulnLoader.CountFromQuery(ctx, query)
	if err != nil {
		return false, err
	}
	return count != 0, nil
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

	nodes, err := nodeLoader.FromQuery(ctx, search.ConjunctionQuery(q, resolver.getNodeCVEQuery()))
	if err != nil || len(nodes) == 0 {
		return nil, err
	} else if len(nodes) > 1 {
		return nil, errors.New("multiple nodes matched for last scanned node vulnerability query")
	}

	return timestamp(nodes[0].GetScan().GetScanTime())
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

// NodeComponentCount is the number of node components that contain the node CVE.
func (resolver *nodeCVEResolver) NodeComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	return resolver.root.NodeComponentCount(resolver.withNodeVulnerabilityScope(ctx), args)
}

// NodeComponents are the node components that contain the node CVE.
func (resolver *nodeCVEResolver) NodeComponents(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	return resolver.root.NodeComponents(resolver.withNodeVulnerabilityScope(ctx), args)
}

// NodeCount is the number of nodes that contain the node CVE
func (resolver *nodeCVEResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	return resolver.root.NodeCount(resolver.withNodeVulnerabilityScope(ctx), args)
}

// Nodes are the nodes that contain the node CVE
func (resolver *nodeCVEResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	return resolver.root.Nodes(resolver.withNodeVulnerabilityScope(ctx), args)
}

// Follows are functions that return information that is nested in the CVEInfo object
// or are convenience functions to allow time for UI to migrate to new naming schemes

// ID of the node CVE
func (resolver *nodeCVEResolver) ID(ctx context.Context) graphql.ID {
	return graphql.ID(resolver.data.GetId())
}

// CreatedAt is the time a node CVE first seen in the system
func (resolver *nodeCVEResolver) CreatedAt(ctx context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetCreatedAt())
}

// CVE name of the node CVE
func (resolver *nodeCVEResolver) CVE(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetCve()
}

// LastModified is the time this node CVE was last modified in the system
func (resolver *nodeCVEResolver) LastModified(ctx context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetLastModified())
}

// Link to the node CVE
func (resolver *nodeCVEResolver) Link(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetLink()
}

// PublishedOn is date and time when this node CVE was first published in the cve feeds
func (resolver *nodeCVEResolver) PublishedOn(ctx context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetPublishedOn())
}

// ScoreVersion of the node CVE
func (resolver *nodeCVEResolver) ScoreVersion(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetScoreVersion().String()
}

// Summary of the node CVE
func (resolver *nodeCVEResolver) Summary(ctx context.Context) string {
	return resolver.data.GetCveBaseInfo().GetSummary()
}

// SuppressActivation returns the snooze start timestamp of the node CVE
func (resolver *nodeCVEResolver) SuppressActivation(ctx context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetSnoozeStart())
}

// SuppressExpiry returns the snooze expiration timestamp of the node CVE
func (resolver *nodeCVEResolver) SuppressExpiry(ctx context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetSnoozeExpiry())
}

// Suppressed returns true if the node CVE is snoozed
func (resolver *nodeCVEResolver) Suppressed(ctx context.Context) bool {
	return resolver.data.GetSnoozed()
}
