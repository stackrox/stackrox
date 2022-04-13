package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("groups: [Group!]!"),
		schema.AddQuery("group(authProviderId: String, key: String, value: String): Group"),
	)
}

// Groups returns GraphQL resolvers for all groups
func (resolver *Resolver) Groups(ctx context.Context) ([]*groupResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Groups")

	err := readGroups(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.wrapGroups(resolver.GroupDataStore.GetAll(ctx))
}

// Group returns a GraphQL resolver for the matching group, if it exists
func (resolver *Resolver) Group(ctx context.Context, args struct{ AuthProviderID, Key, Value *string }) (*groupResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Group")

	err := readGroups(ctx)
	if err != nil {
		return nil, err
	}
	props := &storage.GroupProperties{}
	if args.AuthProviderID != nil {
		props.AuthProviderId = *args.AuthProviderID
	}
	if args.Key != nil {
		props.Key = *args.Key
	}
	if args.Value != nil {
		props.Value = *args.Value
	}
	grp, err := resolver.GroupDataStore.Get(ctx, props)
	return resolver.wrapGroup(grp, grp != nil, err)
}
