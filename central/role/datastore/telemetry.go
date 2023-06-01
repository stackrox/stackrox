package datastore

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather the total amount of permission sets, access scopes, and roles for
// phone home telemetry.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	totals := make(map[string]any)
	rs := Singleton()

	gatherErrs := errorhelpers.NewErrorList("cannot gather from role store")
	gatherErrs.AddError(totalPermissionSets(ctx, totals, rs))
	gatherErrs.AddError(totalRoles(ctx, totals, rs))
	gatherErrs.AddError(totalAccessScopes(ctx, totals, rs))

	return totals, gatherErrs.ToError()
}

func totalPermissionSets(ctx context.Context, props map[string]any, rs DataStore) error {
	permissionSets, err := rs.GetAllPermissionSets(ctx)
	if err != nil {
		return err
	}

	permissionSetsByOrigin := make(map[storage.Traits_Origin]int)

	for _, ps := range permissionSets {
		permissionSetsByOrigin[ps.GetTraits().GetOrigin()]++
	}

	props["Total PermissionSets"] = len(permissionSets)
	for origin, count := range permissionSetsByOrigin {
		props[fmt.Sprintf("Total %s PermissionSets", strings.ToLower(origin.String()))] = count
	}
	return nil
}

func totalRoles(ctx context.Context, props map[string]any, rs DataStore) error {
	roles, err := rs.GetAllRoles(ctx)
	if err != nil {
		return err
	}

	rolesByOrigin := make(map[storage.Traits_Origin]int)

	for _, role := range roles {
		rolesByOrigin[role.GetTraits().GetOrigin()]++
	}

	props["Total Roles"] = len(roles)
	for origin, count := range rolesByOrigin {
		props[fmt.Sprintf("Total %s Roles", strings.ToLower(origin.String()))] = count
	}
	return nil
}

func totalAccessScopes(ctx context.Context, props map[string]any, rs DataStore) error {
	accessScopes, err := rs.GetAllAccessScopes(ctx)
	if err != nil {
		return err
	}

	accessScopesByOrigin := make(map[storage.Traits_Origin]int)

	for _, as := range accessScopes {
		accessScopesByOrigin[as.GetTraits().GetOrigin()]++
	}

	props["Total Access Scopes"] = len(accessScopes)
	for origin, count := range accessScopesByOrigin {
		props[fmt.Sprintf("Total %s Access Scopes", strings.ToLower(origin.String()))] = count
	}
	return nil
}
