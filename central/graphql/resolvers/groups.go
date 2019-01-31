package resolvers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

func init() {
	schema := getBuilder()
	schema.AddQuery("groups: [Group!]!")
	schema.AddQuery("group(authProviderId: String, key: String, value: String): Group")
}

// Groups returns GraphQL resolvers for all groups
func (resolver *Resolver) Groups(ctx context.Context) ([]*groupResolver, error) {
	err := readGroups(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.wrapGroups(resolver.GroupDataStore.GetAll())
}

// Group returns a GraphQL resolver for the matching group, if it exists
func (resolver *Resolver) Group(ctx context.Context, args struct{ AuthProviderID, Key, Value *string }) (*groupResolver, error) {
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
	grp, err := resolver.GroupDataStore.Get(props)
	return resolver.wrapGroup(grp, grp != nil, err)
}
