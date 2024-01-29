package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs"
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
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("NodeVulnerability",
			append(commonVulnerabilitySubResolvers,
				"nodeComponentCount(query: String): Int!",
				"nodeComponents(query: String, pagination: Pagination): [NodeComponent!]!",
				"nodeCount(query: String): Int!",
				"nodes(query: String, pagination: Pagination): [Node!]!",
				"operatingSystem: String!",
			)),
		schema.AddQuery("nodeVulnerability(id: ID): NodeVulnerability"),
		schema.AddQuery("nodeVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability!]!"),
		schema.AddQuery("nodeVulnerabilityCount(query: String): Int!"),
	)
}

// NodeVulnerabilityResolver represents the supported API on node vulnerabilities
//
//	NOTE: This list is and should remain alphabetically ordered
type NodeVulnerabilityResolver interface {
	CommonVulnerabilityResolver

	NodeComponentCount(ctx context.Context, args RawQuery) (int32, error)
	NodeComponents(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error)
	NodeCount(ctx context.Context, args RawQuery) (int32, error)
	Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error)
	OperatingSystem(ctx context.Context) string
}

// NodeVulnerability resolves a single vulnerability based on an id
func (resolver *Resolver) NodeVulnerability(ctx context.Context, args IDQuery) (NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerability")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	vuln, err := vulnLoader.FromID(ctx, string(*args.ID))
	return resolver.wrapNodeCVEWithContext(ctx, vuln, true, err)
}

// NodeVulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) NodeVulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilities")

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
	vulnResolvers, err := resolver.wrapNodeCVEsWithContext(ctx, vulns, err)

	if err != nil {
		return nil, err
	}

	ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		ret = append(ret, res)
	}
	return ret, nil
}

// NodeVulnerabilityCount returns count of node vulnerabilities based on a query
func (resolver *Resolver) NodeVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCount")

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

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	// check for Fixable fields in args
	logErrorOnQueryContainingField(query, search.Fixable, "NodeVulnerabilityCounter")

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

// TopNodeVulnerability returns the most severe node vulnerability found in the scoped context
func (resolver *Resolver) TopNodeVulnerability(ctx context.Context, args RawQuery) (NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "TopNodeVulnerability")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	scope, ok := scoped.GetScope(ctx)
	if !ok {
		return nil, errors.New("TopNodeVulnerability called without scope context")
	} else if scope.Level != v1.SearchCategory_NODE_COMPONENTS && scope.Level != v1.SearchCategory_NODES {
		return nil, errors.New("TopNodeVulnerability called with improper scope context")
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.CVSS.String(),
				Reversed: true,
			},
			{
				Field:    search.CVE.String(),
				Reversed: true,
			},
		},
		Limit:  1,
		Offset: 0,
	}

	vulnLoader, err := loaders.GetNodeCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	vulns, err := vulnLoader.FromQuery(ctx, query)
	if err != nil || len(vulns) == 0 {
		return nil, err
	} else if len(vulns) > 1 {
		return nil, errors.New("multiple vulnerabilities matched for top node vulnerability")
	}

	res, err := resolver.wrapNodeCVEWithContext(ctx, vulns[0], true, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

/*
Utility Functions
*/

func (resolver *nodeCVEResolver) getNodeCVEQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
}

func nodeCveToVulnerabilityWithSeverity(in []*storage.NodeCVE) []VulnerabilityWithSeverity {
	ret := make([]VulnerabilityWithSeverity, 0, len(in))
	for _, vuln := range in {
		ret = append(ret, vuln)
	}
	return ret
}

func (resolver *nodeCVEResolver) nodeVulnerabilityScopeContext(ctx context.Context) context.Context {
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
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
}

/*
Sub Resolver Functions
*/

// EnvImpact is the fraction of nodes that contain the nodeCVE
func (resolver *nodeCVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "EnvImpact")

	allCount, err := resolver.root.NodeCount(ctx, RawQuery{})
	if err != nil || allCount == 0 {
		return 0, err
	}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	scopedCount, err := resolver.root.NodeCount(ctx, RawQuery{})
	if err != nil {
		return 0, err
	}
	return float64(scopedCount) / float64(allCount), nil
}

// FixedByVersion returns the version of the parent component that removes this CVE
func (resolver *nodeCVEResolver) FixedByVersion(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "FixedByVersion")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full node is embedded when node scan resolver is called.
	if embeddedVuln := embeddedobjs.NodeVulnFromContext(resolver.ctx); embeddedVuln != nil {
		return embeddedVuln.GetFixedBy(), nil
	}

	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_NODE_COMPONENTS)
	if !hasScope {
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
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "IsFixable")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full node is embedded when node scan resolver is called.
	if embeddedVuln := embeddedobjs.NodeVulnFromContext(resolver.ctx); embeddedVuln != nil {
		return embeddedVuln.GetFixedBy() != "", nil
	}

	query, err := args.AsV1QueryOrEmpty(search.ExcludeFieldLabel(search.CVEID))
	if err != nil {
		return false, err
	}
	// check for Fixable fields in args
	logErrorOnQueryContainingField(query, search.Fixable, "IsFixable")

	conjuncts := []*v1.Query{query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()}

	// check scoping, add as conjunction if needed
	if scope, ok := scoped.GetScope(resolver.ctx); !ok || scope.Level != v1.SearchCategory_NODE_VULNERABILITIES {
		conjuncts = append(conjuncts, resolver.getNodeCVEQuery())
	}

	query = search.ConjunctionQuery(conjuncts...)
	vulnLoader, err := loaders.GetNodeCVELoader(resolver.ctx)
	if err != nil {
		return false, err
	}
	count, err := vulnLoader.CountFromQuery(resolver.ctx, query)
	if err != nil {
		return false, err
	}
	return count != 0, nil
}

// LastScanned is the last time this node CVE was last scanned in a node
func (resolver *nodeCVEResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "LastScanned")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full node is embedded when node scan resolver is called.
	if scanTime := embeddedobjs.NodeComponentLastScannedFromContext(resolver.ctx); scanTime != nil {
		return timestamp(scanTime)
	}

	nodeLoader, err := loaders.GetNodeLoader(resolver.ctx)
	if err != nil {
		return nil, err
	}

	q := resolver.getNodeCVEQuery()
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

	nodes, err := nodeLoader.FromQuery(resolver.ctx, q)
	if err != nil || len(nodes) == 0 {
		return nil, err
	} else if len(nodes) > 1 {
		return nil, errors.New("multiple nodes matched for last scanned node vulnerability query")
	}

	return timestamp(nodes[0].GetScan().GetScanTime())
}

// UnusedVarSink represents a query sink
func (resolver *nodeCVEResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

// Vectors returns the CVSSV3 or CVSSV2 associated with the node CVE
func (resolver *nodeCVEResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "Vectors")

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
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "NodeComponentCount")
	return resolver.root.NodeComponentCount(resolver.nodeVulnerabilityScopeContext(ctx), args)
}

// NodeComponents are the node components that contain the node CVE.
func (resolver *nodeCVEResolver) NodeComponents(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "NodeComponents")
	return resolver.root.NodeComponents(resolver.nodeVulnerabilityScopeContext(ctx), args)
}

// NodeCount is the number of nodes that contain the node CVE
func (resolver *nodeCVEResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "NodeCount")
	return resolver.root.NodeCount(resolver.nodeVulnerabilityScopeContext(ctx), args)
}

// Nodes are the nodes that contain the node CVE
func (resolver *nodeCVEResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "Nodes")
	return resolver.root.Nodes(resolver.nodeVulnerabilityScopeContext(ctx), args)
}

// ID of the node CVE
func (resolver *nodeCVEResolver) ID(_ context.Context) graphql.ID {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "ID")
	return graphql.ID(resolver.data.GetId())
}

// CreatedAt is the time a node CVE first seen in the system
func (resolver *nodeCVEResolver) CreatedAt(_ context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "CreatedAt")
	return timestamp(resolver.data.GetCveBaseInfo().GetCreatedAt())
}

// CVE name of the node CVE
func (resolver *nodeCVEResolver) CVE(_ context.Context) string {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "CVE")
	return resolver.data.GetCveBaseInfo().GetCve()
}

// LastModified is the time this node CVE was last modified in the system
func (resolver *nodeCVEResolver) LastModified(_ context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "LastModified")
	return timestamp(resolver.data.GetCveBaseInfo().GetLastModified())
}

// Link to the node CVE
func (resolver *nodeCVEResolver) Link(_ context.Context) string {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "Link")
	return resolver.data.GetCveBaseInfo().GetLink()
}

// PublishedOn is date and time when this node CVE was first published in the cve feeds
func (resolver *nodeCVEResolver) PublishedOn(_ context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "PublishedOn")
	return timestamp(resolver.data.GetCveBaseInfo().GetPublishedOn())
}

// ScoreVersion of the node CVE
func (resolver *nodeCVEResolver) ScoreVersion(_ context.Context) string {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "ScoreVersion")
	return resolver.data.GetCveBaseInfo().GetScoreVersion().String()
}

// Summary of the node CVE
func (resolver *nodeCVEResolver) Summary(_ context.Context) string {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "Summary")
	return resolver.data.GetCveBaseInfo().GetSummary()
}

// SuppressActivation returns the snooze start timestamp of the node CVE
func (resolver *nodeCVEResolver) SuppressActivation(_ context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "SuppressActivation")
	return timestamp(resolver.data.GetSnoozeStart())
}

// SuppressExpiry returns the snooze expiration timestamp of the node CVE
func (resolver *nodeCVEResolver) SuppressExpiry(_ context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "SuppressExpiry")
	return timestamp(resolver.data.GetSnoozeExpiry())
}

// Suppressed returns true if the node CVE is snoozed
func (resolver *nodeCVEResolver) Suppressed(_ context.Context) bool {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "Suppressed")
	return resolver.data.GetSnoozed()
}
