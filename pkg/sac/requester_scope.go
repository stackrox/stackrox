package sac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// GetRequesterScopeForReadPermission returns the effective access scope
// tied to the requester ID in context for read on the provided resources.
func GetRequesterScopeForReadPermission(
	ctx context.Context,
	targetResource permissions.ResourceWithAccess,
) (*effectiveaccessscope.ScopeTree, error) {
	sc := ForResource(targetResource.Resource).ScopeChecker(ctx, storage.Access_READ_ACCESS)
	return sc.EffectiveAccessScope(permissions.View(targetResource.Resource))
}
