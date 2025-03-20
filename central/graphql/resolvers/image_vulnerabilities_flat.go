package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ImageVulnerabilityFlat",
			append(commonVulnerabilitySubResolvers,
				"activeState(query: String): ActiveState",
				"deploymentCount(query: String): Int!",
				"deployments(query: String, pagination: Pagination): [Deployment!]!",
				"discoveredAtImage(query: String): Time",
				"effectiveVulnerabilityRequest: VulnerabilityRequest",
				"exceptionCount(requestStatus: [String]): Int!",
				"imageComponentCount(query: String): Int!",
				"imageComponents(query: String, pagination: Pagination): [ImageComponent!]!",
				"imageCount(query: String): Int!",
				"images(query: String, pagination: Pagination): [Image!]!",
				"operatingSystem: String!",
				"vulnerabilityState: String!",
				"nvdCvss: Float!",
				"nvdScoreVersion: String!",
			)),
		schema.AddQuery("imageVulnerabilityFlat(id: ID): ImageVulnerabilityFlat"),
		schema.AddQuery("imageVulnerabilitiesFlat(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerabilityFlat!]!"),
		schema.AddQuery("imageVulnerabilityFlatCount(query: String): Int!"),
	)
}

// ImageVulnerabilityFlatResolver represents the supported API on image vulnerabilities
//
//	NOTE: This list is and should remain alphabetically ordered
type ImageVulnerabilityFlatResolver interface {
	CommonVulnerabilityResolver

	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	DiscoveredAtImage(ctx context.Context, args RawQuery) (*graphql.Time, error)
	EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error)
	ExceptionCount(ctx context.Context, args struct{ RequestStatus *[]*string }) (int32, error)
	ImageComponentCount(ctx context.Context, args RawQuery) (int32, error)
	ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error)
	ImageCount(ctx context.Context, args RawQuery) (int32, error)
	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	OperatingSystem(ctx context.Context) string
	VulnerabilityState(ctx context.Context) string
	Nvdcvss(ctx context.Context) float64
	NvdScoreVersion(ctx context.Context) string
}

// ImageVulnerabilityFlat returns a vulnerability of the given id
func (resolver *Resolver) ImageVulnerabilityFlat(ctx context.Context, args IDQuery) (ImageVulnerabilityFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerability")

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	if features.FlattenCVEData.Enabled() {
		// get loader
		loader, err := loaders.GetImageCVEV2Loader(ctx)
		if err != nil {
			return nil, err
		}

		ret, err := loader.FromID(ctx, string(*args.ID))
		return resolver.wrapImageCVEV2WithContext(ctx, ret, true, err)
	}

	// get loader
	loader, err := loaders.GetImageCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	ret, err := loader.FromID(ctx, string(*args.ID))
	return resolver.wrapImageCVEWithContext(ctx, ret, true, err)
}

// ImageVulnerabilitiesFlat resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilitiesFlat(ctx context.Context, q PaginatedQuery) ([]ImageVulnerabilityFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilities")

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// cast query
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	// get loader
	loader, err := loaders.GetImageCVEV2Loader(ctx)
	if err != nil {
		return nil, err
	}

	// get values
	// TODO(ROX-27780): figure out what to do with this
	//  query = tryUnsuppressedQuery(query)

	vulns, err := loader.FromQuery(ctx, query)
	cveResolvers, err := resolver.wrapImageCVEV2sFlatWithContext(ctx, vulns, err)
	if err != nil {
		return nil, err
	}

	// cast as return type
	ret := make([]ImageVulnerabilityFlatResolver, 0, len(cveResolvers))
	for _, res := range cveResolvers {
		ret = append(ret, res)
	}
	return ret, nil
}

// ImageVulnerabilityFlatCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilityFlatCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilityCount")
	// check permissions
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	// cast query
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	// get loader
	loader, err := loaders.GetImageCVEV2Loader(ctx)
	if err != nil {
		return 0, err
	}

	return loader.CountFromQuery(ctx, query)
}

// ImageVulnerabilityFlatCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ImageVulnerabilityFlatCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerabilityCounter")

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// cast query
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	// check for Fixable fields in args
	logErrorOnQueryContainingField(query, search.Fixable, "ImageVulnerabilityCounter")

	loader, err := loaders.GetImageCVEV2Loader(ctx)
	if err != nil {
		return nil, err
	}

	// get fixable vulns
	fixableQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := loader.FromQuery(ctx, fixableQuery)
	if err != nil {
		return nil, err
	}
	fixable := imageCveV2FlatToVulnerabilityWithSeverity(fixableVulns)

	// get unfixable vulns
	unFixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableVulns, err := loader.FromQuery(ctx, unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}
	unfixable := imageCveV2FlatToVulnerabilityWithSeverity(unFixableVulns)

	return mapCVEsToVulnerabilityCounter(fixable, unfixable), nil
}

// TopImageVulnerabilityFlat returns the most severe image vulnerability found in the scoped context
func (resolver *Resolver) TopImageVulnerabilityFlat(ctx context.Context, args RawQuery) (ImageVulnerabilityFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "TopImageVulnerability")

	// verify scoping
	scope, ok := scoped.GetScope(ctx)
	if !ok {
		return nil, errors.New("TopImageVulnerability called without scope context")
	} else if (scope.Level != v1.SearchCategory_IMAGE_COMPONENTS && scope.Level != v1.SearchCategory_IMAGE_COMPONENTS_V2) && scope.Level != v1.SearchCategory_IMAGES {
		return nil, errors.New("TopImageVulnerability called with improper scope context")
	}

	// form query
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.CVSS.String(),
				Reversed: true,
			},
			{
				Field:    search.CVE.String(),
				Reversed: true,
			},
		},
		Limit:  1,
		Offset: 0,
	}

	loader, err := loaders.GetImageCVEV2Loader(ctx)
	if err != nil {
		return nil, err
	}

	// invoke query
	topVuln, err := loader.FromQuery(ctx, query)
	if err != nil || len(topVuln) == 0 {
		return nil, err
	} else if len(topVuln) > 1 {
		return nil, errors.New("TopImageVulnerability query returned more than one vulnerabilities")
	}

	res, err := resolver.wrapImageCVEV2WithContext(ctx, topVuln[0], true, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

/*
Utility Functions
*/

func imageCveV2FlatToVulnerabilityWithSeverity(in []*storage.ImageCVEV2) []VulnerabilityWithSeverity {
	ret := make([]VulnerabilityWithSeverity, len(in))
	for i, vuln := range in {
		ret[i] = vuln
	}
	return ret
}

// Following are the functions that return information that is nested in the CVEInfo object
// or are convenience functions to allow time for UI to migrate to new naming schemes
// This is awful
func (resolver *imageCVEV2FlatResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(resolver.data.GetId())
}

func (resolver *imageCVEV2FlatResolver) CreatedAt(_ context.Context) (*graphql.Time, error) {
	return protocompat.ConvertTimestampToGraphqlTimeOrError(resolver.data.GetCveBaseInfo().GetCreatedAt())
}

func (resolver *imageCVEV2FlatResolver) CVE(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetCve()
}

func (resolver *imageCVEV2FlatResolver) LastModified(_ context.Context) (*graphql.Time, error) {
	return protocompat.ConvertTimestampToGraphqlTimeOrError(resolver.data.GetCveBaseInfo().GetLastModified())
}

func (resolver *imageCVEV2FlatResolver) Link(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetLink()
}

func (resolver *imageCVEV2FlatResolver) PublishedOn(_ context.Context) (*graphql.Time, error) {
	return protocompat.ConvertTimestampToGraphqlTimeOrError(resolver.data.GetCveBaseInfo().GetPublishedOn())
}

func (resolver *imageCVEV2FlatResolver) ScoreVersion(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetScoreVersion().String()
}

func (resolver *imageCVEV2FlatResolver) Summary(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetSummary()
}

func (resolver *imageCVEV2FlatResolver) SuppressActivation(_ context.Context) (*graphql.Time, error) {
	return nil, nil
}

func (resolver *imageCVEV2FlatResolver) SuppressExpiry(_ context.Context) (*graphql.Time, error) {
	return nil, nil
}

func (resolver *imageCVEV2FlatResolver) Suppressed(_ context.Context) bool {
	return false
}

func (resolver *imageCVEV2FlatResolver) EnvImpact(ctx context.Context) (float64, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "EnvImpact")
	allCount, err := resolver.root.DeploymentCount(ctx, RawQuery{})
	if err != nil || allCount == 0 {
		return 0, err
	}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES_V2,
	})
	scopedCount, err := resolver.root.DeploymentCount(ctx, RawQuery{})
	if err != nil {
		return 0, err
	}
	return float64(scopedCount) / float64(allCount), nil
}

func (resolver *imageCVEV2FlatResolver) FixedByVersion(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "FixedByVersion")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	if embeddedVuln := embeddedobjs.VulnFromContext(resolver.ctx); embeddedVuln != nil {
		return embeddedVuln.GetFixedBy(), nil
	}

	scope, hasScope := scoped.GetScope(resolver.ctx)
	if !hasScope {
		return "", nil
	}
	if scope.Level != v1.SearchCategory_IMAGE_COMPONENTS_V2 {
		return "", nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
	cves, err := resolver.root.ImageCVEV2DataStore.SearchRawImageCVEs(resolver.ctx, query)
	if err != nil || len(cves) == 0 {
		return "", err
	}
	return cves[0].GetFixedBy(), nil
}

// IsFixable returns if the CVE is fixable or not.
//
//	TODO(ROX-28123): Once the old code is removed, this method can become generated.
func (resolver *imageCVEV2FlatResolver) IsFixable(_ context.Context, _ RawQuery) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "IsFixable")
	return resolver.data.IsFixable, nil
}

func (resolver *imageCVEV2FlatResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "LastScanned")

	// Short path. Full image is embedded when image scan resolver is called.
	if scanTime := embeddedobjs.LastScannedFromContext(resolver.ctx); scanTime != nil {
		return &graphql.Time{Time: *scanTime}, nil
	}

	imageLoader, err := loaders.GetImageLoader(resolver.ctx)
	if err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, resolver.data.GetImageId()).ProtoQuery()

	images, err := imageLoader.FromQuery(resolver.imageVulnerabilityScopeContext(ctx), q)
	if err != nil || len(images) == 0 {
		return nil, err
	} else if len(images) > 1 {
		return nil, errors.New("multiple images matched for last scanned image vulnerability query")
	}

	return protocompat.ConvertTimestampToGraphqlTimeOrError(images[0].GetScan().GetScanTime())
}

func (resolver *imageCVEV2FlatResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "Vectors")
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

func (resolver *imageCVEV2FlatResolver) VulnerabilityState(ctx context.Context) string {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "VulnerabilityState")

	return resolver.data.GetState().String()
}

func (resolver *imageCVEV2FlatResolver) ActiveState(_ context.Context, _ RawQuery) (*activeStateResolver, error) {
	// TODO:  Verify Active Vuln Management is no more
	return nil, nil
}

func (resolver *imageCVEV2FlatResolver) EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "EffectiveVulnerabilityRequest")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	var imageID string
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		imageID = scope.ID
	}

	if imageID == "" {
		return nil, errors.Errorf("image scope must be provided for determining effective vulnerability request for cve %s", resolver.data.GetId())
	}
	imageLoader, err := loaders.GetImageLoader(resolver.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting image loader")
	}
	img, err := imageLoader.FromID(resolver.ctx, imageID)
	if err != nil {
		log.Error(errors.Wrapf(err, "fetching image with id %s", imageID))
		return nil, nil
	}

	req, err := resolver.root.vulnReqQueryMgr.EffectiveVulnReq(resolver.ctx, resolver.data.GetCveBaseInfo().GetCve(),
		common.VulnReqScope{
			Registry: img.GetName().GetRegistry(),
			Remote:   img.GetName().GetRemote(),
			Tag:      img.GetName().GetTag(),
		})
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapVulnerabilityRequest(req, nil)
}

func (resolver *imageCVEV2FlatResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEV2FlatResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "Deployments")
	return resolver.root.Deployments(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEV2FlatResolver) DiscoveredAtImage(_ context.Context, _ RawQuery) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "DiscoveredAtImage")
	return protocompat.ConvertTimestampToGraphqlTimeOrError(resolver.data.GetFirstImageOccurrence())
}

func (resolver *imageCVEV2FlatResolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ImageComponents")
	return resolver.root.ImageComponents(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEV2FlatResolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEV2FlatResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ImageCount")
	return resolver.root.ImageCount(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEV2FlatResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "Images")
	return resolver.root.Images(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEV2FlatResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

func (resolver *imageCVEV2FlatResolver) ExceptionCount(ctx context.Context, args struct{ RequestStatus *[]*string }) (int32, error) {
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
		cves:          []string{resolver.data.GetCveBaseInfo().GetCve()},
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

func (resolver *imageCVEV2FlatResolver) imageVulnerabilityScopeContext(ctx context.Context) context.Context {
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
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES_V2,
	})
}

type imageCVEV2FlatResolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.ImageCVEV2
}

func (resolver *Resolver) wrapImageCVEV2Flat(value *storage.ImageCVEV2, ok bool, err error) (*imageCVEV2FlatResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEV2FlatResolver{root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVEV2sFlat(values []*storage.ImageCVEV2, err error) ([]*imageCVEV2FlatResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVEV2FlatResolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEV2FlatResolver{root: resolver, data: v}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageCVEV2FlatWithContext(ctx context.Context, value *storage.ImageCVEV2, ok bool, err error) (*imageCVEV2FlatResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEV2FlatResolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVEV2sFlatWithContext(ctx context.Context, values []*storage.ImageCVEV2, err error) ([]*imageCVEV2FlatResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVEV2FlatResolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEV2FlatResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

func (resolver *imageCVEV2FlatResolver) ComponentId(ctx context.Context) string {
	value := resolver.data.GetComponentId()
	return value
}

func (resolver *imageCVEV2FlatResolver) CveBaseInfo(ctx context.Context) (*cVEInfoResolver, error) {
	value := resolver.data.GetCveBaseInfo()
	return resolver.root.wrapCVEInfo(value, true, nil)
}

func (resolver *imageCVEV2FlatResolver) Cvss(ctx context.Context) float64 {
	value := resolver.data.GetCvss()
	return float64(value)
}

func (resolver *imageCVEV2FlatResolver) FirstImageOccurrence(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetFirstImageOccurrence()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageCVEV2FlatResolver) Id(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *imageCVEV2FlatResolver) ImageId(ctx context.Context) string {
	value := resolver.data.GetImageId()
	return value
}

func (resolver *imageCVEV2FlatResolver) ImpactScore(ctx context.Context) float64 {
	value := resolver.data.GetImpactScore()
	return float64(value)
}

func (resolver *imageCVEV2FlatResolver) NvdScoreVersion(ctx context.Context) string {
	value := resolver.data.GetNvdScoreVersion()
	return value.String()
}

func (resolver *imageCVEV2FlatResolver) Nvdcvss(ctx context.Context) float64 {
	value := resolver.data.GetNvdcvss()
	return float64(value)
}

func (resolver *imageCVEV2FlatResolver) OperatingSystem(ctx context.Context) string {
	value := resolver.data.GetOperatingSystem()
	return value
}

func (resolver *imageCVEV2FlatResolver) Severity(ctx context.Context) string {
	value := resolver.data.GetSeverity()
	return value.String()
}

func (resolver *imageCVEV2FlatResolver) State(ctx context.Context) string {
	value := resolver.data.GetState()
	return value.String()
}
