package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("groups: [Group!]!"),
		schema.AddQuery("group(authProviderId: String, key: String, value: String, id: String): Group"),
	)
}

// Groups returns GraphQL resolvers for all groups
func (resolver *Resolver) Groups(ctx context.Context) ([]*groupResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Groups")

	err := readAccess(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.wrapGroups(resolver.GroupDataStore.GetAll(ctx))
}

// Group returns a GraphQL resolver for the matching group, if it exists
func (resolver *Resolver) Group(ctx context.Context, args struct{ AuthProviderID, Key, Value, ID *string }) (*groupResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Group")

	err := readAccess(ctx)
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
	if args.ID != nil {
		props.Id = *args.ID
	}
	grp, err := resolver.GroupDataStore.Get(ctx, props)
	return resolver.wrapGroup(grp, grp != nil, err)
}
