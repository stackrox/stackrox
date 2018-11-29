package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/schema"
)

func init() {
	schema.AddQuery("secret(id:ID!): Secret")
	schema.AddQuery("secrets(): [Secret!]!")
}

// Secret gets a single secret by ID
func (resolver *Resolver) Secret(ctx context.Context, arg struct{ graphql.ID }) (*secretResolver, error) {
	if err := secretAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapSecret(
		resolver.SecretsDataStore.GetSecret(string(arg.ID)))
}

// Secrets gets a list of all secrets
func (resolver *Resolver) Secrets(ctx context.Context) ([]*secretResolver, error) {
	if err := secretAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapListSecrets(resolver.SecretsDataStore.ListSecrets())
}
