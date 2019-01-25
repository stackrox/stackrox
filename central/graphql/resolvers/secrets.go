package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func init() {
	schema := getBuilder()
	schema.AddQuery("secret(id:ID!): Secret")
	schema.AddQuery("secrets(query: String): [Secret!]!")
	schema.AddExtraResolver("Secret", "deployments(): [Deployment!]!")
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
func (resolver *Resolver) Secrets(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	if err := secretAuth(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if q != nil {
		return resolver.wrapListSecrets(resolver.SecretsDataStore.SearchListSecrets(q))
	}
	return resolver.wrapListSecrets(resolver.SecretsDataStore.ListSecrets())
}

func (resolver *Resolver) getSecret(id string) *storage.Secret {
	secret, ok, err := resolver.SecretsDataStore.GetSecret(id)
	if err != nil || !ok {
		return nil
	}
	return secret
}

func (resolver *secretResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := deploymentAuth(ctx); err != nil {
		return nil, err
	}
	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.Namespace, resolver.data.GetNamespace()).
		AddExactMatches(search.SecretName, resolver.data.GetName()).
		ProtoQuery()
	return resolver.root.wrapListDeployments(
		resolver.root.DeploymentDataStore.SearchListDeployments(psr))
}
