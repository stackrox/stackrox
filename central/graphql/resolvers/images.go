package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("images(query:String): [Image!]!"),
		schema.AddQuery("image(sha:ID!): Image"),
		schema.AddExtraResolver("Image", "deployments(query: String): [Deployment!]!"),
		schema.AddExtraResolver("Image", "deploymentCount: Int!"),
		schema.AddExtraResolver("ImageScanComponent", "layerIndex: Int"),
	)
}

// Images returns GraphQL resolvers for all images
func (resolver *Resolver) Images(ctx context.Context, args rawQuery) ([]*imageResolver, error) {
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

// Image returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) Image(ctx context.Context, args struct{ Sha graphql.ID }) (*imageResolver, error) {
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapImage(
		resolver.ImageDataStore.GetImage(ctx, string(args.Sha)))
}

// Deployments returns the deployments which use this image for the identified image, if it exists
func (resolver *imageResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
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

func (resolver *imageScanComponentResolver) LayerIndex() *int32 {
	w, ok := resolver.data.GetHasLayerIndex().(*storage.ImageScanComponent_LayerIndex)
	if !ok {
		return nil
	}
	v := w.LayerIndex
	return &v
}
