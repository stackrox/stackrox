package resolvers

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedImageVulnerability", []string{
			"id: ID!",
			"cve: String!",
			"cvss: Float!",
			"scoreVersion: String!",
			"vectors: EmbeddedVulnerabilityVectors",
			"link: String!",
			"summary: String!",
			"fixedByVersion: String!",
			"isFixable(query: String): Boolean!",
			"lastScanned: Time",
			"createdAt: Time", // Discovered At System
			"discoveredAtImage(query: String): Time",
			"components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!",
			"componentCount(query: String): Int!",
			"images(query: String, pagination: Pagination): [Image!]!",
			"imageCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"deploymentCount(query: String): Int!",
			"envImpact: Float!",
			"severity: String!",
			"publishedOn: Time",
			"lastModified: Time",
			"impactScore: Float!",
			"vulnerabilityType: String!",
			"vulnerabilityTypes: [String!]!",
			"suppressed: Boolean!",
			"suppressActivation: Time",
			"suppressExpiry: Time",
			"activeState(query: String): ActiveState",
			"vulnerabilityState: String!",
			"effectiveVulnerabilityRequest: VulnerabilityRequest",
		}),
		schema.AddQuery("imageVulnerability(id: ID): EmbeddedImageVulnerability"),
		schema.AddQuery("imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedImageVulnerability!]!"),
		schema.AddQuery("imageVulnerabilityCount(query: String): Int!"),
		schema.AddExtraResolver("EmbeddedImageVulnerability", `unusedVarSink(query: String): Int`),
	)
}

func (resolver *Resolver) ImageVulnerability(ctx context.Context, args IDQuery) (VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerability")
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}

	log.Errorf("osward -- ImageVulnerability")
	vuln, exists, err := resolver.CVEDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.Errorf("image cve not found: %s", string(*args.ID))
	}
	log.Errorf("osward -- vuln.GetType %s", vuln.GetType())
	vulnResolver, err := resolver.wrapCVE(vuln, true, nil)
	if err != nil {
		return nil, err
	}
	vulnResolver.ctx = ctx
	return vulnResolver, nil
}

// ImageVulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) ImageVulnerabilities(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Vulnerabilities")
	log.Errorf("osward -- ImageVulnerabilities")
	return resolver.vulnerabilitiesV2(ctx, q)
}

// ImageVulnerabilityCount returns count of all clusters across infrastructure
func (resolver *Resolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	log.Errorf("osward -- ImageVulnerabilityCount")
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "VulnerabilityCount")
	return resolver.vulnerabilityCountV2(ctx, args)
}

// ImageVulnCounter returns a VulnerabilityCounterResolver for the input query.s
func (resolver *Resolver) ImageVulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	log.Errorf("osward -- ImageVulnCounter")
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "VulnCounter")
	return resolver.vulnCounterV2(ctx, args)
}
