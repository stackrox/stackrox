package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedVulnerability", []string{
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
			"createdAt: Time",
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
			"suppressed: Boolean!",
			"suppressActivation: Time",
			"suppressExpiry: Time",
		}),
		schema.AddQuery("vulnerability(id: ID): EmbeddedVulnerability"),
		schema.AddQuery("vulnerabilities(query: String, pagination: Pagination): [EmbeddedVulnerability!]!"),
		schema.AddQuery("vulnerabilityCount(query: String): Int!"),
		schema.AddQuery("k8sVulnerability(id: ID): EmbeddedVulnerability"),
		schema.AddQuery("k8sVulnerabilities(query: String, pagination: Pagination): [EmbeddedVulnerability!]!"),
		schema.AddQuery("istioVulnerability(id: ID): EmbeddedVulnerability"),
		schema.AddQuery("istioVulnerabilities(query: String, pagination: Pagination): [EmbeddedVulnerability!]!"),
		schema.AddExtraResolver("EmbeddedVulnerability", `unusedVarSink(query: String): Int`),
	)
}

// VulnerabilityResolver represents a generic resolver of vulnerability fields.
// Values may come from either an embedded vulnerability context, or a top level vulnerability context.
type VulnerabilityResolver interface {
	ID(ctx context.Context) graphql.ID
	Cve(ctx context.Context) string
	Cvss(ctx context.Context) float64
	Link(ctx context.Context) string
	Summary(ctx context.Context) string
	EnvImpact(ctx context.Context) (float64, error)
	ImpactScore(ctx context.Context) float64
	ScoreVersion(ctx context.Context) string
	FixedByVersion(ctx context.Context) (string, error)
	IsFixable(ctx context.Context, args RawQuery) (bool, error)
	PublishedOn(ctx context.Context) (*graphql.Time, error)
	CreatedAt(ctx context.Context) (*graphql.Time, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	LastModified(ctx context.Context) (*graphql.Time, error)
	Vectors() *EmbeddedVulnerabilityVectorsResolver
	Severity(ctx context.Context) string
	VulnerabilityType() string

	Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error)
	ComponentCount(ctx context.Context, args RawQuery) (int32, error)

	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)

	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32

	Suppressed(ctx context.Context) bool
	SuppressActivation(ctx context.Context) (*graphql.Time, error)
	SuppressExpiry(ctx context.Context) (*graphql.Time, error)
}

// Vulnerability resolves a single vulnerability based on an id (the CVE value).
func (resolver *Resolver) Vulnerability(ctx context.Context, args idQuery) (VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Vulnerability")
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	if features.Dackbox.Enabled() {
		return resolver.vulnerabilityV2(ctx, args)
	}
	return resolver.vulnerabilityV1(ctx, args)
}

// Vulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) Vulnerabilities(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Vulnerabilities")
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	if features.Dackbox.Enabled() {
		return resolver.vulnerabilitiesV2(ctx, q)
	}
	return resolver.vulnerabilitiesV1(ctx, q)
}

// VulnerabilityCount returns count of all clusters across infrastructure
func (resolver *Resolver) VulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnerabilityCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}
	if features.Dackbox.Enabled() {
		return resolver.vulnerabilityCountV2(ctx, args)
	}
	return resolver.vulnerabilityCountV1(ctx, args)
}

// VulnCounter returns a VulnerabilityCounterResolver for the input query.s
func (resolver *Resolver) VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Cluster, "VulnCounter")
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	if features.Dackbox.Enabled() {
		return resolver.vulnCounterV2(ctx, args)
	}
	return resolver.vulnCounterV1(ctx, args)
}

// Legacy K8s and Istio specific vuln resolvers.
// These can be replaced by hitting the basic vuln resolvers with a query for the K8s or Istio type.
////////////////////////////////////////////////////////////////////////////////////////////////////

// K8sVulnerability resolves a single k8s vulnerability based on an id (the CVE value).
func (resolver *Resolver) K8sVulnerability(ctx context.Context, args idQuery) (VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sVulnerability")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		return resolver.k8sVulnerabilityV2(ctx, args)
	}
	return resolver.k8sVulnerabilityV1(ctx, args)
}

// K8sVulnerabilities resolves a set of k8s vulnerabilities based on a query.
func (resolver *Resolver) K8sVulnerabilities(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "K8sVulnerabilities")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		return resolver.k8sVulnerabilitiesV2(ctx, args)
	}
	return resolver.k8sVulnerabilitiesV1(ctx, args)
}

// IstioVulnerability resolves a single istio vulnerability based on an id (the CVE value).
func (resolver *Resolver) IstioVulnerability(ctx context.Context, args idQuery) (VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioVulnerability")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		return resolver.istioVulnerabilityV2(ctx, args)
	}
	return resolver.istioVulnerabilityV1(ctx, args)
}

// IstioVulnerabilities resolves a set of istio vulnerabilities based on a query.
func (resolver *Resolver) IstioVulnerabilities(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "IstioVulnerabilities")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		return resolver.istioVulnerabilitiesV2(ctx, args)
	}

	return resolver.istioVulnerabilitiesV1(ctx, args)
}

func tryUnsuppressedQuery(q *v1.Query) *v1.Query {
	var suppressedSet bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.CVESuppressed.String() && mfQ.MatchFieldQuery.GetValue() == "true" {
			suppressedSet = true
		}
	})
	if suppressedSet {
		return q
	}

	local := q.Clone()
	pagination := local.GetPagination()
	local.Pagination = nil
	local = search.NewConjunctionQuery(local, search.NewQueryBuilder().AddBools(search.CVESuppressed, false).ProtoQuery())
	local.Pagination = pagination
	return local
}
