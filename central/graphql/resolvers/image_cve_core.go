package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ImageCVECore",
			[]string{
				"affectedImageCount: Int!",
				"affectedImageCountBySeverity: ResourceCountByCVESeverity!",
				"cve: String!",
				"distroTuples: [ImageVulnerability!]!",
				"firstDiscoveredInSystem: Time",
				"topCVSS: Float!",
			}),
		schema.AddQuery("imageCVECount(query: String): Int!"),
		schema.AddQuery("imageCVEs(query: String, pagination: Pagination): [ImageCVECore!]!"),
		// `subfieldScopeQuery` applies the scope query to all the subfields of the ImageCVE resolver.
		// This eliminates the need to pass queries to individual resolvers.
		schema.AddQuery("imageCVE(cve: String, subfieldScopeQuery: String): ImageCVECore"),
	)
}

type imageCVECoreResolver struct {
	ctx  context.Context
	root *Resolver
	data imagecve.CveCore
}

func (resolver *Resolver) wrapImageCVECoreWithContext(ctx context.Context, value imagecve.CveCore, err error) (*imageCVECoreResolver, error) {
	if err != nil || value == nil {
		return nil, err
	}
	return &imageCVECoreResolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVECoresWithContext(ctx context.Context, values []imagecve.CveCore, err error) ([]*imageCVECoreResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVECoreResolver, len(values))
	for i, v := range values {
		output[i] = &imageCVECoreResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

// ImageCVECount returns the count of image cves satisfying the specified query.
// Note: Client must explicitly pass observed/deferred CVEs.
func (resolver *Resolver) ImageCVECount(ctx context.Context, q RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageCVEs")

	if !features.VulnMgmtWorkloadCVEs.Enabled() {
		return 0, errors.Errorf("%s=false. Set %s=true and retry", features.VulnMgmtWorkloadCVEs.Name(), features.VulnMgmtWorkloadCVEs.Name())
	}
	if err := readImages(ctx); err != nil {
		return 0, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	count, err := resolver.ImageCVEView.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// ImageCVEs returns graphQL resolver for image cves satisfying the specified query.
// Note: Client must explicitly pass observed/deferred CVEs.
func (resolver *Resolver) ImageCVEs(ctx context.Context, q PaginatedQuery) ([]*imageCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageCVEs")

	if !features.VulnMgmtWorkloadCVEs.Enabled() {
		return nil, errors.Errorf("%s=false. Set %s=true and retry", features.VulnMgmtWorkloadCVEs.Name(), features.VulnMgmtWorkloadCVEs.Name())
	}
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	cves, err := resolver.ImageCVEView.Get(ctx, query, views.ReadOptions{})
	return resolver.wrapImageCVECoresWithContext(ctx, cves, err)
}

func (resolver *imageCVECoreResolver) AffectedImageCount(_ context.Context) int32 {
	return int32(resolver.data.GetAffectedImages())
}

func (resolver *imageCVECoreResolver) AffectedImageCountBySeverity(ctx context.Context) (*resourceCountBySeverityResolver, error) {
	return resolver.root.wrapResourceCountByCVESeverityWithContext(ctx, resolver.data.GetImagesBySeverity(), nil)
}

func (resolver *imageCVECoreResolver) CVE(_ context.Context) string {
	return resolver.data.GetCVE()
}

func (resolver *imageCVECoreResolver) DistroTuples(ctx context.Context) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVECore, "DistroTuples")
	q := PaginatedQuery{
		Query: pointers.String(search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetCVEIDs()...).Query()),
	}
	return resolver.root.ImageVulnerabilities(ctx, q)
}

func (resolver *imageCVECoreResolver) FirstDiscoveredInSystem(_ context.Context) *graphql.Time {
	return &graphql.Time{
		Time: resolver.data.GetFirstDiscoveredInSystem(),
	}
}

func (resolver *imageCVECoreResolver) TopCVSS(_ context.Context) float64 {
	return float64(resolver.data.GetTopCVSS())
}

// ImageCVE returns graphQL resolver for specified image cve.
func (resolver *Resolver) ImageCVE(ctx context.Context, args struct {
	Cve                *string
	SubfieldScopeQuery *string
}) (*imageCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageCVEMetadata")

	if !features.VulnMgmtWorkloadCVEs.Enabled() {
		return nil, errors.Errorf("%s=false. Set %s=true and retry", features.VulnMgmtWorkloadCVEs.Name(), features.VulnMgmtWorkloadCVEs.Name())
	}
	if err := readImages(ctx); err != nil {
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

	cves, err := resolver.ImageCVEView.Get(ctx, query, views.ReadOptions{})
	if len(cves) == 0 {
		return nil, nil
	}
	if len(cves) > 1 {
		utils.Should(errors.Errorf("Retrieved multiple rows when only one row is expected for CVE=%s query", *args.Cve))
		return nil, err
	}
	return resolver.wrapImageCVECoreWithContext(ctx, cves[0], err)
}
