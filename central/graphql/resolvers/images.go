package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("images(query: String, pagination: Pagination): [Image!]!"),
		schema.AddQuery("imageCount(query: String): Int!"),
		schema.AddQuery("image(sha:ID!): Image"),
		schema.AddExtraResolver("Image", "deployments(query: String): [Deployment!]!"),
		schema.AddExtraResolver("Image", "deploymentCount: Int!"),
		schema.AddExtraResolver("Image", "topVuln(query: String): EmbeddedVulnerability"),
		schema.AddExtraResolver("Image", "vulns(query: String): [EmbeddedVulnerability]!"),
		schema.AddExtraResolver("Image", "vulnCount(query: String): Int!"),
		schema.AddExtraResolver("Image", "vulnCounter: VulnerabilityCounter!"),
		schema.AddExtraResolver("EmbeddedImageScanComponent", "layerIndex: Int"),
	)
}

// Images returns GraphQL resolvers for all images
func (resolver *Resolver) Images(ctx context.Context, args paginatedQuery) ([]*imageResolver, error) {
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
func (resolver *Resolver) ImageCount(ctx context.Context, args rawQuery) (int32, error) {
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
func (resolver *Resolver) Image(ctx context.Context, args struct{ Sha graphql.ID }) (*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Image")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	image, err := imageLoader.FromID(ctx, string(args.Sha))
	return resolver.wrapImage(image, image != nil, err)
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Deployments")
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	imageIDQuery := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, resolver.data.GetId()).ProtoQuery()

	return resolver.root.wrapDeployments(
		resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, search.NewConjunctionQuery(imageIDQuery, q)))
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) DeploymentCount(ctx context.Context) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, resolver.data.GetId()).ProtoQuery()
	results, err := resolver.root.DeploymentDataStore.Search(ctx, query)
	if err != nil {
		return 0, nil
	}
	return int32(len(results)), nil
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (resolver *imageResolver) TopVuln(ctx context.Context, args rawQuery) (*EmbeddedVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "TopVulnerability")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	resolvers, err := mapImagesToVulnerabilityResolvers(resolver.root, []*storage.Image{resolver.data}, query)
	if err != nil {
		return nil, err
	}

	// create a set of the CVEs to return.
	var maxCvss *storage.EmbeddedVulnerability
	for _, resolver := range resolvers {
		if maxCvss == nil || resolver.data.GetCvss() > maxCvss.GetCvss() {
			maxCvss = resolver.data
		}
	}
	return resolver.root.wrapEmbeddedVulnerability(maxCvss, nil)
}

// Vulns returns all of the vulnerabilities in the image.
func (resolver *imageResolver) Vulns(ctx context.Context, args rawQuery) ([]*EmbeddedVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "Vulnerabilities")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return mapImagesToVulnerabilityResolvers(resolver.root, []*storage.Image{resolver.data}, query)
}

// VulnCount returns the number of vulnerabilities the image has.
func (resolver *imageResolver) VulnCount(ctx context.Context, args rawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Images, "VulnerabilityCount")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	resolvers, err := mapImagesToVulnerabilityResolvers(resolver.root, []*storage.Image{resolver.data}, query)
	if err != nil {
		return 0, err
	}
	return int32(len(resolvers)), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (resolver *imageResolver) VulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return mapImagesToVulnerabilityCounter([]*storage.Image{resolver.data}), nil
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
