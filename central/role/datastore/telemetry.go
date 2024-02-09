package datastore

import (
	"context"
	stdErrors "errors"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	var gatherErrs error
	gatherErrs = stdErrors.Join(gatherErrs, totalPermissionSets(ctx, totals, rs))
	gatherErrs = stdErrors.Join(gatherErrs, totalRoles(ctx, totals, rs))
	gatherErrs = stdErrors.Join(gatherErrs, totalAccessScopes(ctx, totals, rs))

	return totals, errors.Wrap(gatherErrs, "gather data from role store")
}

func totalPermissionSets(ctx context.Context, props map[string]any, rs DataStore) error {
	permissionSets, err := rs.GetAllPermissionSets(ctx)
	if err != nil {
		return err
	}

	permissionSetsByOrigin := map[storage.Traits_Origin]int{
		storage.Traits_DEFAULT:              0,
		storage.Traits_IMPERATIVE:           0,
		storage.Traits_DECLARATIVE:          0,
		storage.Traits_DECLARATIVE_ORPHANED: 0,
	}

	for _, ps := range permissionSets {
		permissionSetsByOrigin[ps.GetTraits().GetOrigin()]++
	}

	props["Total PermissionSets"] = len(permissionSets)
	for origin, count := range permissionSetsByOrigin {
		props[fmt.Sprintf("Total %s PermissionSets",
			cases.Title(language.English, cases.Compact).String(origin.String()))] = count
	}
	return nil
}

func totalRoles(ctx context.Context, props map[string]any, rs DataStore) error {
	roles, err := rs.GetAllRoles(ctx)
	if err != nil {
		return err
	}

	rolesByOrigin := map[storage.Traits_Origin]int{
		storage.Traits_DEFAULT:              0,
		storage.Traits_IMPERATIVE:           0,
		storage.Traits_DECLARATIVE:          0,
		storage.Traits_DECLARATIVE_ORPHANED: 0,
	}

	for _, role := range roles {
		rolesByOrigin[role.GetTraits().GetOrigin()]++
	}

	props["Total Roles"] = len(roles)
	for origin, count := range rolesByOrigin {
		props[fmt.Sprintf("Total %s Roles",
			cases.Title(language.English, cases.Compact).String(origin.String()))] = count
	}
	return nil
}

func totalAccessScopes(ctx context.Context, props map[string]any, rs DataStore) error {
	accessScopes, err := rs.GetAllAccessScopes(ctx)
	if err != nil {
		return err
	}

	accessScopesByOrigin := map[storage.Traits_Origin]int{
		storage.Traits_DEFAULT:              0,
		storage.Traits_IMPERATIVE:           0,
		storage.Traits_DECLARATIVE:          0,
		storage.Traits_DECLARATIVE_ORPHANED: 0,
	}

	for _, as := range accessScopes {
		accessScopesByOrigin[as.GetTraits().GetOrigin()]++
	}

	props["Total Access Scopes"] = len(accessScopes)
	for origin, count := range accessScopesByOrigin {
		props[fmt.Sprintf("Total %s Access Scopes",
			cases.Title(language.English, cases.Compact).String(origin.String()))] = count
	}
	return nil
}
