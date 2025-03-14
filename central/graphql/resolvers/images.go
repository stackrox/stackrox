package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageWatchStatuses []string

	unknownImageWatchStatus    = registerImageWatchStatus("UNKNOWN")
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
			"deploymentCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"imageComponentCount(query: String): Int!",
			"imageComponents(query: String, pagination: Pagination): [ImageComponent!]!",
			"imageCVECountBySeverity(query: String): ResourceCountByCVESeverity!",
			"imageVulnerabilityCount(query: String): Int!",
			"imageVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerability]!",
			"plottedImageVulnerabilities(query: String): PlottedImageVulnerabilities!",
			"scan: ImageScan",
			"topImageVulnerability(query: String): ImageVulnerability",
			"unusedVarSink(query: String): Int",
			"watchStatus: ImageWatchStatus!",

			// Image scan-related fields
			"dataSource: DataSource",
			"scanNotes: [ImageScan_Note!]!",
			"operatingSystem: String!",
			"scanTime: Time",
			"scannerVersion: String!",
		}),
		schema.AddQuery("image(id: ID!): Image"),
		schema.AddQuery("fullImage(id: ID!): Image"),
		schema.AddQuery("images(query: String, pagination: Pagination): [Image!]!"),
		schema.AddQuery("imageCount(query: String): Int!"),
		schema.AddEnumType("ImageWatchStatus", imageWatchStatuses),
	)
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
	images, err := imageLoader.FromQuery(ctx, q)
	return resolver.wrapImagesWithContext(ctx, images, err)
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
	return resolver.wrapImageWithContext(ctx, image, image != nil, err)
}

// FullImage returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) FullImage(ctx context.Context, args struct{ ID graphql.ID }) (*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "FullImage")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FullImageWithID(ctx, string(args.ID))
	return resolver.wrapImageWithContext(ctx, image, image != nil, err)
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Deployments")
	return resolver.root.Deployments(resolver.withImageScopeContext(ctx), args)
}

// DeploymentCount returns the number of deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.withImageScopeContext(ctx), args)
}

// TopImageVulnerability returns the image vulnerability with the top CVSS score.
func (resolver *imageResolver) TopImageVulnerability(ctx context.Context, args RawQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "TopImageVulnerability")
	return resolver.root.TopImageVulnerability(resolver.withImageScopeContext(ctx), args)
}

// ImageVulnerabilities returns, as ImageVulnerabilityResolver, the vulnerabilities for the image
func (resolver *imageResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilities")
	// TODO(ROX-28320): Data here needs to be grouped by CVE not the ID as is done in the Image Vulnerabilities resolver.
	log.Infof("SHREWS -- image.ImageVulnerabilities -- context %v, args %v", ctx, args.String())
	if features.FlattenCVEData.Enabled() {
		// Grab distinct CVEs
		//query, err := args.AsV1QueryOrEmpty()
		//if err != nil {
		//	return nil, err
		//}
		//cveListish, err := resolver.root.ImageCVEView.GetCVE(resolver.withImageScopeContext(ctx), query)
		//if err != nil {
		//	return nil, err
		//}
		//for _, cve := range cveListish {
		//	log.Infof("SHREWS -- CVE: %s", cve.GetCVE())
		//	log.Infof("SHREWS -- CVE IDs: %v", cve.GetCVEIDs())
		//}
		return resolver.root.ImageFlatVulnerabilities(resolver.withImageScopeContext(ctx), args)
	}
	return resolver.root.ImageVulnerabilities(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageResolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilityCount")
	if features.FlattenCVEData.Enabled() {
		return resolver.root.ImageFlatVulnerabilityCount(resolver.withImageScopeContext(ctx), args)
	}
	return resolver.root.ImageVulnerabilityCount(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageResolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilityCounter")
	// In this function, the resolver is obtained by a call to either Image or Images from the root resolver,
	// applying scoped access control to avoid exposing images the requester should not be aware of.
	// The image ID is added to the context (by withImageScopeContext) to restrict the vulnerability search
	// to the CVEs linked to the image.
	// If no context elevation is done, then scoped access control is applied again on top of the image ID filtering
	// leading to additional table joins in DB and poor performance.
	// The context is elevated to bypass the scoped access control and improve the performance,
	// considering the fact that the image ID was obtained by applying scoped access control rules.
	return resolver.root.ImageVulnerabilityCounter(resolver.withElevatedImageScopeContext(ctx), args)
}

func (resolver *imageResolver) ImageCVECountBySeverity(ctx context.Context, q RawQuery) (*resourceCountBySeverityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageCVECountBySeverity")

	if err := readImages(ctx); err != nil {
		return nil, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	val, err := resolver.root.ImageCVEView.CountBySeverity(resolver.withImageScopeContext(ctx), query)
	return resolver.root.wrapResourceCountByCVESeverityWithContext(ctx, val, err)
}

func (resolver *imageResolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageComponents")
	log.Infof("SHREWS -- image.ImageComponents -- context %v, args %v", ctx, args.String())
	return resolver.root.ImageComponents(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageResolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageResolver) withImageScopeContext(ctx context.Context) context.Context {
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
		Level: v1.SearchCategory_IMAGES,
		ID:    resolver.data.GetId(),
	})
}

func (resolver *imageResolver) withElevatedImageScopeContext(ctx context.Context) context.Context {
	return sac.WithGlobalAccessScopeChecker(
		resolver.withImageScopeContext(ctx),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		),
	)
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

// PlottedImageVulnerabilities returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
func (resolver *imageResolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "PlottedImageVulnerabilities")
	return resolver.root.PlottedImageVulnerabilities(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageResolver) Scan(ctx context.Context) (*imageScanResolver, error) {
	resolver.ensureData(ctx)

	// If scan is pulled, it is most likely to fetch all components and vulns contained in image.
	// Therefore, load the image again with full scan.
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	image, err := imageLoader.FullImageWithID(ctx, resolver.data.GetId())
	if err != nil {
		return nil, err
	}
	scan := image.GetScan()

	res, err := resolver.root.wrapImageScan(scan, true, nil)
	if err != nil || res == nil {
		return nil, err
	}
	res.ctx = resolver.withImageScopeContext(ctx)
	return res, nil
}

func (resolver *imageResolver) WatchStatus(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "WatchStatus")
	if err := readAuth(resources.WatchedImage)(ctx); err != nil {
		if errors.Is(err, errox.NotAuthorized) {
			return unknownImageWatchStatus, nil
		}
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

func (resolver *imageResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

//// Image scan-related fields pulled as direct sub-resolvers of image.

func (resolver *imageResolver) DataSource(_ context.Context) (*dataSourceResolver, error) {
	value := resolver.data.GetScan().GetDataSource()
	return resolver.root.wrapDataSource(value, true, nil)
}

func (resolver *imageResolver) ScanNotes(_ context.Context) []string {
	value := resolver.data.GetScan().GetNotes()
	return stringSlice(value)
}

func (resolver *imageResolver) OperatingSystem(_ context.Context) string {
	value := resolver.data.GetScan().GetOperatingSystem()
	return value
}

func (resolver *imageResolver) ScanTime(_ context.Context) (*graphql.Time, error) {
	value := resolver.data.GetScan().GetScanTime()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageResolver) ScannerVersion(_ context.Context) string {
	value := resolver.data.GetScan().GetScannerVersion()
	return value
}
