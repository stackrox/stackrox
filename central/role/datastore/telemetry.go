package datastore

import (
	"context"

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
			sac.ResourceScopeKeys(resources.Role)))

	totals := make(map[string]any)
	rs := Singleton()

	gatherErrs := errorhelpers.NewErrorList("cannot gather from role store")
	gatherErrs.AddError(phonehome.AddTotal(ctx, totals, "PermissionSets", rs.GetAllPermissionSets))
	gatherErrs.AddError(phonehome.AddTotal(ctx, totals, "Roles", rs.GetAllRoles))
	gatherErrs.AddError(phonehome.AddTotal(ctx, totals, "Access Scopes", rs.GetAllAccessScopes))

	return totals, gatherErrs.ToError()
}
