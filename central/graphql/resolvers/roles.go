package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/role/service"
	roleUtils "github.com/stackrox/rox/central/role/utils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("roles: [Role!]!"),
		schema.AddQuery("role(id: ID): Role"),
		schema.AddQuery("myPermissions: Role"),
		schema.AddExtraResolver("Role", "resourceToAccess: [Label!]!"),
	)
}

// Roles returns GraphQL resolvers for all roles
func (resolver *Resolver) Roles(ctx context.Context) ([]*roleResolver, error) {
	err := readRoles(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := resolver.RoleDataStore.GetAllRoles(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to retrieve roles")
	}
	for _, role := range roles {
		roleUtils.FillAccessList(role)
	}
	return resolver.wrapRoles(roles, nil)
}

// Role returns a GraphQL resolver for the matching role, if it exists
func (resolver *Resolver) Role(ctx context.Context, args struct{ *graphql.ID }) (*roleResolver, error) {
	err := readRoles(ctx)
	if err != nil {
		return nil, err
	}

	role, err := resolver.RoleDataStore.GetRole(ctx, string(*args.ID))
	roleUtils.FillAccessList(role)
	return resolver.wrapRole(role, role != nil, err)
}

// MyPermissions returns a GraphQL resolver for the role of the current authenticated user. Only supplies permissions.
func (resolver *Resolver) MyPermissions(ctx context.Context) (*roleResolver, error) {
	role, err := service.GetMyPermissions(ctx)
	return resolver.wrapRole(role, role != nil, err)
}

// Enable returning of the ResourceToAccess map.
func (resolver *roleResolver) ResourceToAccess() labels {
	rToA := resolver.data.GetResourceToAccess()
	if rToA == nil {
		return nil
	}
	rToS := make(map[string]string)
	for resource, access := range rToA {
		rToS[resource] = string(access.String())
	}
	return labelsResolver(rToS)
}
