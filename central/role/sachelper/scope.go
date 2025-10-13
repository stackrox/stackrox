package sachelper

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

func getRequesterScopeForReadPermission(
	ctx context.Context,
	resourceWithAccess permissions.ResourceWithAccess,
) (*effectiveaccessscope.ScopeTree, error) {
	scopeChecker := sac.ForResource(resourceWithAccess.Resource).ScopeChecker(ctx, storage.Access_READ_ACCESS)
	return scopeChecker.EffectiveAccessScope(resourceWithAccess)
}
