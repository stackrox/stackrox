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
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ClusterVulnerability",
			append(commonVulnerabilitySubResolvers,
				"clusterCount(query: String): Int!",
				"clusters(query: String, pagination: Pagination): [Cluster!]!",
				"vulnerabilityType: String!",
				"vulnerabilityTypes: [String!]!",
			)),
		schema.AddQuery("clusterVulnerability(id: ID): ClusterVulnerability"),
		schema.AddQuery("clusterVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ClusterVulnerability!]!"),
		schema.AddQuery("clusterVulnerabilityCount(query: String): Int!"),
		schema.AddQuery("k8sClusterVulnerabilities(query: String, pagination: Pagination): [ClusterVulnerability!]!"),
		schema.AddQuery("k8sClusterVulnerability(id: ID): ClusterVulnerability"),
		schema.AddQuery("k8sClusterVulnerabilityCount(query: String): Int!"),
		schema.AddQuery("istioClusterVulnerabilities(query: String, pagination: Pagination): [ClusterVulnerability!]!"),
		schema.AddQuery("istioClusterVulnerability(id: ID): ClusterVulnerability"),
		schema.AddQuery("istioClusterVulnerabilityCount(query: String): Int!"),
		schema.AddQuery("openShiftClusterVulnerabilities(query: String, pagination: Pagination): [ClusterVulnerability!]!"),
		schema.AddQuery("openShiftClusterVulnerability(id: ID): ClusterVulnerability"),
		schema.AddQuery("openShiftClusterVulnerabilityCount(query: String): Int!"),
	)
}

// ClusterVulnerabilityResolver represents the supported API on image vulnerabilities
//
//	NOTE: This list is and should remain alphabetically ordered
type ClusterVulnerabilityResolver interface {
	CommonVulnerabilityResolver

	ClusterCount(ctx context.Context, args RawQuery) (int32, error)
	Clusters(ctx context.Context, args PaginatedQuery) ([]*clusterResolver, error)
	VulnerabilityType() string
	VulnerabilityTypes() []string
}

// ClusterVulnerability returns a vulnerability of the given id
func (resolver *Resolver) ClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerability")

	// check permissions
	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	// get loader
	loader, err := loaders.GetClusterCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	ret, err := loader.FromID(ctx, string(*args.ID))
	return resolver.wrapClusterCVEWithContext(ctx, ret, true, err)
}

// ClusterVulnerabilities resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ClusterVulnerabilities(ctx context.Context, q PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilities")

	// check permissions
	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	// cast query
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	// get loader
	loader, err := loaders.GetClusterCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	// get values
	query = tryUnsuppressedQuery(query)
	vulns, err := loader.FromQuery(ctx, query)
	cveResolvers, err := resolver.wrapClusterCVEsWithContext(ctx, vulns, err)
	if err != nil {
		return nil, err
	}

	// cast as return type
	ret := make([]ClusterVulnerabilityResolver, 0, len(cveResolvers))
	for _, res := range cveResolvers {
		ret = append(ret, res)
	}
	return ret, nil
}

// ClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilityCount")

	// check permissions
	if err := readClusters(ctx); err != nil {
		return 0, err
	}

	// cast query
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	// get loader
	loader, err := loaders.GetClusterCVELoader(ctx)
	if err != nil {
		return 0, err
	}
	query = tryUnsuppressedQuery(query)

	return loader.CountFromQuery(ctx, query)
}

// ClusterVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ClusterVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilityCounter")

	// check permissions
	if err := readClusters(ctx); err != nil {
		return nil, err
	}

	// cast query
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	// check for Fixable fields in args
	logErrorOnQueryContainingField(query, search.Fixable, "ClusterVulnerabilityCounter")

	// get loader
	loader, err := loaders.GetClusterCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)

	// get fixable vulns
	fixableQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := loader.FromQuery(ctx, fixableQuery)
	if err != nil {
		return nil, err
	}
	fixable := clusterCveToVulnerabilityWithSeverity(fixableVulns)

	// get unfixable vulns
	unFixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableVulns, err := loader.FromQuery(ctx, unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}
	unfixable := clusterCveToVulnerabilityWithSeverity(unFixableVulns)

	return mapCVEsToVulnerabilityCounter(fixable, unfixable), nil
}

// K8sClusterVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) K8sClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sClusterVulnerability")
	return resolver.ClusterVulnerability(ctx, args)
}

// K8sClusterVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) K8sClusterVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sClusterVulnerabilities")
	query := withK8sTypeFiltering(args.String())
	return resolver.ClusterVulnerabilities(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// K8sClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) K8sClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sClusterVulnerabilityCount")
	query := withK8sTypeFiltering(args.String())
	return resolver.ClusterVulnerabilityCount(ctx, RawQuery{Query: &query})
}

// IstioClusterVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) IstioClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioClusterVulnerability")
	return resolver.ClusterVulnerability(ctx, args)
}

// IstioClusterVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) IstioClusterVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioClusterVulnerabilities")
	query := withIstioTypeFiltering(args.String())
	return resolver.ClusterVulnerabilities(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// IstioClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) IstioClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioClusterVulnerabilityCount")
	query := withIstioTypeFiltering(args.String())
	return resolver.ClusterVulnerabilityCount(ctx, RawQuery{Query: &query})
}

// OpenShiftClusterVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) OpenShiftClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "OpenShiftClusterVulnerability")
	return resolver.ClusterVulnerability(ctx, args)
}

// OpenShiftClusterVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) OpenShiftClusterVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "OpenShiftClusterVulnerabilities")
	query := withOpenShiftTypeFiltering(args.String())
	return resolver.ClusterVulnerabilities(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// OpenShiftClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) OpenShiftClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "OpenShiftClusterVulnerabilityCount")
	query := withOpenShiftTypeFiltering(args.String())
	return resolver.ClusterVulnerabilityCount(ctx, RawQuery{Query: &query})
}

/*
Utility Functions
*/
// withK8sTypeFiltering adds a conjunction as a raw query to filter vulnerability k8s type
func withK8sTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).Query())
}

// withIstioTypeFiltering adds a conjunction as a raw query to filter vulnerability istio type
func withIstioTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_ISTIO_CVE.String()).Query())
}

// withOpenShiftTypeFiltering adds a conjunction as a raw query to filter vulnerability open shift type
func withOpenShiftTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_OPENSHIFT_CVE.String()).Query())
}

func (resolver *clusterCVEResolver) clusterVulnerabilityScopeContext(ctx context.Context) context.Context {
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
		Level: v1.SearchCategory_CLUSTER_VULNERABILITIES,
	})
}

func clusterCveToVulnerabilityWithSeverity(in []*storage.ClusterCVE) []VulnerabilityWithSeverity {
	ret := make([]VulnerabilityWithSeverity, len(in))
	for i, vuln := range in {
		ret[i] = vuln
	}
	return ret
}

func (resolver *clusterCVEResolver) getClusterCVEQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
}

/*
Sub Resolver Functions
*/

// Clusters returns resolvers for clusters affected by cluster vulnerability.
func (resolver *clusterCVEResolver) Clusters(ctx context.Context, args PaginatedQuery) ([]*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "Clusters")
	return resolver.root.Clusters(resolver.clusterVulnerabilityScopeContext(ctx), args)
}

// ClusterCount returns a number of clusters affected by cluster vulnerability.
func (resolver *clusterCVEResolver) ClusterCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "ClusterCount")
	return resolver.root.ClusterCount(resolver.clusterVulnerabilityScopeContext(ctx), args)
}

func (resolver *clusterCVEResolver) VulnerabilityType() string {
	return resolver.data.GetType().String()
}

func (resolver *clusterCVEResolver) VulnerabilityTypes() []string {
	return []string{resolver.data.GetType().String()}
}

func (resolver *clusterCVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "EnvImpact")
	allCount, err := resolver.root.ClusterCount(ctx, RawQuery{})
	if err != nil || allCount == 0 {
		return 0, err
	}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_CLUSTER_VULNERABILITIES,
	})
	scopedCount, err := resolver.root.ClusterCount(ctx, RawQuery{})
	if err != nil {
		return 0, err
	}
	return float64(scopedCount) / float64(allCount), nil
}

func (resolver *clusterCVEResolver) FixedByVersion(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "FixedByVersion")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_CLUSTERS)
	if !hasScope {
		return "", nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, scope.ID).AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
	edges, err := resolver.root.ClusterCVEEdgeDataStore.SearchRawEdges(resolver.ctx, query)
	if err != nil || len(edges) == 0 {
		return "", err
	}
	return edges[0].GetFixedBy(), nil
}

func (resolver *clusterCVEResolver) IsFixable(ctx context.Context, args RawQuery) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "IsFixable")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}
	query, err := args.AsV1QueryOrEmpty(search.ExcludeFieldLabel(search.CVEID))
	if err != nil {
		return false, err
	}
	// check for Fixable fields in args
	logErrorOnQueryContainingField(query, search.Fixable, "IsFixable")

	conjuncts := []*v1.Query{query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()}

	// check scoping, add as conjunction if needed
	if scope, ok := scoped.GetScope(resolver.ctx); !ok || scope.Level != v1.SearchCategory_CLUSTER_VULNERABILITIES {
		conjuncts = append(conjuncts, resolver.getClusterCVEQuery())
	}

	query = search.ConjunctionQuery(conjuncts...)
	loader, err := loaders.GetClusterCVELoader(resolver.ctx)
	if err != nil {
		return false, err
	}
	count, err := loader.CountFromQuery(resolver.ctx, query)
	if err != nil {
		return false, err
	}
	return count != 0, nil
}

func (resolver *clusterCVEResolver) LastScanned(_ context.Context) (*graphql.Time, error) {
	// TODO we're temporarily pointing it at LastModified until this information is actually in the data model
	return resolver.LastModified(resolver.ctx)
}

func (resolver *clusterCVEResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "Vectors")
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

func (resolver *clusterCVEResolver) CreatedAt(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetCreatedAt())
}

func (resolver *clusterCVEResolver) CVE(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetCve()
}

func (resolver *clusterCVEResolver) LastModified(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetLastModified())
}

func (resolver *clusterCVEResolver) Link(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetLink()
}

func (resolver *clusterCVEResolver) PublishedOn(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetPublishedOn())
}

func (resolver *clusterCVEResolver) ScoreVersion(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetScoreVersion().String()
}

func (resolver *clusterCVEResolver) Summary(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetSummary()
}

func (resolver *clusterCVEResolver) SuppressActivation(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetSnoozeStart())
}

func (resolver *clusterCVEResolver) SuppressExpiry(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetSnoozeExpiry())
}

func (resolver *clusterCVEResolver) Suppressed(_ context.Context) bool {
	return resolver.data.GetSnoozed()
}

func (resolver *clusterCVEResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}
