package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/schema"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func init() {
	schema.AddQuery("policies(query: String): [Policy!]!")
	schema.AddQuery("policy(id: ID): Policy")
	schema.AddResolver(&storage.Policy{}, `alerts: [Alert!]!`)
}

// Policies returns GraphQL resolvers for all policies
func (resolver *Resolver) Policies(ctx context.Context, args rawQuery) ([]*policyResolver, error) {
	if err := policyAuth(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if q == nil {
		return resolver.wrapPolicies(resolver.PolicyDataStore.GetPolicies())
	}
	return resolver.wrapPolicies(resolver.PolicyDataStore.SearchRawPolicies(q))
}

// Policy returns a GraphQL resolver for a given policy
func (resolver *Resolver) Policy(ctx context.Context, args struct{ *graphql.ID }) (*policyResolver, error) {
	if err := policyAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapPolicy(resolver.PolicyDataStore.GetPolicy(string(*args.ID)))
}

// Alerts returns GraphQL resolvers for all alerts for this policy
func (resolver *policyResolver) Alerts(ctx context.Context) ([]*alertResolver, error) {
	if err := alertAuth(ctx); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddStrings(search.PolicyID, resolver.data.GetId()).ProtoQuery()
	return resolver.root.wrapAlerts(
		resolver.root.ViolationsDataStore.SearchRawAlerts(query))
}
