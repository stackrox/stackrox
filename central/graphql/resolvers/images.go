package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/schema"
)

func init() {
	schema.AddQuery("images: [Image!]!")
	schema.AddQuery("image(sha:ID!): Image")
}

// Images returns GraphQL resolvers for all images
func (resolver *Resolver) Images(ctx context.Context) ([]*imageResolver, error) {
	if err := imageAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapImages(resolver.ImageDataStore.GetImages())
}

// Image returns a graphql resolver for the identified image, if it exists
func (resolver *Resolver) Image(ctx context.Context, args struct{ Sha graphql.ID }) (*imageResolver, error) {
	if err := imageAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapImage(
		resolver.ImageDataStore.GetImage(string(args.Sha)))
}
