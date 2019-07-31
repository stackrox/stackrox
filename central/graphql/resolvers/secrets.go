package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("secret(id:ID!): Secret"),
		schema.AddQuery("secrets(query: String): [Secret!]!"),
		schema.AddExtraResolver("Secret", "deployments(query: String): [Deployment!]!"),
	)
}

// Secret gets a single secret by ID
func (resolver *Resolver) Secret(ctx context.Context, arg struct{ graphql.ID }) (*secretResolver, error) {
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}

	secret := resolver.getSecret(ctx, string(arg.ID))
	if secret == nil {
		return resolver.wrapSecret(nil, false, errors.Errorf("error locating secret with id: %s", arg.ID))
	}
	return resolver.wrapSecret(secret, true, nil)
}

// Secrets gets a list of all secrets
func (resolver *Resolver) Secrets(ctx context.Context, args rawQuery) ([]*secretResolver, error) {
	if err := readSecrets(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	secrets, err := resolver.SecretsDataStore.SearchRawSecrets(ctx, q)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		resolver.getDeploymentRelationships(ctx, secret)
	}
	return resolver.wrapSecrets(secrets, nil)
}

func (resolver *secretResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	deploymentFilterQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	secret := resolver.data
	deploymentIDs := set.NewStringSet()

	for _, dr := range secret.Relationship.GetDeploymentRelationships() {
		deploymentIDs.Add(dr.GetId())
	}
	deploymentIDQuery := search.NewQueryBuilder().AddDocIDs(deploymentIDs.AsSlice()...).ProtoQuery()

	return resolver.root.wrapDeployments(
		resolver.root.DeploymentDataStore.SearchRawDeployments(ctx, search.NewConjunctionQuery(deploymentIDQuery, deploymentFilterQuery)))
}

func (resolver *Resolver) getSecret(ctx context.Context, id string) *storage.Secret {
	secret, ok, err := resolver.SecretsDataStore.GetSecret(ctx, id)
	if err != nil || !ok {
		return nil
	}

	resolver.getDeploymentRelationships(ctx, secret)
	return secret
}

func (resolver *Resolver) getDeploymentRelationships(ctx context.Context, secret *storage.Secret) {
	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, secret.GetClusterId()).
		AddExactMatches(search.Namespace, secret.GetNamespace()).
		AddExactMatches(search.SecretName, secret.GetName()).
		ProtoQuery()

	deploymentResults, err := resolver.DeploymentDataStore.SearchListDeployments(ctx, psr)
	if err != nil {
		return
	}

	var deployments []*storage.SecretDeploymentRelationship
	for _, r := range deploymentResults {
		deployments = append(deployments, &storage.SecretDeploymentRelationship{
			Id:   r.Id,
			Name: r.Name,
		})
	}
	secret.Relationship = &storage.SecretRelationship{
		DeploymentRelationships: deployments,
	}
}
