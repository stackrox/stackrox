package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/namespace"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("namespaces: [Namespace!]!"),
		schema.AddQuery("namespace(id: ID!): Namespace"),
		schema.AddQuery("namespaceByClusterIDAndName(clusterID: ID!, name: String!): Namespace"),
	)
}

// Namespace returns a GraphQL resolver for the given namespace.
func (resolver *Resolver) Namespace(ctx context.Context, args struct{ graphql.ID }) (*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByID(string(args.ID), resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

// Namespaces returns GraphQL resolvers for all namespaces.
func (resolver *Resolver) Namespaces(ctx context.Context) ([]*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespaces(namespace.ResolveAll(resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

type clusterIDAndNameQuery struct {
	ClusterID graphql.ID
	Name      string
}

// NamespaceByClusterIDAndName returns a GrapQL resolver for the (unique) namespace specified by this query.
func (resolver *Resolver) NamespaceByClusterIDAndName(ctx context.Context, args clusterIDAndNameQuery) (*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByClusterIDAndName(string(args.ClusterID), args.Name, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}
