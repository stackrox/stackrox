package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/imagecve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxDeployments = 1000
	maxImages      = 1000
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
				"deployments(pagination: Pagination): [Deployment!]!",
				"distroTuples: [ImageVulnerability!]!",
				"firstDiscoveredInSystem: Time",
				"exceptionCount(requestStatus: [String]): Int!",
				"images(pagination: Pagination): [Image!]!",
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

	subFieldQuery *v1.Query
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
	ret, err := resolver.wrapImageCVECoresWithContext(ctx, cves, err)
	if err != nil {
		return nil, err
	}
	for _, r := range ret {
		r.subFieldQuery = query
	}

	return ret, nil
}

func (resolver *imageCVECoreResolver) AffectedImageCount(_ context.Context) int32 {
	return int32(resolver.data.GetAffectedImageCount())
}

func (resolver *imageCVECoreResolver) AffectedImageCountBySeverity(ctx context.Context) (*resourceCountBySeverityResolver, error) {
	return resolver.root.wrapResourceCountByCVESeverityWithContext(ctx, resolver.data.GetImagesBySeverity(), nil)
}

func (resolver *imageCVECoreResolver) CVE(_ context.Context) string {
	return resolver.data.GetCVE()
}

func (resolver *imageCVECoreResolver) Deployments(ctx context.Context, args struct{ Pagination *inputtypes.Pagination }) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVECore, "Deployments")

	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	// Get full query for deployments
	query := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetCVE()).ProtoQuery()
	if resolver.subFieldQuery != nil {
		query = search.ConjunctionQuery(query, resolver.subFieldQuery)
	}
	if args.Pagination != nil {
		paginated.FillPagination(query, args.Pagination.AsV1Pagination(), maxDeployments)
	}

	// ROX-17254: Because of the incompatibility between
	// the data model and search framework, run the query through on CVE datastore through SQF.
	deploymentIDs, err := resolver.root.ImageCVEView.GetDeploymentIDs(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(deploymentIDs) == 0 {
		return nil, nil
	}

	depQ := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentIDs...).Query()
	return resolver.root.Deployments(ctx, PaginatedQuery{
		Query:      pointers.String(depQ),
		Pagination: args.Pagination,
	})
}

func (resolver *imageCVECoreResolver) DistroTuples(ctx context.Context) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVECore, "DistroTuples")
	// ImageVulnerabilities resolver filters out snoozed CVEs when no explicit filter by CVESuppressed is provided.
	// When ImageVulnerabilities resolver is called from here, it is to get the details of a single CVE which cannot be
	// obtained via SQF. So, the auto removal of snoozed CVEs is unintentional here. Hence, we add explicit filter with
	// CVESuppressed == true OR false
	q := PaginatedQuery{
		Query: pointers.String(search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetCVEIDs()...).
			AddBools(search.CVESuppressed, true, false).
			Query()),
	}
	return resolver.root.ImageVulnerabilities(ctx, q)
}

func (resolver *imageCVECoreResolver) FirstDiscoveredInSystem(_ context.Context) *graphql.Time {
	ts := resolver.data.GetFirstDiscoveredInSystem()
	if ts == nil {
		return nil
	}
	return &graphql.Time{
		Time: *ts,
	}
}

func (resolver *imageCVECoreResolver) ExceptionCount(ctx context.Context, args struct{ RequestStatus *[]*string }) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ExceptionCount")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	var requestStatusArr []string
	if args.RequestStatus != nil {
		for _, status := range *args.RequestStatus {
			if status != nil {
				requestStatusArr = append(requestStatusArr, *status)
			}
		}
	}
	filters := exceptionQueryFilters{
		cves:          []string{resolver.data.GetCVE()},
		requestStates: requestStatusArr,
	}
	q, err := unExpiredExceptionQuery(resolver.ctx, filters)
	if err != nil {
		return 0, err
	}

	count, err := resolver.root.vulnReqStore.Count(ctx, q)
	if err != nil {
		if errors.Is(err, errox.NotAuthorized) {
			return 0, nil
		}
		return 0, err
	}
	return int32(count), nil
}

func (resolver *imageCVECoreResolver) Images(ctx context.Context, args struct{ Pagination *inputtypes.Pagination }) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVECore, "Images")

	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// Get full query for deployments
	query := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetCVE()).ProtoQuery()
	if resolver.subFieldQuery != nil {
		query = search.ConjunctionQuery(query, resolver.subFieldQuery)
	}
	if args.Pagination != nil {
		paginated.FillPagination(query, args.Pagination.AsV1Pagination(), maxImages)
	}

	// ROX-17254: Because of the incompatibility between
	// the data model and search framework, run the query through on CVE datastore through SQF.
	imageIDs, err := resolver.root.ImageCVEView.GetImageIDs(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(imageIDs) == 0 {
		return nil, nil
	}

	imageQ := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageIDs...).Query()
	return resolver.root.Images(ctx, PaginatedQuery{
		Query:      pointers.String(imageQ),
		Pagination: args.Pagination,
	})
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
	ret, err := resolver.wrapImageCVECoreWithContext(ctx, cves[0], err)
	if err != nil {
		return nil, err
	}
	ret.subFieldQuery = query

	return ret, nil
}
