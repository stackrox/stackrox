package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

func addTotal[T any](ctx context.Context, props phonehome.Properties, key string, f func(context.Context) ([]*T, error)) error {
	ps, err := f(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s", key)
	}
	props["Total "+key] = len(ps)
	return nil
}

// Gather a few properties for phone home telemetry.
func Gather(ctx context.Context) (phonehome.Properties, error) {
	ctx = sac.WithAllAccess(ctx)
	totals := make(phonehome.Properties)
	rs := Singleton()

	el := errorhelpers.NewErrorList("cannot gather from role store")
	el.AddError(addTotal(ctx, totals, "PermissionSets", rs.GetAllPermissionSets))
	el.AddError(addTotal(ctx, totals, "Roles", rs.GetAllRoles))
	el.AddError(addTotal(ctx, totals, "Access Scopes", rs.GetAllAccessScopes))

	return totals, el.ToError()
}
