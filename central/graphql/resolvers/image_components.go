package resolvers

import (
	"context"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	cveConverter "github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/graphql/resolvers/deploymentctx"
	"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/image/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(schema.AddType("ImageComponentV2", []string{
		"architecture: String!",
		"fixedBy: String!",
		"id: ID!",
		"imageId: String!",
		"name: String!",
		"operatingSystem: String!",
		"priority: Int!",
		"riskScore: Float!",
		"source: SourceType!",
		"version: String!",
	}))

	utils.Must(
		schema.AddQuery("imageComponent(id: ID): ImageComponent"),
		schema.AddQuery("imageComponents(query: String, scopeQuery: String, pagination: Pagination): [ImageComponent!]!"),
		schema.AddQuery("imageComponentCount(query: String): Int!"),
	)
}

// ImageComponentResolver represents a generic resolver of image component fields.
// Values may come from either an embedded component context, or a top level component context.
// NOTE: This list is and should remain alphabetically ordered
type ImageComponentResolver interface {
	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	FixedBy(ctx context.Context) string
	Id(ctx context.Context) graphql.ID
	ImageCount(ctx context.Context, args RawQuery) (int32, error)
	Images(ctx context.Context, args PaginatedQuery) ([]ImageResolver, error)
	ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error)
	ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)
	ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	LayerIndex() (*int32, error)
	License(ctx context.Context) (*licenseResolver, error)
	Location(ctx context.Context, args RawQuery) (string, error)
	Name(ctx context.Context) string
	OperatingSystem(ctx context.Context) string
	PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error)
	Priority(ctx context.Context) int32
	RiskScore(ctx context.Context) float64
	Source(ctx context.Context) string
	TopImageVulnerability(ctx context.Context) (ImageVulnerabilityResolver, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Version(ctx context.Context) string

	// deprecated functions

	FixedIn(ctx context.Context) string
}

// ImageComponent returns an image component based on an input id (name:version)
func (resolver *Resolver) ImageComponent(ctx context.Context, args IDQuery) (ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// get loader
	loader, err := loaders.GetComponentV2Loader(ctx)
	if err != nil {
		return nil, err
	}

	ret, err := loader.FromID(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	}

	// With flattened model, there can be multiple component IDs for a given component name, version and OS.
	// But the component list page on VM 1.0 groups results by name, version and OS. So on component single page,
	// we should also show component details (like top CVSS, priority etc) and
	// related entities (like images, CVEs and deployments) grouped by component name + version + OS.
	query := search.NewQueryBuilder().
		AddExactMatches(search.Component, ret.GetName()).
		AddExactMatches(search.ComponentVersion, ret.GetVersion()).
		AddExactMatches(search.OperatingSystem, ret.GetOperatingSystem()).
		ProtoQuery()
	componentFlatData, err := resolver.ImageComponentFlatView.Get(ctx, query)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-28808): The ticket referenced is the reason we can get here.  FromID will find
	// a component ID excluded by context but the FlatView will not. We should honor the context. This
	// will be cleaned up with 28808.
	if len(componentFlatData) != 1 {
		return nil, errors.New("unable to find component")
	}

	return resolver.wrapImageComponentV2FlatWithContext(ctx, ret, componentFlatData[0], true, err)
}

// ImageComponents returns image components that match the input query.
func (resolver *Resolver) ImageComponents(ctx context.Context, q PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// cast query
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	// Get the flattened data
	componentFlatData, err := resolver.ImageComponentFlatView.Get(ctx, query)
	if err != nil {
		return nil, err
	}

	componentIDs := make([]string, 0, len(componentFlatData))
	for _, componentFlat := range componentFlatData {
		componentIDs = append(componentIDs, componentFlat.GetComponentIDs()...)
	}

	// get loader
	loader, err := loaders.GetComponentV2Loader(ctx)
	if err != nil {
		return nil, err
	}

	// Get the Components themselves.  This will be denormalized.  So use the IDs to get them, but use
	// the data returned from Component Flat View to keep order and set just 1 instance of a Component
	componentQuery := search.NewQueryBuilder().AddExactMatches(search.ComponentID, componentIDs...).ProtoQuery()
	componentQuery.Pagination = &v1.QueryPagination{
		SortOptions: query.GetPagination().GetSortOptions(),
	}
	comps, err := loader.FromQuery(ctx, componentQuery)

	// Stash a single instance of a Component to aid in normalizing
	foundComponent := make(map[normalizedImageComponent]*storage.ImageComponentV2)
	for _, comp := range comps {
		normalized := normalizedImageComponent{
			name:    comp.GetName(),
			version: comp.GetVersion(),
			os:      comp.GetOperatingSystem(),
		}
		if _, ok := foundComponent[normalized]; !ok {
			foundComponent[normalized] = comp
		}
	}

	// Normalize the Components based on the flat view to keep them in the correct paging and sort order
	normalizedComponents := make([]*storage.ImageComponentV2, 0, len(componentFlatData))
	for _, componentFlat := range componentFlatData {
		normalized := normalizedImageComponent{
			name:    componentFlat.GetComponent(),
			version: componentFlat.GetVersion(),
			os:      componentFlat.GetOperatingSystem(),
		}
		normalizedComponents = append(normalizedComponents, foundComponent[normalized])
	}

	componentResolvers, err := resolver.wrapImageComponentV2sFlatWithContext(ctx, normalizedComponents, componentFlatData, err)
	if err != nil {
		return nil, err
	}

	// cast as return type
	ret := make([]ImageComponentResolver, 0, len(componentResolvers))
	for _, res := range componentResolvers {
		ret = append(ret, res)
	}
	return ret, nil
}

// ImageComponentCount returns count of image components that match the input query
func (resolver *Resolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponentCount")
	// check permissions
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	// cast query
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	if features.FlattenCVEData.Enabled() {
		componentCount, err := resolver.ImageComponentFlatView.Count(ctx, query)
		return int32(componentCount), err
	}
	// get loader
	loader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return 0, err
	}

	return loader.CountFromQuery(ctx, query)
}

/*
Utility Functions
*/

func (resolver *imageComponentResolver) imageComponentScopeContext(ctx context.Context) context.Context {
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
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		IDs:   []string{resolver.data.GetId()},
	})
}

func (resolver *imageComponentResolver) componentQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ComponentID, resolver.data.GetId()).ProtoQuery()
}

func getDeploymentIDFromQuery(q *v1.Query) string {
	if q == nil {
		return ""
	}
	var deploymentID string
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if strings.EqualFold(matchFieldQuery.MatchFieldQuery.GetField(), search.DeploymentID.String()) {
			deploymentID = matchFieldQuery.MatchFieldQuery.Value
			deploymentID = strings.TrimRight(deploymentID, `"`)
			deploymentID = strings.TrimLeft(deploymentID, `"`)
		}
	})
	return deploymentID
}

func getDeploymentScope(scopeQuery *v1.Query, contexts ...context.Context) string {
	for _, ctx := range contexts {
		if scope, ok := scoped.GetScope(ctx); ok && scope.Level == v1.SearchCategory_DEPLOYMENTS {
			if len(scope.IDs) != 1 {
				return ""
			}
			return scope.IDs[0]
		} else if deploymentID := deploymentctx.FromContext(ctx); deploymentID != "" {
			return deploymentID
		}
	}
	if scopeQuery != nil {
		return getDeploymentIDFromQuery(scopeQuery)
	}
	return ""
}

func getImageIDFromScope(contexts ...context.Context) string {
	var scope scoped.Scope
	var hasScope bool
	for _, ctx := range contexts {
		searchCategory := v1.SearchCategory_IMAGES
		if features.FlattenImageData.Enabled() {
			searchCategory = v1.SearchCategory_IMAGES_V2
		}
		if scope, hasScope = scoped.GetScopeAtLevel(ctx, searchCategory); hasScope {
			if len(scope.IDs) != 1 {
				return ""
			}
			return scope.IDs[0]
		}
	}
	return ""
}

func getImageCVEV2Resolvers(ctx context.Context, root *Resolver, imageID string, component *storage.EmbeddedImageScanComponent, query *v1.Query) ([]ImageVulnerabilityResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.VulnerabilityOptionsMap)
	predicate, err := vulnPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	componentID, err := scancomponent.ComponentIDV2(component, imageID)
	if err != nil {
		return nil, err
	}
	resolvers := make([]ImageVulnerabilityResolver, 0, len(component.GetVulns()))
	for _, vuln := range component.GetVulns() {
		if !predicate.Matches(vuln) {
			continue
		}
		converted, err := cveConverter.EmbeddedVulnerabilityToImageCVEV2(imageID, componentID, vuln)
		if err != nil {
			return nil, err
		}

		resolver, err := root.wrapImageCVEV2(converted, true, nil)
		if err != nil {
			return nil, err
		}
		resolver.ctx = embeddedobjs.VulnContext(ctx, vuln)

		resolvers = append(resolvers, resolver)
	}

	return paginate(query.GetPagination(), resolvers, nil)
}

/*
Sub Resolver Functions
*/

func (resolver *imageComponentV2Resolver) ActiveState(_ context.Context, _ RawQuery) (*activeStateResolver, error) {
	// No longer supported as scanner V4 does not support it.
	return &activeStateResolver{}, nil
}

func (resolver *imageComponentV2Resolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "Deployments")
	return resolver.root.Deployments(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageCount")
	return resolver.root.ImageCount(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) Images(ctx context.Context, args PaginatedQuery) ([]ImageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "Images")
	return resolver.root.Images(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilityCount")
	return resolver.root.ImageVulnerabilityCount(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilityCounter")
	return resolver.root.ImageVulnerabilityCounter(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilities")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	embeddedComponent := embeddedobjs.ComponentFromContext(resolver.ctx)
	if embeddedComponent == nil {
		return resolver.root.ImageVulnerabilities(resolver.imageComponentScopeContext(ctx), args)
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return getImageCVEV2Resolvers(resolver.ctx, resolver.root, resolver.ImageId(resolver.ctx), embeddedComponent, query)
}

func (resolver *imageComponentV2Resolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "LastScanned")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	if scanTime := embeddedobjs.LastScannedFromContext(resolver.ctx); scanTime != nil {
		return &graphql.Time{Time: *scanTime}, nil
	}

	if features.FlattenImageData.Enabled() {
		imageLoader, err := loaders.GetImageV2Loader(resolver.ctx)
		if err != nil {
			return nil, err
		}
		q := search.NewQueryBuilder().AddExactMatches(search.ImageID, resolver.data.GetImageIdV2()).ProtoQuery()

		images, err := imageLoader.FromQuery(resolver.ctx, q)
		if err != nil || len(images) == 0 {
			return nil, err
		} else if len(images) > 1 {
			return nil, errors.New("multiple images matched for last scanned image component query")
		}

		return protocompat.ConvertTimestampToGraphqlTimeOrError(images[0].GetScan().GetScanTime())
	}
	imageLoader, err := loaders.GetImageLoader(resolver.ctx)
	if err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, resolver.data.GetImageId()).ProtoQuery()

	images, err := imageLoader.FromQuery(resolver.ctx, q)
	if err != nil || len(images) == 0 {
		return nil, err
	} else if len(images) > 1 {
		return nil, errors.New("multiple images matched for last scanned image component query")
	}

	return protocompat.ConvertTimestampToGraphqlTimeOrError(images[0].GetScan().GetScanTime())
}

// PlottedImageVulnerabilities returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
func (resolver *imageComponentV2Resolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "PlottedImageVulnerabilities")
	return resolver.root.PlottedImageVulnerabilities(resolver.imageComponentScopeContext(ctx), args)
}

func (resolver *imageComponentV2Resolver) TopImageVulnerability(ctx context.Context) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "TopImageVulnerability")
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}

	// Short path. Full image is embedded when image scan resolver is called.
	if embeddedComponent := embeddedobjs.ComponentFromContext(resolver.ctx); embeddedComponent != nil {
		var topVuln *storage.EmbeddedVulnerability
		for _, vuln := range embeddedComponent.GetVulns() {
			if topVuln == nil || vuln.GetCvss() > topVuln.GetCvss() {
				topVuln = vuln
			}
		}
		if topVuln == nil {
			return nil, nil
		}
		componentID, err := scancomponent.ComponentIDV2(embeddedComponent, resolver.ImageId(resolver.ctx))
		if err != nil {
			return nil, err
		}

		convertedTopVuln, err := cveConverter.EmbeddedVulnerabilityToImageCVEV2(resolver.ImageId(resolver.ctx), componentID, topVuln)
		if err != nil {
			return nil, err
		}
		return resolver.root.wrapImageCVEV2WithContext(resolver.ctx, convertedTopVuln, true, nil)
	}

	return resolver.root.TopImageVulnerability(resolver.imageComponentScopeContext(ctx), RawQuery{})
}

func (resolver *imageComponentV2Resolver) LayerIndex() (*int32, error) {
	w, ok := resolver.data.GetHasLayerIndex().(*storage.ImageComponentV2_LayerIndex)
	if !ok {
		return nil, nil
	}
	v := w.LayerIndex
	return &v, nil
}

// Location returns the location of the component.
//
//	TODO(ROX-28123): Once the old code is removed, the parameters can be removed.
func (resolver *imageComponentV2Resolver) Location(_ context.Context, _ RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "Location")

	return resolver.data.GetLocation(), nil
}

func (resolver *imageComponentV2Resolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

// Following are deprecated functions that are retained to allow UI time to migrate away from them

func (resolver *imageComponentV2Resolver) FixedIn(_ context.Context) string {
	return resolver.data.GetFixedBy()
}

func (resolver *imageComponentV2Resolver) License(ctx context.Context) (*licenseResolver, error) {
	return nil, nil
}

/*
Utility Functions
*/

func (resolver *imageComponentV2Resolver) imageComponentScopeContext(ctx context.Context) context.Context {
	if ctx == nil {
		err := utils.ShouldErr(errors.New("argument 'ctx' is nil"))
		if err != nil {
			log.Error(err)
		}
	}
	if resolver.ctx == nil {
		resolver.ctx = ctx
	}
	if features.FlattenCVEData.Enabled() && resolver.flatData != nil && len(resolver.flatData.GetComponentIDs()) > 0 {
		return scoped.Context(resolver.ctx, scoped.Scope{
			Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
			IDs:   resolver.flatData.GetComponentIDs(),
		})
	}
	return scoped.Context(resolver.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
		IDs:   []string{resolver.data.GetId()},
	})
}
