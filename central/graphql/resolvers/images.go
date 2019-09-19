package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
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
	return resolver.wrapImages(
		resolver.ImageDataStore.SearchRawImages(ctx, q))
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
	results, err := resolver.ImageDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

// Image returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) Image(ctx context.Context, args struct{ Sha graphql.ID }) (*imageResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Image")

	if err := readImages(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapImage(
		resolver.ImageDataStore.GetImage(ctx, string(args.Sha)))
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

func (resolver *Resolver) getImage(ctx context.Context, id string) *storage.Image {
	alert, ok, err := resolver.ImageDataStore.GetImage(ctx, id)
	if err != nil || !ok {
		return nil
	}
	return alert
}

func (resolver *embeddedImageScanComponentResolver) LayerIndex() *int32 {
	w, ok := resolver.data.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
	if !ok {
		return nil
	}
	v := w.LayerIndex
	return &v
}
