package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/namespace"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("namespaces(query: String): [Namespace!]!"),
		schema.AddQuery("namespace(id: ID!): Namespace"),
		schema.AddQuery("namespaceByClusterIDAndName(clusterID: ID!, name: String!): Namespace"),
		schema.AddExtraResolver("Namespace", `complianceResults(query: String): [ControlResult!]!`),
		schema.AddExtraResolver("Namespace", `images(clusterId : ID!): [Image!]!`),
		schema.AddExtraResolver("Namespace", `imageCount(clusterId : ID!): Int!`),
	)
}

// Namespace returns a GraphQL resolver for the given namespace.
func (resolver *Resolver) Namespace(ctx context.Context, args struct{ graphql.ID }) (*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByID(ctx, string(args.ID), resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

// Namespaces returns GraphQL resolvers for all namespaces based on an optional query.
func (resolver *Resolver) Namespaces(ctx context.Context, args rawQuery) ([]*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if query == nil {
		return resolver.wrapNamespaces(namespace.ResolveAll(ctx, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
	}
	return resolver.wrapNamespaces(namespace.ResolveByQuery(ctx, query, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

type clusterIDAndNameQuery struct {
	ClusterID graphql.ID
	Name      string
}

// NamespaceByClusterIDAndName returns a GraphQL resolver for the (unique) namespace specified by this query.
func (resolver *Resolver) NamespaceByClusterIDAndName(ctx context.Context, args clusterIDAndNameQuery) (*namespaceResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNamespace(namespace.ResolveByClusterIDAndName(ctx, string(args.ClusterID), args.Name, resolver.NamespaceDataStore, resolver.DeploymentDataStore, resolver.SecretsDataStore, resolver.NetworkPoliciesStore))
}

func (resolver *namespaceResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nsID := resolver.data.GetMetadata().GetId()
	output.addDeploymentData(resolver.root, runResults, func(d *storage.Deployment, _ *v1.ComplianceControl) bool {
		return d.GetNamespaceId() == nsID
	})

	return *output, nil
}

func (resolver *namespaceResolver) Images(ctx context.Context, args struct{ ClusterID graphql.ID }) ([]*imageResolver, error) {
	if err := readNamespaces(ctx); err != nil {
		return nil, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, string(args.ClusterID)).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).ProtoQuery()
	return resolver.root.wrapListImages(resolver.root.ImageDataStore.SearchListImages(ctx, q))
}

func (resolver *namespaceResolver) ImageCount(ctx context.Context, args struct{ ClusterID graphql.ID }) (int32, error) {
	if err := readNamespaces(ctx); err != nil {
		return 0, err
	}
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, string(args.ClusterID)).
		AddExactMatches(search.Namespace, resolver.data.Metadata.GetName()).ProtoQuery()
	results, err := resolver.root.ImageDataStore.Search(ctx, q)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}
