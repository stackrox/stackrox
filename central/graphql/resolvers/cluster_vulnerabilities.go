package resolvers

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ClusterVulnerability",
			append(commonVulnerabilitySubResolvers,
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
//  NOTE: This list is and should remain alphabetically ordered
type ClusterVulnerabilityResolver interface {
	CommonVulnerabilityResolver

	VulnerabilityType() string
	VulnerabilityTypes() []string
}

// ClusterVulnerability returns a vulnerability of the given id
func (resolver *Resolver) ClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ClusterVulnerability does not support postgres yet")
}

// ClusterVulnerabilities resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ClusterVulnerabilities(ctx context.Context, q PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withClusterTypeFiltering(q.String())
		return resolver.clusterVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ClusterVulnerabilities does not support postgres yet")
}

// ClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withClusterTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver ClusterVulnerabilityCount does not support postgres yet")
}

// ClusterVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ClusterVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withClusterTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver ClusterVulnerabilityCounter does not support postgres yet")
}

// K8sClusterVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) K8sClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sClusterVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver K8sClusterVulnerability does not support postgres yet")
}

// K8sClusterVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) K8sClusterVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sClusterVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withK8sTypeFiltering(args.String())
		return resolver.clusterVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver K8sClusterVulnerabilities does not support postgres yet")
}

// K8sClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) K8sClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sClusterVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withK8sTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver K8sClusterVulnerabilityCount does not support postgres yet")
}

// IstioClusterVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) IstioClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioClusterVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver IstioClusterVulnerability does not support postgres yet")
}

// IstioClusterVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) IstioClusterVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioClusterVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withIstioTypeFiltering(args.String())
		return resolver.clusterVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver IstioClusterVulnerabilities does not support postgres yet")
}

// IstioClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) IstioClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioClusterVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withIstioTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver IstioClusterVulnerabilityCount does not support postgres yet")
}

// OpenShiftClusterVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) OpenShiftClusterVulnerability(ctx context.Context, args IDQuery) (ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "OpenShiftClusterVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	// TODO add postgres support
	return nil, errors.New("Resolver OpenShiftClusterVulnerability does not support postgres yet")
}

// OpenShiftClusterVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) OpenShiftClusterVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "OpenShiftClusterVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withOpenShiftTypeFiltering(args.String())
		return resolver.clusterVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
	}
	// TODO add postgres support
	return nil, errors.New("Resolver OpenShiftClusterVulnerabilities does not support postgres yet")
}

// OpenShiftClusterVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) OpenShiftClusterVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "OpenShiftClusterVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withOpenShiftTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO add postgres support
	return 0, errors.New("Resolver OpenShiftClusterVulnerabilityCount does not support postgres yet")
}

// withClusterTypeFiltering adds a conjunction as a raw query to filter vulnerability type by cluster
// this is needed to support pre postgres requests
func withClusterTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType,
			storage.CVE_ISTIO_CVE.String(),
			storage.CVE_OPENSHIFT_CVE.String(),
			storage.CVE_K8S_CVE.String()).Query())
}

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
