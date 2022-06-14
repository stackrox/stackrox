package access

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/sac"
)

// CheckAccess returns nil if requested access level is granted in context.
func CheckAccess(ctx context.Context, access storage.Access) error {
	helper := sac.ForResources(sac.ForResource(resources.ServiceIdentity), sac.ForResource(resources.APIToken))
	if allowed, err := helper.AccessAllowedToAll(ctx, access); err != nil {
		return errors.Wrap(err, "checking access")
	} else if !allowed {
		return errox.NotAuthorized
	}
	return nil
}
