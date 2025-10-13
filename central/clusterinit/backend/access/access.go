package access

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

// CheckAccess returns nil if requested access level is granted in context.
func CheckAccess(ctx context.Context, access storage.Access) error {
	helper := sac.ForResources(sac.ForResource(resources.Administration), sac.ForResource(resources.Integration))
	if allowed, err := helper.AccessAllowedToAll(ctx, access); err != nil {
		return errors.Wrap(err, "checking access")
	} else if !allowed {
		return errox.NotAuthorized
	}
	return nil
}
