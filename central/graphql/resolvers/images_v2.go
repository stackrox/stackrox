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
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolvers("ImageV2", []string{
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

			// Image signature-related fields.
			"signatureCount: Int!",
		}),
		schema.AddQuery("imageV2(id: ID!): ImageV2"),
		schema.AddQuery("fullImageV2(id: ID!): ImageV2"),
		schema.AddQuery("imageV2s(query: String, pagination: Pagination): [ImageV2!]!"),
		schema.AddQuery("imageV2Count(query: String): Int!"),
		schema.AddEnumType("ImageV2WatchStatus", imageWatchStatuses),
	)
}

// ImagesV2 returns GraphQL resolvers for all images using the ImageV2 model
func (resolver *Resolver) ImageV2s(ctx context.Context, args PaginatedQuery) ([]ImageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageV2s")
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	imageLoader, err := loaders.GetImageV2Loader(ctx)
	if err != nil {
		return nil, err
	}
	images, err := imageLoader.FromQuery(ctx, q)
	resolvers, err := resolver.wrapImageV2sWithContext(ctx, images, err)
	res := make([]ImageResolver, 0, len(resolvers))
	for _, resolver := range resolvers {
		res = append(res, resolver)
	}
	return res, err
}

// ImageV2Count returns count of all images across deployments using the ImageV2 model
func (resolver *Resolver) ImageV2Count(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageV2Count")
	if err := readImages(ctx); err != nil {
		return 0, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	imageLoader, err := loaders.GetImageV2Loader(ctx)
	if err != nil {
		return 0, err
	}
	return imageLoader.CountFromQuery(ctx, q)
}

// ImageV2 returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) ImageV2(ctx context.Context, args struct{ ID graphql.ID }) (ImageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageV2")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageV2Loader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FromID(ctx, string(args.ID))
	return resolver.wrapImageV2WithContext(ctx, image, image != nil, err)
}

// FullImage returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) FullImageV2(ctx context.Context, args struct{ ID graphql.ID }) (ImageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "FullImageV2")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageV2Loader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FullImageWithID(ctx, string(args.ID))
	return resolver.wrapImageV2WithContext(ctx, image, image != nil, err)
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageV2Resolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Deployments")
	return resolver.root.Deployments(resolver.withImageScopeContext(ctx), args)
}

// DeploymentCount returns the number of deployments which use this image for the identified image, if it exists
func (resolver *imageV2Resolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.withImageScopeContext(ctx), args)
}

// TopImageVulnerability returns the image vulnerability with the top CVSS score.
func (resolver *imageV2Resolver) TopImageVulnerability(ctx context.Context, args RawQuery) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "TopImageVulnerability")
	return resolver.root.TopImageVulnerability(resolver.withImageScopeContext(ctx), args)
}

// ImageVulnerabilities returns, as ImageVulnerabilityResolver, the vulnerabilities for the image
func (resolver *imageV2Resolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilities")
	return resolver.root.ImageVulnerabilities(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageV2Resolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilityCount")
	return resolver.root.ImageVulnerabilityCount(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageV2Resolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
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

func (resolver *imageV2Resolver) ImageCVECountBySeverity(ctx context.Context, q RawQuery) (*resourceCountBySeverityResolver, error) {
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

func (resolver *imageV2Resolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageComponents")
	return resolver.root.ImageComponents(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageV2Resolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageV2Resolver) Names(ctx context.Context) ([]*imageNameResolver, error) {
	resolver.ensureData(ctx)
	value := []*storage.ImageName{resolver.data.GetName()}
	return resolver.root.wrapImageNames(value, nil)
}

func (resolver *imageV2Resolver) withImageScopeContext(ctx context.Context) context.Context {
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
		Level: v1.SearchCategory_IMAGES_V2,
		IDs:   []string{resolver.data.GetId()},
	})
}

func (resolver *imageV2Resolver) withElevatedImageScopeContext(ctx context.Context) context.Context {
	return sac.WithGlobalAccessScopeChecker(
		resolver.withImageScopeContext(ctx),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		),
	)
}

func (resolver *Resolver) getImageV2(ctx context.Context, id string) *storage.ImageV2 {
	imageLoader, err := loaders.GetImageV2Loader(ctx)
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
func (resolver *imageV2Resolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "PlottedImageVulnerabilities")
	return resolver.root.PlottedImageVulnerabilities(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageV2Resolver) Scan(ctx context.Context) (*imageScanResolver, error) {
	resolver.ensureData(ctx)

	// If scan is pulled, it is most likely to fetch all components and vulns contained in image.
	// Therefore, load the image again with full scan.
	imageLoader, err := loaders.GetImageV2Loader(ctx)
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

func (resolver *imageV2Resolver) WatchStatus(ctx context.Context) (string, error) {
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

func (resolver *imageV2Resolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

//// Image scan-related fields pulled as direct sub-resolvers of image.

func (resolver *imageV2Resolver) DataSource(_ context.Context) (*dataSourceResolver, error) {
	value := resolver.data.GetScan().GetDataSource()
	return resolver.root.wrapDataSource(value, true, nil)
}

func (resolver *imageV2Resolver) ScanNotes(_ context.Context) []string {
	value := resolver.data.GetScan().GetNotes()
	return stringSlice(value)
}

func (resolver *imageV2Resolver) OperatingSystem(_ context.Context) string {
	value := resolver.data.GetScan().GetOperatingSystem()
	return value
}

func (resolver *imageV2Resolver) ScanTime(_ context.Context) (*graphql.Time, error) {
	value := resolver.data.GetScan().GetScanTime()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageV2Resolver) ScannerVersion(_ context.Context) string {
	value := resolver.data.GetScan().GetScannerVersion()
	return value
}

func (resolver *imageV2Resolver) SignatureCount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)

	imageLoader, err := loaders.GetImageV2Loader(ctx)
	if err != nil {
		return 0, err
	}

	image, err := imageLoader.FullImageWithID(ctx, resolver.data.GetId())
	if err != nil {
		return 0, err
	}
	return int32(len(image.GetSignature().GetSignatures())), nil
}

func (resolver *imageV2Resolver) ComponentCount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetComponentCount(), nil
}

func (resolver *imageV2Resolver) CVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetCveCount(), nil
}

func (resolver *imageV2Resolver) FixableCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetFixableCveCount(), nil
}

func (resolver *imageV2Resolver) UnknownCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetUnknownCveCount(), nil
}

func (resolver *imageV2Resolver) FixableUnknownCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetFixableUnknownCveCount(), nil
}

func (resolver *imageV2Resolver) CriticalCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetCriticalCveCount(), nil
}

func (resolver *imageV2Resolver) FixableCriticalCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetFixableCriticalCveCount(), nil
}

func (resolver *imageV2Resolver) ImportantCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetImportantCveCount(), nil
}

func (resolver *imageV2Resolver) FixableImportantCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetFixableImportantCveCount(), nil
}

func (resolver *imageV2Resolver) ModerateCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetModerateCveCount(), nil
}

func (resolver *imageV2Resolver) FixableModerateCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetFixableModerateCveCount(), nil
}

func (resolver *imageV2Resolver) LowCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetLowCveCount(), nil
}

func (resolver *imageV2Resolver) FixableLowCVECount(ctx context.Context) (int32, error) {
	resolver.ensureData(ctx)
	return resolver.data.GetScanStats().GetFixableLowCveCount(), nil
}
