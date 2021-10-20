package sac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// ForResourcesHelper is a helper for querying access scopes related to multiple resources.
type ForResourcesHelper struct {
	resourceHelpers []ForResourceHelper
}

// ForResources returns a helper for querying access scopes related to multiple resources.
func ForResources(resourceHelper ...ForResourceHelper) ForResourcesHelper {
	return ForResourcesHelper{
		resourceHelpers: resourceHelper,
	}
}

// AccessAllowedToAny checks if in the given context, we have access of the specified kind to _any_ of the resources or
// subscopes thereof.
func (h ForResourcesHelper) AccessAllowedToAny(ctx context.Context, am storage.Access, keys ...ScopeKey) (bool, error) {
	for _, rh := range h.resourceHelpers {
		if ok, err := rh.AccessAllowed(ctx, am, keys...); err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

// ReadAllowedToAny checks if in the given context, we have read access to _any_ of the resources or subscopes thereof.
func (h ForResourcesHelper) ReadAllowedToAny(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowedToAny(ctx, storage.Access_READ_ACCESS, keys...)
}

// WriteAllowedToAny checks if in the given context, we have write access to _any_ of the resources or subscopes thereof.
func (h ForResourcesHelper) WriteAllowedToAny(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowedToAny(ctx, storage.Access_READ_WRITE_ACCESS, keys...)
}

// AccessAllowedToAll checks if in the given context, we have access of the specified kind to _all_ of the resources or
// subscopes thereof.
func (h ForResourcesHelper) AccessAllowedToAll(ctx context.Context, am storage.Access, keys ...ScopeKey) (bool, error) {
	for _, rh := range h.resourceHelpers {
		if ok, err := rh.AccessAllowed(ctx, am, keys...); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

// ReadAllowedToAll checks if in the given context, we have read access to _all_ of the resources or subscopes thereof.
func (h ForResourcesHelper) ReadAllowedToAll(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowedToAll(ctx, storage.Access_READ_ACCESS, keys...)
}

// WriteAllowedToAll checks if in the given context, we have write access to _all_ of the resources or subscopes thereof.
func (h ForResourcesHelper) WriteAllowedToAll(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowedToAll(ctx, storage.Access_READ_WRITE_ACCESS, keys...)
}
