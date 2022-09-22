package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("permissionSets: [PermissionSet!]!"),
		schema.AddQuery("permissionSet(id: ID): PermissionSet"),
		schema.AddExtraResolver("PermissionSet", "resourceToAccess: [Label!]!"),
	)
}

// PermissionSets returns GraphQL resolvers for all permission sets
func (resolver *Resolver) PermissionSets(ctx context.Context) ([]*permissionSetResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PermissionSets")
	err := readAccess(ctx)
	if err != nil {
		return nil, err
	}
	permissionSets, err := resolver.RoleDataStore.GetAllPermissionSets(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to retrieve permission sets")
	}
	return resolver.wrapPermissionSets(permissionSets, nil)
}

// PermissionSet returns a GraphQL resolver for the matching permission set, if it exists
func (resolver *Resolver) PermissionSet(ctx context.Context, args struct{ *graphql.ID }) (*permissionSetResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "PermissionSet")
	err := readAccess(ctx)
	if err != nil {
		return nil, err
	}

	permissionSet, found, err := resolver.RoleDataStore.GetPermissionSet(ctx, string(*args.ID))
	return resolver.wrapPermissionSet(permissionSet, found, err)
}

// Enable returning of the ResourceToAccess map.
func (resolver *permissionSetResolver) ResourceToAccess() labels {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.PermissionSets, "ResourceIDToAccess")
	resourceToAccess := resolver.data.GetResourceToAccess()
	if resourceToAccess == nil {
		return nil
	}
	resourceToString := make(map[string]string)
	for resource, access := range resourceToAccess {
		resourceToString[resource] = access.String()
	}
	return labelsResolver(resourceToString)
}
