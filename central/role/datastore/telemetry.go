package datastore

import (
	"context"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather a few properties for phone home telemetry.
func Gather(ctx context.Context) (phonehome.Properties, error) {
	// WithAllAccess is required only to fetch and calculate the number of
	// permission sets, roles and access scopes. It is not propagated anywhere
	// else.
	ctx = sac.WithAllAccess(ctx)
	totals := make(phonehome.Properties)
	rs := Singleton()

	gatherErrs := errorhelpers.NewErrorList("cannot gather from role store")
	gatherErrs.AddError(phonehome.AddTotal(ctx, totals, "PermissionSets", rs.GetAllPermissionSets))
	gatherErrs.AddError(phonehome.AddTotal(ctx, totals, "Roles", rs.GetAllRoles))
	gatherErrs.AddError(phonehome.AddTotal(ctx, totals, "Access Scopes", rs.GetAllAccessScopes))

	return totals, gatherErrs.ToError()
}
