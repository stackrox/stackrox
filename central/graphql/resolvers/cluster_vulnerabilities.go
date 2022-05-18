package resolvers

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ClusterVulnerability",
			append(commonVulnerabilitySubResolvers,
				"vulnerabilityType: String!",
				"vulnerabilityTypes: [String!]!",
			)),
		schema.AddQuery("clusterVulnerability(id: ID): ClusterVulnerability"),
		schema.AddQuery("clusterVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ClusterVulnerability!]!"),
		schema.AddQuery("clusterVulnerabilityCount(query: String): Int!"),
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

// withClusterTypeFiltering adds a conjunction as a raw query to filter vulnerability type by cluster
// this is needed to support pre postgres requests
func withClusterTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType,
			storage.CVE_ISTIO_CVE.String(),
			storage.CVE_OPENSHIFT_CVE.String(),
			storage.CVE_K8S_CVE.String()).Query())
}
