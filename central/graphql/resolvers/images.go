package resolvers

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/distroctx"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageWatchStatuses []string

	notWatchedImageWatchStatus = registerImageWatchStatus("NOT_WATCHED")
	watchedImageStatus         = registerImageWatchStatus("WATCHED")
)

func registerImageWatchStatus(s string) string {
	imageWatchStatuses = append(imageWatchStatuses, s)
	return s
}

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddExtraResolvers("Image", []string{
			"componentCount(query: String): Int!",
			"components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!",
			"deploymentCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"imageVulnerabilityCount(query: String): Int!",
			"imageVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability]!",
			"plottedVulns(query: String): PlottedVulnerabilities!",
			"topImageVulnerability(query: String): ImageVulnerability",
			"unusedVarSink(query: String): Int",
			"watchStatus: ImageWatchStatus!",
		}),
		// deprecated fields
		schema.AddExtraResolvers("Image", []string{
			"topVuln(query: String): EmbeddedVulnerability " +
				"@deprecated(reason: \"use 'topImageVulnerability'\")",
			"vulnCount(query: String): Int! " +
				"@deprecated(reason: \"use 'imageVulnerabilityCount'\")",
			"vulnCounter(query: String): VulnerabilityCounter! " +
				"@deprecated(reason: \"use 'imageVulnerabilityCounter'\")",
			"vulns(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedVulnerability]! " +
				"@deprecated(reason: \"use 'imageVulnerabilities'\")",
		}),
		schema.AddInput("sortBy", []string{
			"sortByFixable: Boolean!",
		}),
		schema.AddQuery("images(query: String, pagination: Pagination): [Image!]!"),
		schema.AddQuery("sortedImages(query: String, pagination: Pagination, sortBy: sortBy): [Image!]!"),
		schema.AddQuery("imageCount(query: String): Int!"),
		schema.AddQuery("image(id: ID!): Image"),
		schema.AddExtraResolver("EmbeddedImageScanComponent", "layerIndex: Int"),
		schema.AddEnumType("ImageWatchStatus", imageWatchStatuses),
	)
}

// SortBy represents sort options to sort images by
type SortBy struct {
	SortByFixable bool
}

// SortedImages returns a list of images sorted by CVE severity distribution
func (resolver *Resolver) SortedImages(ctx context.Context, args struct {
	Query      *string
	Pagination *inputtypes.Pagination
	SortBy     *SortBy
}) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Images")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	var sortByFixable bool
	if args.SortBy == nil {
		sortByFixable = false
	} else {
		sortByFixable = args.SortBy.SortByFixable
	}
	var q *v1.Query
	if args.Query == nil {
		q := search.EmptyQuery()
		paginated.FillPagination(q, args.Pagination.AsV1Pagination(), math.MaxInt32)
	} else {
		q, err := search.ParseQuery(*args.Query, search.MatchAllIfEmpty())
		if err != nil {
			return nil, err
		}
		paginated.FillPagination(q, args.Pagination.AsV1Pagination(), math.MaxInt32)

	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	images, err := resolver.wrapImages(imageLoader.FromQuery(ctx, q))
	if err != nil {
		return nil, err
	}

	sort.Slice(images, func(i, j int) bool {
		vulnCount1, err := images[i].ImageVulnerabilityCounter(ctx, RawQuery{args.Query, nil})
		if err != nil {
			return false
		}
		vulnCount2, err := images[j].ImageVulnerabilityCounter(ctx, RawQuery{args.Query, nil})
		if err != nil {
			return false
		}
		return compareVulnDistributionBySeverity(vulnCount1, vulnCount2, sortByFixable)
	})
	return images, nil
}

func compareVulnDistributionBySeverity(vulnCount1 *VulnerabilityCounterResolver,
	vulnCount2 *VulnerabilityCounterResolver, sortByFixable bool) bool {
	if sortByFixable {
		return vulnCount1.critical.fixable > vulnCount2.critical.fixable ||
			(vulnCount1.critical.fixable == vulnCount2.critical.fixable && vulnCount1.important.fixable > vulnCount2.important.fixable) ||
			(vulnCount1.important.fixable == vulnCount2.important.fixable && vulnCount1.moderate.fixable > vulnCount2.moderate.fixable) ||
			(vulnCount1.moderate.fixable == vulnCount2.moderate.fixable && vulnCount1.low.fixable > vulnCount2.low.fixable)
	}
	return vulnCount1.critical.total > vulnCount2.critical.total ||
		(vulnCount1.critical.total == vulnCount2.critical.total && vulnCount1.important.total > vulnCount2.important.total) ||
		(vulnCount1.important.total == vulnCount2.important.total && vulnCount1.moderate.total > vulnCount2.moderate.total) ||
		(vulnCount1.moderate.total == vulnCount2.moderate.total && vulnCount1.low.total > vulnCount2.low.total)
}

// Images returns GraphQL resolvers for all images
func (resolver *Resolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Images")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.wrapImages(imageLoader.FromQuery(ctx, q))
}

// ImageCount returns count of all images across deployments
func (resolver *Resolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageCount")
	if err := readImages(ctx); err != nil {
		return 0, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return 0, err
	}
	return imageLoader.CountFromQuery(ctx, q)
}

// Image returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) Image(ctx context.Context, args struct{ ID graphql.ID }) (*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Image")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FromID(ctx, string(args.ID))
	return resolver.wrapImage(image, image != nil, err)
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())

	return resolver.root.Deployments(ctx, PaginatedQuery{Pagination: args.Pagination, Query: &query})
}

// DeploymentCount returns the number of deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())

	return resolver.root.DeploymentCount(ctx, RawQuery{Query: &query})
}

// TopImageVulnerability returns the image vulnerability with the top CVSS score.
func (resolver *imageResolver) TopImageVulnerability(ctx context.Context, args RawQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "TopImageVulnerability")
	if !features.PostgresDatastore.Enabled() {
		return resolver.topVulnV2(ctx, args)
	}
	// TODO postgres support
	return nil, errors.New("Sub-resolver TopVulnerability in image does not support postgres")
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (resolver *imageResolver) TopVuln(ctx context.Context, args RawQuery) (VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "TopVuln")
	return resolver.topVulnV2(ctx, args)
}

func (resolver *imageResolver) topVulnV2(ctx context.Context, args RawQuery) (VulnerabilityResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	if resolver.data.GetSetTopCvss() == nil {
		return nil, nil
	}

	if args.IsEmpty() {
		var max *storage.EmbeddedVulnerability
		for _, c := range resolver.data.GetScan().GetComponents() {
			for _, v := range c.GetVulns() {
				if max == nil {
					max = v
					continue
				}
				if v.GetCvss() > max.GetCvss() || (v.GetCvss() == max.GetCvss() && v.GetCve() > max.GetCve()) {
					max = v
				}
			}
		}
		return resolver.root.wrapEmbeddedVulnerability(max, nil)
	}

	query = search.ConjunctionQuery(query, resolver.getImageQuery())
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

	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	vulns, err := vulnLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	} else if len(vulns) == 0 {
		return nil, err
	} else if len(vulns) > 1 {
		return nil, errors.New("multiple vulnerabilities matched for top image vulnerability")
	}
	return &cVEResolver{root: resolver.root, data: vulns[0]}, nil
}

func (resolver *imageResolver) vulnQueryScoping(ctx context.Context) context.Context {
	ctx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	})
	ctx = distroctx.Context(ctx, resolver.data.GetScan().GetOperatingSystem())

	return ctx
}

// ImageVulnerabilities returns, as ImageVulnerabilityResolver, the vulnerabilities for the image
func (resolver *imageResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilities")

	ctx = resolver.vulnQueryScoping(ctx)

	return resolver.root.ImageVulnerabilities(ctx, args)
}

func (resolver *imageResolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilityCount")

	ctx = resolver.vulnQueryScoping(ctx)

	return resolver.root.ImageVulnerabilityCount(ctx, args)
}

func (resolver *imageResolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilityCounter")

	ctx = resolver.vulnQueryScoping(ctx)

	return resolver.root.ImageVulnerabilityCounter(ctx, args)
}

// Vulns returns all of the vulnerabilities in the image.
func (resolver *imageResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Vulns")

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())

	ctx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	})
	ctx = distroctx.Context(ctx, resolver.data.GetScan().GetOperatingSystem())
	return resolver.root.Vulnerabilities(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// VulnCount returns the number of vulnerabilities the image has.
func (resolver *imageResolver) VulnCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "VulnCount")

	// if the request isn't being filtered down we can use cached data
	if args.IsEmpty() {
		vulnSet := set.NewStringSet()
		for _, c := range resolver.data.GetScan().GetComponents() {
			for _, v := range c.GetVulns() {
				vulnSet.Add(v.GetCve())
			}
		}
		return int32(len(vulnSet)), nil
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())

	ctx = distroctx.Context(ctx, resolver.data.GetScan().GetOperatingSystem())
	return resolver.root.VulnerabilityCount(scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	}), RawQuery{Query: &query})
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (resolver *imageResolver) VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "VulnCounter")

	// if the request isn't being filtered down we can use cached data
	if args.IsEmpty() {
		var vulns []*storage.EmbeddedVulnerability
		vulnSet := set.NewStringSet()
		for _, component := range resolver.data.GetScan().GetComponents() {
			for _, v := range component.GetVulns() {
				if vulnSet.Add(v.GetCve()) {
					vulns = append(vulns, v)
				}
			}
		}
		return mapVulnsToVulnerabilityCounter(vulns), nil
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())
	return resolver.root.VulnCounter(scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	}), RawQuery{Query: &query})
}

// Components returns all of the components in the image.
func (resolver *imageResolver) Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Components")

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())

	ctx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	})
	ctx = distroctx.Context(ctx, resolver.data.GetScan().GetOperatingSystem())
	return resolver.root.Components(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

func (resolver *imageResolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ComponentCount")

	if args.IsEmpty() {
		return int32(len(resolver.data.GetScan().GetComponents())), nil
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())
	return resolver.root.ComponentCount(scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	}), RawQuery{Query: &query})
}

func (resolver *Resolver) getImage(ctx context.Context, id string) *storage.Image {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil
	}
	image, err := imageLoader.FromID(ctx, id)
	if err != nil {
		return nil
	}
	return image
}

func (resolver *imageResolver) getImageRawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.ImageSHA, resolver.data.GetId()).Query()
}

func (resolver *imageResolver) getImageQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ImageSHA, resolver.data.GetId()).ProtoQuery()
}

func (resolver *imageResolver) PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getImageRawQuery())
	return newPlottedVulnerabilitiesResolver(ctx, resolver.root, RawQuery{Query: &query})
}

func (resolver *imageResolver) WatchStatus(ctx context.Context) (string, error) {
	if err := readAuth(resources.WatchedImage)(ctx); err != nil {
		return "", err
	}
	watched, err := resolver.root.WatchedImageDataStore.Exists(ctx, resolver.data.GetName().GetFullName())
	if err != nil {
		return "", err
	}
	if watched {
		return watchedImageStatus, nil
	}
	return notWatchedImageWatchStatus, nil
}

func (resolver *imageResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}
