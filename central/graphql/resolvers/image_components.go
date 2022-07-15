package resolvers

import (
	"context"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	acConverter "github.com/stackrox/rox/central/activecomponent/converter"
	"github.com/stackrox/rox/central/graphql/resolvers/deploymentctx"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
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
		schema.AddExtraResolvers("ImageComponent", []string{
			"activeState(query: String): ActiveState",
			"deploymentCount(query: String, scopeQuery: String): Int!",
			"deployments(query: String, scopeQuery: String, pagination: Pagination): [Deployment!]!",
			"fixedIn: String! @deprecated(reason: \"use 'fixedBy'\")",
			"imageCount(query: String, scopeQuery: String): Int!",
			"images(query: String, scopeQuery: String, pagination: Pagination): [Image!]!",
			"imageVulnerabilityCount(query: String, scopeQuery: String): Int!",
			"imageVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability]!",
			"lastScanned: Time",
			"location(query: String): String!",
			"plottedImageVulnerabilities(query: String): PlottedImageVulnerabilities!",
			"topImageVulnerability: ImageVulnerability",
			"unusedVarSink(query: String): Int",
		}),
		schema.AddQuery("imageComponent(id: ID): ImageComponent"),
		schema.AddQuery("imageComponents(query: String, scopeQuery: String, pagination: Pagination): [ImageComponent!]!"),
		schema.AddQuery("imageComponentCount(query: String): Int!"),

		// TODO
		schema.AddExtraResolver("ImageScan", `components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("ImageScan", `componentCount(query: String): Int!`),
	)
}

// ImageComponentResolver represents a generic resolver of image component fields.
// Values may come from either an embedded component context, or a top level component context.
// NOTE: This list is and should remain alphabetically ordered
type ImageComponentResolver interface {
	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	DeploymentCount(ctx context.Context, args RawQuery) (int32, error)
	Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error)
	FixedIn(ctx context.Context) string
	FixedBy(ctx context.Context) string
	ID(ctx context.Context) graphql.ID
	ImageCount(ctx context.Context, args RawQuery) (int32, error)
	Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error)
	ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error)
	ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error)
	ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	LayerIndex() *int32
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
}

// ImageComponent returns an image component based on an input id (name:version)
func (resolver *Resolver) ImageComponent(ctx context.Context, args IDQuery) (ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")
	if !features.PostgresDatastore.Enabled() {
		return resolver.imageComponentV2(ctx, args)
	}

	// check permissions
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// get loader
	loader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return nil, err
	}

	ret, err := loader.FromID(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	}
	return resolver.wrapImageComponent(ret, true, err)
}

// ImageComponents returns image components that match the input query.
func (resolver *Resolver) ImageComponents(ctx context.Context, q PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	if !features.PostgresDatastore.Enabled() {
		query := queryWithImageIDRegexFilter(q.String())

		return resolver.imageComponentsV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
	}

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
	loader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return nil, err
	}

	// get values
	query = tryUnsuppressedQuery(query)
	componentResolvers, err := resolver.wrapImageComponents(loader.FromQuery(ctx, query))
	if err != nil {
		return nil, err
	}

	// cast as return type
	ret := make([]ImageComponentResolver, 0, len(componentResolvers))
	for _, res := range componentResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, nil
}

// ImageComponentCount returns count of image components that match the input query
func (resolver *Resolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponentCount")
	if !features.PostgresDatastore.Enabled() {
		query := queryWithImageIDRegexFilter(args.String())

		return resolver.componentCountV2(ctx, RawQuery{Query: &query})
	}

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
	loader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return 0, err
	}
	query = tryUnsuppressedQuery(query)

	return loader.CountFromQuery(ctx, query)
}

/*
Utility Functions
*/

func (resolver *imageComponentResolver) withImageComponentScope(ctx context.Context) context.Context {
	return scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    resolver.data.GetId(),
	})
}

func (resolver *imageComponentResolver) componentQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ComponentID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *imageComponentResolver) componentRawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.ComponentID, resolver.data.GetId()).Query()
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
	var deploymentID string
	for _, ctx := range contexts {
		deploymentID = deploymentctx.FromContext(ctx)
		if deploymentID != "" {
			return deploymentID
		}
	}
	if scopeQuery != nil {
		deploymentID = getDeploymentIDFromQuery(scopeQuery)
	}
	return deploymentID
}

func queryWithImageIDRegexFilter(q string) string {
	return search.AddRawQueriesAsConjunction(q,
		search.NewQueryBuilder().AddRegexes(search.ImageSHA, ".+").Query())
}

/*
Sub Resolver Functions
*/

func (resolver *imageComponentResolver) ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ActiveState")
	scopeQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	deploymentID := getDeploymentScope(scopeQuery, resolver.ctx)
	if deploymentID == "" {
		return nil, nil
	}

	if resolver.data.GetSource() != storage.SourceType_OS {
		return &activeStateResolver{
			root:  resolver.root,
			state: Undetermined,
		}, nil
	}
	acID := acConverter.ComposeID(deploymentID, resolver.data.GetId())

	var found bool
	imageID := getImageIDFromQuery(scopeQuery)
	if imageID == "" {
		found, err = resolver.root.ActiveComponent.Exists(ctx, acID)
		if err != nil {
			return nil, err
		}
	} else {
		query := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageID).ProtoQuery()
		results, err := resolver.root.ActiveComponent.Search(ctx, query)
		if err != nil {
			return nil, err
		}
		found = search.ResultsToIDSet(results).Contains(acID)
	}
	if !found {
		return &activeStateResolver{
			root:  resolver.root,
			state: Inactive,
		}, nil
	}

	return &activeStateResolver{
		root:               resolver.root,
		state:              Active,
		activeComponentIDs: []string{acID},
		imageScope:         imageID,
	}, nil
}

func (resolver *imageComponentResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "Deployments")
	return resolver.root.Deployments(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageCount")
	return resolver.root.ImageCount(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "Images")
	return resolver.root.Images(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilityCount")
	return resolver.root.ImageVulnerabilityCount(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilityCounter")
	return resolver.root.ImageVulnerabilityCounter(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilities")
	return resolver.root.ImageVulnerabilities(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "LastScanned")
	imageLoader, err := loaders.GetImageLoader(ctx)
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

	images, err := imageLoader.FromQuery(resolver.withImageComponentScope(ctx), q)
	if err != nil || len(images) == 0 {
		return nil, err
	} else if len(images) > 1 {
		return nil, errors.New("multiple images matched for last scanned image component query")
	}

	return timestamp(images[0].GetScan().GetScanTime())
}

// Location returns the location of the component.
func (resolver *imageComponentResolver) Location(ctx context.Context, args RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "Location")
	var imageID string
	scope, hasScope := scoped.GetScope(ctx)
	if hasScope && scope.Level == v1.SearchCategory_IMAGES {
		imageID = scope.ID
	} else {
		var err error
		imageID, err = getImageIDFromIfImageShaQuery(ctx, resolver.root, args)
		if err != nil {
			return "", errors.Wrap(err, "could not determine component location")
		}
	}

	if imageID == "" {
		return "", nil
	}

	if !features.PostgresDatastore.Enabled() {
		edgeID := edges.EdgeID{ParentID: imageID, ChildID: resolver.data.GetId()}.ToString()
		edge, found, err := resolver.root.ImageComponentEdgeDataStore.Get(ctx, edgeID)
		if err != nil || !found {
			return "", err
		}
		return edge.GetLocation(), nil
	}
	query := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageID).AddExactMatches(search.ComponentID, resolver.data.GetId()).ProtoQuery()
	edges, err := resolver.root.ImageComponentEdgeDataStore.SearchRawEdges(ctx, query)
	if err != nil || len(edges) == 0 {
		return "", err
	}
	return edges[0].GetLocation(), nil
}

// PlottedImageVulnerabilities returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
func (resolver *imageComponentResolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "PlottedImageVulnerabilities")
	return resolver.root.PlottedImageVulnerabilities(resolver.withImageComponentScope(ctx), args)
}

func (resolver *imageComponentResolver) TopImageVulnerability(ctx context.Context) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "TopImageVulnerability")
	return resolver.root.TopImageVulnerability(resolver.withImageComponentScope(ctx), RawQuery{})
}

func (resolver *imageComponentResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(resolver.data.GetId())
}

func (resolver *imageComponentResolver) LayerIndex() *int32 {
	return nil
}

func (resolver *imageComponentResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

// Following are deprecated functions that are retained to allow UI time to migrate away from them

func (resolver *imageComponentResolver) FixedIn(_ context.Context) string {
	return resolver.data.GetFixedBy()
}
