package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/common"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/views/nodecve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxNodes = 1000
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("NodeCVECore",
			[]string{
				"affectedNodeCount: Int!",
				"affectedNodeCountBySeverity: ResourceCountByCVESeverity!",
				"cve: String!",
				"distroTuples: [NodeVulnerability!]!",
				"firstDiscoveredInSystem: Time",
				"nodes(pagination: Pagination): [Node!]!",
				"operatingSystemCount: Int!",
				"topCVSS: Float!",
			}),
		schema.AddQuery("nodeCVECount(query: String): Int!"),
		schema.AddQuery("nodeCVEs(query: String, pagination: Pagination): [NodeCVECore!]!"),
		// `subfieldScopeQuery` applies the scope query to all the subfields of the NodeCVE resolver.
		// This eliminates the need to pass queries to individual resolvers.
		schema.AddQuery("nodeCVE(cve: String, subfieldScopeQuery: String): NodeCVECore"),
		schema.AddQuery("nodeCVECountBySeverity(query: String): ResourceCountByCVESeverity!"),
	)
}

type nodeCVECoreResolver struct {
	ctx  context.Context
	root *Resolver
	data nodecve.CveCore

	subFieldQuery *v1.Query
}

func (resolver *Resolver) wrapNodeCVECoreWithContext(ctx context.Context, value nodecve.CveCore, err error) (*nodeCVECoreResolver, error) {
	if err != nil || value == nil {
		return nil, err
	}
	return &nodeCVECoreResolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapNodeCVECoresWithContext(ctx context.Context, values []nodecve.CveCore, err error) ([]*nodeCVECoreResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*nodeCVECoreResolver, len(values))
	for i, v := range values {
		output[i] = &nodeCVECoreResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

func (resolver *nodeCVECoreResolver) AffectedNodeCount(_ context.Context) int32 {
	return int32(resolver.data.GetNodeCount())
}

func (resolver *nodeCVECoreResolver) AffectedNodeCountBySeverity(ctx context.Context) (*resourceCountBySeverityResolver, error) {
	return resolver.root.wrapResourceCountByCVESeverityWithContext(ctx, resolver.data.GetNodeCountBySeverity(), nil)
}

func (resolver *nodeCVECoreResolver) CVE(_ context.Context) string {
	return resolver.data.GetCVE()
}

func (resolver *nodeCVECoreResolver) DistroTuples(ctx context.Context) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "DistroTuples")
	// DistroTuples resolver filters out snoozed CVEs when no explicit filter by CVESuppressed is provided.
	// When DistroTuples resolver is called from here, it is to get the details of a single CVE which cannot be
	// obtained via SQF. So, the auto removal of snoozed CVEs is unintentional here. Hence, we add explicit filter with
	// CVESuppressed == true OR false
	q := PaginatedQuery{
		Query: pointers.String(search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetCVEIDs()...).
			AddBools(search.CVESuppressed, true, false).
			Query()),
	}
	return resolver.root.NodeVulnerabilities(ctx, q)
}

func (resolver *nodeCVECoreResolver) FirstDiscoveredInSystem(_ context.Context) *graphql.Time {
	ts := resolver.data.GetFirstDiscoveredInSystem()
	if ts == nil {
		return nil
	}
	return &graphql.Time{
		Time: *ts,
	}
}

func (resolver *nodeCVECoreResolver) Nodes(ctx context.Context, args struct{ Pagination *inputtypes.Pagination }) ([]*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "Nodes")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	nodeQ := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetCVE()).ProtoQuery()
	if resolver.subFieldQuery != nil {
		nodeQ = search.ConjunctionQuery(nodeQ, resolver.subFieldQuery)
	}
	if args.Pagination != nil {
		paginated.FillPagination(nodeQ, args.Pagination.AsV1Pagination(), maxNodes)
	}
	nodes, err := resolver.root.NodeDataStore.SearchRawNodes(ctx, nodeQ)
	return resolver.root.wrapNodes(nodes, err)
}

func (resolver *nodeCVECoreResolver) OperatingSystemCount(_ context.Context) int32 {
	return int32(resolver.data.GetOperatingSystemCount())
}

func (resolver *nodeCVECoreResolver) TopCVSS(_ context.Context) float64 {
	return float64(resolver.data.GetTopCVSS())
}

// NodeCVE returns graphQL resolver for specified node cve.
func (resolver *Resolver) NodeCVE(ctx context.Context, args struct {
	Cve                *string
	SubfieldScopeQuery *string
}) (*nodeCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.NodeCVEs, "NodeCVEMetadata")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	if args.Cve == nil {
		return nil, errors.New("cve variable must be set")
	}

	query := search.NewQueryBuilder().AddExactMatches(search.CVE, *args.Cve).ProtoQuery()
	if args.SubfieldScopeQuery != nil {
		rQuery := RawQuery{
			Query: args.SubfieldScopeQuery,
		}
		filterQuery, err := rQuery.AsV1QueryOrEmpty()
		if err != nil {
			return nil, err
		}
		query = search.ConjunctionQuery(query, filterQuery)
	}
	query = common.WithoutOrphanedNodeCVEsQuery(query)

	cves, err := resolver.NodeCVEView.Get(ctx, query)
	if len(cves) == 0 {
		return nil, nil
	}
	if len(cves) > 1 {
		utils.Should(errors.Errorf("Retrieved multiple rows when only one row is expected for CVE=%s query", *args.Cve))
		return nil, err
	}
	ret, err := resolver.wrapNodeCVECoreWithContext(ctx, cves[0], err)
	if err != nil {
		return nil, err
	}
	ret.subFieldQuery = query

	return ret, nil
}

// NodeCVECount returns the count of node cves satisfying the specified query.
// Note: Client must explicitly pass observed/deferred CVEs.
func (resolver *Resolver) NodeCVECount(ctx context.Context, q RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeCVECount")

	if err := readNodes(ctx); err != nil {
		return 0, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	query = tryUnsuppressedQuery(query)
	query = common.WithoutOrphanedNodeCVEsQuery(query)

	count, err := resolver.NodeCVEView.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// NodeCVEs returns graphQL resolver for node cves satisfying the specified query.
// Note: Client must explicitly pass observed/deferred CVEs.
func (resolver *Resolver) NodeCVEs(ctx context.Context, q PaginatedQuery) ([]*nodeCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeCVEs")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	query = common.WithoutOrphanedNodeCVEsQuery(query)

	cves, err := resolver.NodeCVEView.Get(ctx, query)
	ret, err := resolver.wrapNodeCVECoresWithContext(ctx, cves, err)
	if err != nil {
		return nil, err
	}
	for _, r := range ret {
		r.subFieldQuery = query
	}

	return ret, nil
}

// NodeCVECountBySeverity returns the count of node cves satisfying the specified query by severity.
// Note: Client must explicitly pass observed/deferred CVEs.
func (resolver *Resolver) NodeCVECountBySeverity(ctx context.Context, q RawQuery) (*resourceCountBySeverityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeCVECountBySeverity")

	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	query = common.WithoutOrphanedNodeCVEsQuery(query)

	response, err := resolver.NodeCVEView.CountBySeverity(ctx, query)
	if err != nil {
		return nil, err
	}
	return resolver.wrapResourceCountByCVESeverityWithContext(ctx, response, nil)
}
