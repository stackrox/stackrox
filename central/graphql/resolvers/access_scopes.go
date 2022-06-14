package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/stackrox/central/metrics"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("simpleAccessScopes: [SimpleAccessScope!]!"),
		schema.AddQuery("simpleAccessScope(id: ID): SimpleAccessScope"),
	)
}

// SimpleAccessScopes returns GraphQL resolvers for all simple access scopes.
func (resolver *Resolver) SimpleAccessScopes(ctx context.Context) ([]*simpleAccessScopeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "SimpleAccessScopes")
	err := readRoles(ctx)
	if err != nil {
		return nil, err
	}

	return resolver.wrapSimpleAccessScopes(resolver.RoleDataStore.GetAllAccessScopes(ctx))
}

// SimpleAccessScope returns a GraphQL resolver for the matching access scope
// if it exists.
func (resolver *Resolver) SimpleAccessScope(ctx context.Context, args struct{ *graphql.ID }) (*simpleAccessScopeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "SimpleAccessScope")
	err := readRoles(ctx)
	if err != nil {
		return nil, err
	}

	return resolver.wrapSimpleAccessScope(resolver.RoleDataStore.GetAccessScope(ctx, string(*args.ID)))
}
