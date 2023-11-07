package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/vulnerabilityrequest/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ImageVulnerability",
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
			)),
		schema.AddQuery("imageVulnerability(id: ID): ImageVulnerability"),
		schema.AddQuery("imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability!]!"),
		schema.AddQuery("imageVulnerabilityCount(query: String): Int!"),
	)
}

// ImageVulnerabilityResolver represents the supported API on image vulnerabilities
//
//	NOTE: This list is and should remain alphabetically ordered
type ImageVulnerabilityResolver interface {
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
}

// ImageVulnerability returns a vulnerability of the given id
func (resolver *Resolver) ImageVulnerability(ctx context.Context, args IDQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageVulnerability")

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// get loader
	loader, err := loaders.GetImageCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	ret, err := loader.FromID(ctx, string(*args.ID))
	return resolver.wrapImageCVEWithContext(ctx, ret, true, err)
}

// ImageVulnerabilities resolves a set of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilities(ctx context.Context, q PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
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
	loader, err := loaders.GetImageCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	// get values
	query = tryUnsuppressedQuery(query)
	vulns, err := loader.FromQuery(ctx, query)
	cveResolvers, err := resolver.wrapImageCVEsWithContext(ctx, vulns, err)
	if err != nil {
		return nil, err
	}

	// cast as return type
	ret := make([]ImageVulnerabilityResolver, 0, len(cveResolvers))
	for _, res := range cveResolvers {
		ret = append(ret, res)
	}
	return ret, nil
}

// ImageVulnerabilityCount returns count of image vulnerabilities for the input query
func (resolver *Resolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
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
	loader, err := loaders.GetImageCVELoader(ctx)
	if err != nil {
		return 0, err
	}
	query = tryUnsuppressedQuery(query)

	return loader.CountFromQuery(ctx, query)
}

// ImageVulnerabilityCounter returns a VulnerabilityCounterResolver for the input query
func (resolver *Resolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
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

	// get loader
	loader, err := loaders.GetImageCVELoader(ctx)
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
	fixable := imageCveToVulnerabilityWithSeverity(fixableVulns)

	// get unfixable vulns
	unFixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableVulns, err := loader.FromQuery(ctx, unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}
	unfixable := imageCveToVulnerabilityWithSeverity(unFixableVulns)

	return mapCVEsToVulnerabilityCounter(fixable, unfixable), nil
}

// TopImageVulnerability returns the most severe image vulnerability found in the scoped context
func (resolver *Resolver) TopImageVulnerability(ctx context.Context, args RawQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "TopImageVulnerability")

	// verify scoping
	scope, ok := scoped.GetScope(ctx)
	if !ok {
		return nil, errors.New("TopImageVulnerability called without scope context")
	} else if scope.Level != v1.SearchCategory_IMAGE_COMPONENTS && scope.Level != v1.SearchCategory_IMAGES {
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

	// get loader
	loader, err := loaders.GetImageCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)

	// invoke query
	topVuln, err := loader.FromQuery(ctx, query)
	if err != nil || len(topVuln) == 0 {
		return nil, err
	} else if len(topVuln) > 1 {
		return nil, errors.New("TopImageVulnerability query returned more than one vulnerabilities")
	}

	res, err := resolver.wrapImageCVEWithContext(ctx, topVuln[0], true, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

/*
Utility Functions
*/

func imageCveToVulnerabilityWithSeverity(in []*storage.ImageCVE) []VulnerabilityWithSeverity {
	ret := make([]VulnerabilityWithSeverity, len(in))
	for i, vuln := range in {
		ret[i] = vuln
	}
	return ret
}

func (resolver *imageCVEResolver) imageVulnerabilityScopeContext(ctx context.Context) context.Context {
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
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES,
	})
}

func (resolver *imageCVEResolver) getImageCVEQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *imageCVEResolver) getImageCVERawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).Query()
}

// withImageCveTypeFiltering adds a conjunction as a raw query to filter vulnerability type by image
// this is needed to support pre postgres requests
func withImageCveTypeFiltering(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_IMAGE_CVE.String()).Query())
}

/*
Sub Resolver Functions
*/

func (resolver *imageCVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "EnvImpact")
	allCount, err := resolver.root.DeploymentCount(ctx, RawQuery{})
	if err != nil || allCount == 0 {
		return 0, err
	}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES,
	})
	scopedCount, err := resolver.root.DeploymentCount(ctx, RawQuery{})
	if err != nil {
		return 0, err
	}
	return float64(scopedCount) / float64(allCount), nil
}

func (resolver *imageCVEResolver) FixedByVersion(ctx context.Context) (string, error) {
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
	if scope.Level != v1.SearchCategory_IMAGE_COMPONENTS {
		return "", nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ComponentID, scope.ID).AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
	edges, err := resolver.root.ComponentCVEEdgeDataStore.SearchRawEdges(resolver.ctx, query)
	if err != nil || len(edges) == 0 {
		return "", err
	}
	return edges[0].GetFixedBy(), nil
}

func (resolver *imageCVEResolver) IsFixable(ctx context.Context, args RawQuery) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "IsFixable")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	if embeddedVuln := embeddedobjs.VulnFromContext(resolver.ctx); embeddedVuln != nil {
		return embeddedVuln.GetFixedBy() != "", nil
	}

	query, err := args.AsV1QueryOrEmpty(search.ExcludeFieldLabel(search.CVEID))
	if err != nil {
		return false, err
	}
	// check for Fixable fields in args
	logErrorOnQueryContainingField(query, search.Fixable, "IsFixable")

	conjuncts := []*v1.Query{query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()}

	// check scoping, add as conjunction if needed
	if scope, ok := scoped.GetScope(resolver.ctx); !ok || scope.Level != v1.SearchCategory_IMAGE_VULNERABILITIES {
		conjuncts = append(conjuncts, resolver.getImageCVEQuery())
	}

	query = search.ConjunctionQuery(conjuncts...)
	loader, err := loaders.GetImageCVELoader(resolver.ctx)
	if err != nil {
		return false, err
	}
	count, err := loader.CountFromQuery(resolver.ctx, query)
	if err != nil {
		return false, err
	}
	return count != 0, nil
}

func (resolver *imageCVEResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "LastScanned")

	// Short path. Full image is embedded when image scan resolver is called.
	if scanTime := embeddedobjs.LastScannedFromContext(resolver.ctx); scanTime != nil {
		return timestamp(scanTime)
	}

	imageLoader, err := loaders.GetImageLoader(resolver.ctx)
	if err != nil {
		return nil, err
	}

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit:  1,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.ImageScanTime.String(),
				Reversed: true,
			},
		},
	}

	images, err := imageLoader.FromQuery(resolver.imageVulnerabilityScopeContext(ctx), q)
	if err != nil || len(images) == 0 {
		return nil, err
	} else if len(images) > 1 {
		return nil, errors.New("multiple images matched for last scanned image vulnerability query")
	}

	return timestamp(images[0].GetScan().GetScanTime())
}

func (resolver *imageCVEResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
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

func (resolver *imageCVEResolver) VulnerabilityState(ctx context.Context) string {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "VulnerabilityState")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	if embeddedVuln := embeddedobjs.VulnFromContext(resolver.ctx); embeddedVuln != nil {
		return embeddedVuln.GetState().String()
	}

	if resolver.data.GetSnoozed() {
		return storage.VulnerabilityState_DEFERRED.String()
	}

	var imageID string
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		imageID = scope.ID
	}

	if imageID == "" {
		return ""
	}

	imageLoader, err := loaders.GetImageLoader(resolver.ctx)
	if err != nil {
		log.Error(errors.Wrap(err, "getting image loader"))
		return ""
	}
	img, err := imageLoader.FromID(resolver.ctx, imageID)
	if err != nil {
		log.Error(errors.Wrapf(err, "fetching image with id %s", imageID))
		return ""
	}

	states, err := resolver.root.vulnReqQueryMgr.VulnsWithState(resolver.ctx,
		common.VulnReqScope{
			Registry: img.GetName().GetRegistry(),
			Remote:   img.GetName().GetRemote(),
			Tag:      img.GetName().GetTag(),
		})
	if err != nil {
		log.Error(errors.Wrapf(err, "fetching vuln requests for image %s/%s:%s", img.GetName().GetRegistry(), img.GetName().GetRemote(), img.GetName().GetTag()))
		return ""
	}
	if s, ok := states[resolver.data.GetCveBaseInfo().GetCve()]; ok {
		return s.String()
	}

	return storage.VulnerabilityState_OBSERVED.String()
}

func (resolver *imageCVEResolver) ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error) {
	if !features.ActiveVulnMgmt.Enabled() {
		return &activeStateResolver{}, nil
	}
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ActiveState")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	scopeQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	deploymentID := getDeploymentScope(scopeQuery, ctx, resolver.ctx)
	if deploymentID == "" {
		return nil, nil
	}
	// We only support OS level component. The active state is not determined if there is no OS level component associate with this vuln.
	query := search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.data.GetId()).AddStrings(search.ComponentSource, storage.SourceType_OS.String()).ProtoQuery()
	osLevelComponents, err := resolver.root.ImageComponentDataStore.Count(resolver.ctx, query)
	if err != nil {
		return nil, err
	}
	if osLevelComponents == 0 {
		return &activeStateResolver{root: resolver.root, state: Undetermined}, nil
	}

	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentID)
	imageID := getImageIDFromQuery(scopeQuery)
	if imageID != "" {
		qb.AddExactMatches(search.ImageSHA, imageID)
	}
	query = search.ConjunctionQuery(resolver.getImageCVEQuery(), qb.ProtoQuery())

	results, err := resolver.root.ActiveComponent.Search(resolver.ctx, query)
	if err != nil {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	state := Inactive
	if len(ids) != 0 {
		state = Active
	}
	return &activeStateResolver{root: resolver.root, state: state, activeComponentIDs: ids, imageScope: imageID}, nil
}

func (resolver *imageCVEResolver) EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error) {
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

func (resolver *imageCVEResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "Deployments")
	return resolver.root.Deployments(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEResolver) DiscoveredAtImage(ctx context.Context, args RawQuery) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "DiscoveredAtImage")

	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	if embeddedVuln := embeddedobjs.VulnFromContext(resolver.ctx); embeddedVuln != nil {
		return timestamp(embeddedVuln.GetFirstImageOccurrence())
	}

	var imageID string
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		imageID = scope.ID
	} else {
		var err error
		imageID, err = getImageIDFromIfImageShaQuery(resolver.ctx, resolver.root, args)
		if err != nil {
			return nil, errors.Wrap(err, "could not determine vulnerability discovered time in image")
		}
	}

	if imageID == "" {
		return nil, nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageID).AddExactMatches(search.CVEID, resolver.data.GetId()).ProtoQuery()
	edges, err := resolver.root.ImageCVEEdgeDataStore.SearchRawEdges(resolver.ctx, query)
	if err != nil || len(edges) == 0 {
		return nil, err
	}
	return timestamp(edges[0].GetFirstImageOccurrence())
}

func (resolver *imageCVEResolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ImageComponents")
	return resolver.root.ImageComponents(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEResolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "ImageCount")
	return resolver.root.ImageCount(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageCVEs, "Images")
	return resolver.root.Images(resolver.imageVulnerabilityScopeContext(ctx), args)
}

func (resolver *imageCVEResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

func (resolver *imageCVEResolver) ExceptionCount(ctx context.Context, args struct{ RequestStatus *[]*string }) (int32, error) {
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

// Following are the functions that return information that is nested in the CVEInfo object
// or are convenience functions to allow time for UI to migrate to new naming schemes

func (resolver *imageCVEResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(resolver.data.GetId())
}

func (resolver *imageCVEResolver) CreatedAt(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetCreatedAt())
}

func (resolver *imageCVEResolver) CVE(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetCve()
}

func (resolver *imageCVEResolver) LastModified(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetLastModified())
}

func (resolver *imageCVEResolver) Link(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetLink()
}

func (resolver *imageCVEResolver) PublishedOn(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetCveBaseInfo().GetPublishedOn())
}

func (resolver *imageCVEResolver) ScoreVersion(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetScoreVersion().String()
}

func (resolver *imageCVEResolver) Summary(_ context.Context) string {
	return resolver.data.GetCveBaseInfo().GetSummary()
}

func (resolver *imageCVEResolver) SuppressActivation(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetSnoozeStart())
}

func (resolver *imageCVEResolver) SuppressExpiry(_ context.Context) (*graphql.Time, error) {
	return timestamp(resolver.data.GetSnoozeExpiry())
}

func (resolver *imageCVEResolver) Suppressed(_ context.Context) bool {
	return resolver.data.GetSnoozed()
}
