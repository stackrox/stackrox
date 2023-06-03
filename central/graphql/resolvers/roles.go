package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/role/service"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("roles: [Role!]!"),
		schema.AddQuery("role(id: ID): Role"),
		schema.AddExtraResolver("Role", "resourceToAccess: [Label!]!"),
		schema.AddQuery("myPermissions: GetPermissionsResponse"),
		schema.AddExtraResolver("GetPermissionsResponse", "resourceToAccess: [Label!]!"),
	)
}

// Roles returns GraphQL resolvers for all roles
func (resolver *Resolver) Roles(ctx context.Context) ([]*roleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Roles")
	err := readAccess(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := resolver.RoleDataStore.GetAllRoles(ctx)
	if err != nil {
		return nil, errors.New("unable to retrieve roles")
	}
	return resolver.wrapRoles(roles, nil)
}

// Role returns a GraphQL resolver for the matching role, if it exists
func (resolver *Resolver) Role(ctx context.Context, args struct{ *graphql.ID }) (*roleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Role")
	err := readAccess(ctx)
	if err != nil {
		return nil, err
	}

	role, found, err := resolver.RoleDataStore.GetRole(ctx, string(*args.ID))
	return resolver.wrapRole(role, found, err)
}

// MyPermissions returns a GraphQL resolver for the role of the current authenticated user. Only supplies permissions.
func (resolver *Resolver) MyPermissions(ctx context.Context) (*getPermissionsResponseResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "MyPermissions")
	perms, err := service.GetMyPermissions(ctx)
	return resolver.wrapGetPermissionsResponse(perms, perms != nil, err)
}

// Enable returning of the ResourceToAccess map for Role.
func (resolver *roleResolver) ResourceToAccess() labels {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Roles, "ResourceToAccess")
	rToA := resolver.data.GetResourceToAccess()
	if rToA == nil {
		return nil
	}
	rToS := make(map[string]string)
	for resource, access := range rToA {
		rToS[resource] = access.String()
	}
	return labelsResolver(rToS)
}

// Enable returning of the ResourceToAccess map for MyPermissions.
func (resolver *getPermissionsResponseResolver) ResourceToAccess() labels {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Roles, "ResourceToAccess")
	rToA := resolver.data.GetResourceToAccess()
	if rToA == nil {
		return nil
	}
	rToS := make(map[string]string)
	for resource, access := range rToA {
		rToS[resource] = access.String()
	}
	return labelsResolver(rToS)
}
