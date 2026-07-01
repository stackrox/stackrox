package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/views/platformcve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxClusters = 1000
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("PlatformCVECore",
			[]string{
				"clusterCountByType: ClusterCountByType!",
				"clusterCount: Int!",
				"clusters(pagination: Pagination): [Cluster!]!",
				"clusterVulnerability: ClusterVulnerability!",
				"cve: String!",
				"cveType: String!",
				"cvss: Float!",
				"firstDiscoveredTime: Time",
				"id: ID!",
				"isFixable(query: String): Boolean!",
			}),
		schema.AddQuery("platformCVECount(query: String): Int!"),
		schema.AddQuery("platformCVEs(query: String, pagination: Pagination): [PlatformCVECore!]!"),
		// `subfieldScopeQuery` applies the scope query to all the subfields of the PlatformCVE resolver.
		// This eliminates the need to pass queries to individual resolvers.
		schema.AddQuery("platformCVE(cveID: String, subfieldScopeQuery: String): PlatformCVECore"),
	)
}

type platformCVECoreResolver struct {
	ctx  context.Context
	root *Resolver
	data platformcve.CveCore

	subFieldQuery *v1.Query
}

func (resolver *Resolver) wrapPlatformCVECoreWithContext(ctx context.Context, value platformcve.CveCore, err error) (*platformCVECoreResolver, error) {
	if err != nil || value == nil {
		return nil, err
	}
	return &platformCVECoreResolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapPlatformCVECoresWithContext(ctx context.Context, values []platformcve.CveCore, err error) ([]*platformCVECoreResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	ret := make([]*platformCVECoreResolver, len(values))
	for i, v := range values {
		ret[i] = &platformCVECoreResolver{ctx: ctx, root: resolver, data: v}
	}
	return ret, nil
}

// PlatformCVECount returns the count of platform cves satisfying the specified query.
// Note: By default, snoozed CVEs will be excluded. To get the count of snoozed CVEs, client should explicitly pass
// the filter "CVE Snoozed: true"
func (resolver *Resolver) PlatformCVECount(ctx context.Context, q RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PlatformCVECount")

	if err := readClusters(ctx); err != nil {
		return 0, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	query = tryUnsuppressedQuery(query)

	count, err := resolver.PlatformCVEView.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// PlatformCVEs returns graphQL resolver for platform cves satisfying the specified query.
// Note: By default, snoozed CVEs will be excluded. To get snoozed CVEs, client should explicitly pass
// the filter "CVE Snoozed: true"
func (resolver *Resolver) PlatformCVEs(ctx context.Context, q PaginatedQuery) ([]*platformCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PlatformCVEs")

	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)

	cves, err := resolver.PlatformCVEView.Get(ctx, query)
	ret, err := resolver.wrapPlatformCVECoresWithContext(ctx, cves, err)
	if err != nil {
		return nil, err
	}
	for _, r := range ret {
		r.subFieldQuery = query
	}

	return ret, nil
}

// PlatformCVE returns graphQL resolver for specified platform cve id.
func (resolver *Resolver) PlatformCVE(ctx context.Context, args struct {
	CveID              *string
	SubfieldScopeQuery *string
}) (*platformCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PlatformCVE")

	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	if args.CveID == nil {
		return nil, errors.New("cveID variable must be set")
	}

	query := search.NewQueryBuilder().AddExactMatches(search.CVEID, *args.CveID).ProtoQuery()
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

	cves, err := resolver.PlatformCVEView.Get(ctx, query)
	if len(cves) == 0 {
		return nil, nil
	}
	if len(cves) > 1 {
		utils.Should(errors.Errorf("Retrieved multiple rows when only one row is expected for CVEID=%s query", *args.CveID))
		return nil, err
	}
	ret, err := resolver.wrapPlatformCVECoreWithContext(ctx, cves[0], err)
	if err != nil {
		return nil, err
	}
	ret.subFieldQuery = query

	return ret, nil
}

// CVE returns the platform cve name
func (resolver *platformCVECoreResolver) CVE(_ context.Context) string {
	return resolver.data.GetCVE()
}

// CVEType returns the type of the given platform cve
func (resolver *platformCVECoreResolver) CVEType(_ context.Context) string {
	return resolver.data.GetCVEType().String()
}

func (resolver *platformCVECoreResolver) CVSS(_ context.Context) float64 {
	return float64(resolver.data.GetCVSS())
}

// ClusterCountByType returns the number of clusters of each type affected by the given platform cve
func (resolver *platformCVECoreResolver) ClusterCountByType(ctx context.Context) (*clusterCountByTypeResolver, error) {
	return resolver.root.wrapClusterCountByTypeWithContext(ctx, resolver.data.GetClusterCountByPlatformType(), nil)
}

// ClusterCount returns the number of clusters affected by the given platform cve
func (resolver *platformCVECoreResolver) ClusterCount(_ context.Context) int32 {
	return int32(resolver.data.GetClusterCount())
}

// Clusters returns a paginated list of clusters containing given platform cve
func (resolver *platformCVECoreResolver) Clusters(ctx context.Context, args struct{ Pagination *inputtypes.Pagination }) ([]*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.PlatformCVECore, "Clusters")

	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	query := search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetCVEID()).ProtoQuery()
	if resolver.subFieldQuery != nil {
		query = search.ConjunctionQuery(query, resolver.subFieldQuery)
	}
	if args.Pagination != nil {
		paginated.FillPagination(query, args.Pagination.AsV1Pagination(), maxClusters)
	}

	clusters, err := resolver.root.ClusterDataStore.SearchRawClusters(ctx, query)
	return resolver.root.wrapClustersWithContext(ctx, clusters, err)
}

// ClusterVulnerability returns the associated cluster vulnerability with the given platform cve.
// The cluster vulnerability contains metadata for cve like link, summary, etc
func (resolver *platformCVECoreResolver) ClusterVulnerability(ctx context.Context) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.PlatformCVECore, "ClusterVulnerability")

	cveID := graphql.ID(resolver.data.GetCVEID())

	return resolver.root.ClusterVulnerability(ctx, IDQuery{
		ID: &cveID,
	})
}

// FirstDiscoveredTime returns the first time the given platform cve was discovered across all clusters
func (resolver *platformCVECoreResolver) FirstDiscoveredTime(_ context.Context) *graphql.Time {
	ts := resolver.data.GetFirstDiscoveredTime()
	if ts == nil {
		return nil
	}
	return &graphql.Time{
		Time: *ts,
	}
}

// ID of the given platform CVE
func (resolver *platformCVECoreResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(resolver.data.GetCVEID())
}

// IsFixable returns true if the given platform cve is fixable in any of the clusters matched by the given query
func (resolver *platformCVECoreResolver) IsFixable(ctx context.Context, args RawQuery) bool {
	return resolver.data.GetFixability()
}
