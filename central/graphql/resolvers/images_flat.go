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
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddExtraResolvers("ImageFlat", []string{
			"deploymentCount(query: String): Int!",
			"deployments(query: String, pagination: Pagination): [Deployment!]!",
			"imageComponentCount(query: String): Int!",
			"imageComponents(query: String, pagination: Pagination): [ImageComponent!]!",
			"imageCVECountBySeverity(query: String): ResourceCountByCVESeverity!",
			"imageVulnerabilityCount(query: String): Int!",
			"imageVulnerabilityCounter(query: String): VulnerabilityCounter!",
			"imageVulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [ImageVulnerabilityFlat]!",
			"plottedImageVulnerabilities(query: String): ImageVulnerabilityFlat!",
			"scan: ImageScan",
			"topImageVulnerability(query: String): ImageVulnerabilityFlat",
			"unusedVarSink(query: String): Int",
			"watchStatus: ImageWatchStatus!",

			// Image scan-related fields
			"dataSource: DataSource",
			"scanNotes: [ImageScan_Note!]!",
			"operatingSystem: String!",
			"scanTime: Time",
			"scannerVersion: String!",
		}),
		schema.AddQuery("imageFlat(id: ID!): ImageFlat"),
		schema.AddQuery("fullImageFlat(id: ID!): ImageFlat"),
		schema.AddQuery("imagesFlat(query: String, pagination: Pagination): [ImageFlat!]!"),
		schema.AddQuery("imageFlatCount(query: String): Int!"),
		schema.AddEnumType("ImageFlatWatchStatus", imageWatchStatuses),
	)
}

// ImagesFlat returns GraphQL resolvers for all images
func (resolver *Resolver) ImagesFlat(ctx context.Context, args PaginatedQuery) ([]*imageFlatResolver, error) {
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
	return resolver.wrapImagesFlatWithContext(ctx, images, err)
}

// ImageFlatCount returns count of all images across deployments
func (resolver *Resolver) ImageFlatCount(ctx context.Context, args RawQuery) (int32, error) {
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

// ImageFlat returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) ImageFlat(ctx context.Context, args struct{ ID graphql.ID }) (*imageFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Image")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FromID(ctx, string(args.ID))
	return resolver.wrapImageFlatWithContext(ctx, image, image != nil, err)
}

// FullImageFlat returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) FullImageFlat(ctx context.Context, args struct{ ID graphql.ID }) (*imageFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "FullImage")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FullImageWithID(ctx, string(args.ID))
	return resolver.wrapImageFlatWithContext(ctx, image, image != nil, err)
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageFlatResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Deployments")
	return resolver.root.Deployments(resolver.withImageScopeContext(ctx), args)
}

// DeploymentCount returns the number of deployments which use this image for the identified image, if it exists
func (resolver *imageFlatResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "DeploymentCount")
	return resolver.root.DeploymentCount(resolver.withImageScopeContext(ctx), args)
}

// TopImageVulnerability returns the image vulnerability with the top CVSS score.
func (resolver *imageFlatResolver) TopImageVulnerability(ctx context.Context, args RawQuery) (ImageVulnerabilityFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "TopImageVulnerability")
	return resolver.root.TopImageVulnerability(resolver.withImageScopeContext(ctx), args)
}

// ImageVulnerabilities returns, as ImageVulnerabilityResolver, the vulnerabilities for the image
func (resolver *imageFlatResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityFlatResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilities")
	// TODO(ROX-28320): Data here needs to be grouped by CVE not the ID as is done in the Image Vulnerabilities resolver.
	return resolver.root.ImageVulnerabilitiesFlat(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageFlatResolver) ImageVulnerabilityCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageVulnerabilityCount")
	return resolver.root.ImageCVEFlatCount(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageFlatResolver) ImageVulnerabilityCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
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

func (resolver *imageFlatResolver) ImageCVECountBySeverity(ctx context.Context, q RawQuery) (*resourceCountBySeverityResolver, error) {
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

func (resolver *imageFlatResolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageComponents")
	return resolver.root.ImageComponents(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageFlatResolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageFlatResolver) withImageScopeContext(ctx context.Context) context.Context {
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

func (resolver *imageFlatResolver) withElevatedImageScopeContext(ctx context.Context) context.Context {
	return sac.WithGlobalAccessScopeChecker(
		resolver.withImageScopeContext(ctx),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		),
	)
}

func (resolver *Resolver) getImageFlat(ctx context.Context, id string) *storage.Image {
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
func (resolver *imageFlatResolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "PlottedImageVulnerabilities")
	return resolver.root.PlottedImageVulnerabilities(resolver.withImageScopeContext(ctx), args)
}

func (resolver *imageFlatResolver) Scan(ctx context.Context) (*imageScanResolver, error) {
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

func (resolver *imageFlatResolver) WatchStatus(ctx context.Context) (string, error) {
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

func (resolver *imageFlatResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

//// Image scan-related fields pulled as direct sub-resolvers of image.

func (resolver *imageFlatResolver) DataSource(_ context.Context) (*dataSourceResolver, error) {
	value := resolver.data.GetScan().GetDataSource()
	return resolver.root.wrapDataSource(value, true, nil)
}

func (resolver *imageFlatResolver) ScanNotes(_ context.Context) []string {
	value := resolver.data.GetScan().GetNotes()
	return stringSlice(value)
}

func (resolver *imageFlatResolver) OperatingSystem(_ context.Context) string {
	value := resolver.data.GetScan().GetOperatingSystem()
	return value
}

func (resolver *imageFlatResolver) ScanTime(_ context.Context) (*graphql.Time, error) {
	value := resolver.data.GetScan().GetScanTime()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageFlatResolver) ScannerVersion(_ context.Context) string {
	value := resolver.data.GetScan().GetScannerVersion()
	return value
}

type imageFlatResolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.Image
	list *storage.ListImage
}

func (resolver *Resolver) wrapImageFlat(value *storage.Image, ok bool, err error) (*imageFlatResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageFlatResolver{root: resolver, data: value, list: nil}, nil
}

func (resolver *Resolver) wrapImagesFlat(values []*storage.Image, err error) ([]*imageFlatResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageFlatResolver, len(values))
	for i, v := range values {
		output[i] = &imageFlatResolver{root: resolver, data: v, list: nil}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageFlatWithContext(ctx context.Context, value *storage.Image, ok bool, err error) (*imageFlatResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageFlatResolver{ctx: ctx, root: resolver, data: value, list: nil}, nil
}

func (resolver *Resolver) wrapImagesFlatWithContext(ctx context.Context, values []*storage.Image, err error) ([]*imageFlatResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageFlatResolver, len(values))
	for i, v := range values {
		output[i] = &imageFlatResolver{ctx: ctx, root: resolver, data: v, list: nil}
	}
	return output, nil
}

func (resolver *Resolver) wrapListImagesFlat(values []*storage.ListImage, err error) ([]*imageFlatResolver, error) {
	if err != nil || values == nil {
		return nil, err
	}
	output := make([]*imageFlatResolver, len(values))
	for i, v := range values {
		output[i] = &imageFlatResolver{root: resolver, data: nil, list: v}
	}
	return output, nil
}

func (resolver *imageFlatResolver) ensureData(ctx context.Context) {
	if resolver.data == nil {
		resolver.data = resolver.root.getImageFlat(ctx, resolver.list.GetId())
	}
}

func (resolver *imageFlatResolver) Id(_ context.Context) graphql.ID {
	value := resolver.data.GetId()
	if resolver.data == nil {
		value = resolver.list.GetId()
	}
	return graphql.ID(value)
}

func (resolver *imageFlatResolver) IsClusterLocal(ctx context.Context) bool {
	resolver.ensureData(ctx)
	value := resolver.data.GetIsClusterLocal()
	return value
}

func (resolver *imageFlatResolver) LastUpdated(_ context.Context) (*graphql.Time, error) {
	value := resolver.data.GetLastUpdated()
	if resolver.data == nil {
		value = resolver.list.GetLastUpdated()
	}
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageFlatResolver) Metadata(ctx context.Context) (*imageMetadataResolver, error) {
	resolver.ensureData(ctx)
	value := resolver.data.GetMetadata()
	return resolver.root.wrapImageMetadata(value, true, nil)
}

func (resolver *imageFlatResolver) Name(ctx context.Context) (*imageNameResolver, error) {
	resolver.ensureData(ctx)
	value := resolver.data.GetName()
	return resolver.root.wrapImageName(value, true, nil)
}

func (resolver *imageFlatResolver) Names(ctx context.Context) ([]*imageNameResolver, error) {
	resolver.ensureData(ctx)
	value := resolver.data.GetNames()
	return resolver.root.wrapImageNames(value, nil)
}

func (resolver *imageFlatResolver) NotPullable(ctx context.Context) bool {
	resolver.ensureData(ctx)
	value := resolver.data.GetNotPullable()
	return value
}

func (resolver *imageFlatResolver) Notes(ctx context.Context) []string {
	resolver.ensureData(ctx)
	value := resolver.data.GetNotes()
	return stringSlice(value)
}

func (resolver *imageFlatResolver) Priority(_ context.Context) int32 {
	value := resolver.data.GetPriority()
	if resolver.data == nil {
		value = resolver.list.GetPriority()
	}
	return int32(value)
}

func (resolver *imageFlatResolver) RiskScore(ctx context.Context) float64 {
	resolver.ensureData(ctx)
	value := resolver.data.GetRiskScore()
	return float64(value)
}

func (resolver *imageFlatResolver) Signature(ctx context.Context) (*imageSignatureResolver, error) {
	resolver.ensureData(ctx)
	value := resolver.data.GetSignature()
	return resolver.root.wrapImageSignature(value, true, nil)
}

func (resolver *imageFlatResolver) SignatureVerificationData(ctx context.Context) (*imageSignatureVerificationDataResolver, error) {
	resolver.ensureData(ctx)
	value := resolver.data.GetSignatureVerificationData()
	return resolver.root.wrapImageSignatureVerificationData(value, true, nil)
}
