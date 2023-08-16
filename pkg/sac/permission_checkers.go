package sac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

type anyGlobalResourceAllowed struct {
	helper ForResourcesHelper
}

// NewAnyGlobalResourceAllowedPermissionChecker returns a permission checker that allows actions if the user has
// global access to at least one of the requested resources.
func NewAnyGlobalResourceAllowedPermissionChecker(targetResources ...permissions.ResourceMetadata) *anyGlobalResourceAllowed {
	subHelpers := make([]ForResourceHelper, 0, len(targetResources))
	for _, md := range targetResources {
		subHelpers = append(subHelpers, ForResource(md))
	}
	return &anyGlobalResourceAllowed{
		helper: ForResources(subHelpers...),
	}
}

func (a *anyGlobalResourceAllowed) ReadAllowed(ctx context.Context) (bool, error) {
	return a.helper.AccessAllowedToAny(ctx, storage.Access_READ_ACCESS)
}

func (a *anyGlobalResourceAllowed) WriteAllowed(ctx context.Context) (bool, error) {
	return a.helper.AccessAllowedToAny(ctx, storage.Access_READ_WRITE_ACCESS)
}

type allGlobalResourcesAllowed struct {
	helper ForResourcesHelper
}

// NewAllGlobalResourceAllowedPermissionChecker returns a permission checker that allows actions if the user has
// global access to all the requested resources.
func NewAllGlobalResourceAllowedPermissionChecker(targetResources ...permissions.ResourceMetadata) *allGlobalResourcesAllowed {
	subHelpers := make([]ForResourceHelper, 0, len(targetResources))
	for _, md := range targetResources {
		subHelpers = append(subHelpers, ForResource(md))
	}
	return &allGlobalResourcesAllowed{
		helper: ForResources(subHelpers...),
	}
}

func (a *allGlobalResourcesAllowed) ReadAllowed(ctx context.Context) (bool, error) {
	return a.helper.AccessAllowedToAll(ctx, storage.Access_READ_ACCESS)
}

func (a *allGlobalResourcesAllowed) WriteAllowed(ctx context.Context) (bool, error) {
	return a.helper.AccessAllowedToAll(ctx, storage.Access_READ_WRITE_ACCESS)
}

type notGloballyDenied struct {
	targetResource permissions.ResourceMetadata
}

// NewNotGloballyDeniedPermissionChecker returns a permission checker that allows actions if the user scope
// for the target permission is not deny-all.
func NewNotGloballyDeniedPermissionChecker(targetResource permissions.ResourceMetadata) *notGloballyDenied {
	return &notGloballyDenied{
		targetResource: targetResource,
	}
}

func (a *notGloballyDenied) ReadAllowed(ctx context.Context) (bool, error) {
	return a.accessAllowed(ctx, storage.Access_READ_ACCESS)
}

func (a *notGloballyDenied) WriteAllowed(ctx context.Context) (bool, error) {
	return a.accessAllowed(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (a *notGloballyDenied) accessAllowed(ctx context.Context, access storage.Access) (bool, error) {
	scopeChecker := ForResource(a.targetResource).ScopeChecker(ctx, access)
	eas, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
		Resource: a.targetResource,
		Access:   access,
	})
	if err != nil {
		return false, err
	}
	if eas == nil {
		return false, nil
	}
	if eas.State == effectiveaccessscope.Excluded {
		return false, nil
	}
	return true, nil
}
