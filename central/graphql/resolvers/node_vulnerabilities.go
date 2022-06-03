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
	// TODO : Add postgres support
	return nil, errors.New("Resolver NodeVulnerability does not support postgres yet")
}

// NodeVulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) NodeVulnerabilities(ctx context.Context, q PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeTypeFiltering(q.String())
		return resolver.nodeVulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}
	// TODO : Add postgres support
	return nil, errors.New("Resolver NodeVulnerabilities does not support postgres yet")
}

// NodeVulnerabilityCount returns count of all clusters across infrastructure
func (resolver *Resolver) NodeVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeTypeFiltering(args.String())
		return resolver.vulnerabilityCountV2(ctx, RawQuery{Query: &query})
	}
	// TODO : Add postgres support
	return 0, errors.New("Resolver NodeVulnerabilityCount does not support postgres yet")
}

// NodeVulnCounter returns a VulnerabilityCounterResolver for the input query.s
func (resolver *Resolver) NodeVulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		query := withNodeTypeFiltering(args.String())
		return resolver.vulnCounterV2(ctx, RawQuery{Query: &query})
	}
	// TODO : Add postgres support
	return nil, errors.New("Resolver NodeVulnCounter does not support postgres yet")
}

// withNodeTypeFiltering adds a conjunction as a raw query to filter vulns by CVEType Node
func withNodeTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_NODE_CVE.String()).Query())
}
