package sac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/utils"
)

// ForResourceHelper is a helper for querying access scopes related to a resource.
type ForResourceHelper struct {
	resourceMD permissions.ResourceMetadata
}

// ForResource returns a helper for querying access scopes related to the given resource.
func ForResource(resourceMD permissions.ResourceMetadata) ForResourceHelper {
	return ForResourceHelper{
		resourceMD: resourceMD,
	}
}

// ScopeChecker returns the scope checker for accessing the given resource in the specified way.
func (h ForResourceHelper) ScopeChecker(ctx context.Context, am storage.Access, keys ...ScopeKey) ScopeChecker {
	resourceScopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(am).Resource(
		h.resourceMD).SubScopeChecker(keys...)

	if h.resourceMD.GetReplacingResource() == nil {
		return resourceScopeChecker
	}
	// Conditionally create a OR scope checker if a replacing resource is given. This way we check access to either
	// the old resource OR the replacing resource, keeping backwards-compatibility.
	return NewOrScopeChecker(
		resourceScopeChecker,
		GlobalAccessScopeChecker(ctx).AccessMode(am).
			Resource(h.resourceMD.ReplacingResource).SubScopeChecker(keys...))
}

// AccessAllowed checks if in the given context, we have access of the specified kind to the resource or
// a subscope thereof.
func (h ForResourceHelper) AccessAllowed(ctx context.Context, am storage.Access, keys ...ScopeKey) (bool, error) {
	return h.ScopeChecker(ctx, am, keys...).IsAllowed(), nil
}

// ReadAllowed checks if in the given context, we have read access to the resource or a subscope thereof.
func (h ForResourceHelper) ReadAllowed(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowed(ctx, storage.Access_READ_ACCESS, keys...)
}

// WriteAllowed checks if in the given context, we have write access to the resource or a subscope thereof.
func (h ForResourceHelper) WriteAllowed(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowed(ctx, storage.Access_READ_WRITE_ACCESS, keys...)
}

// MustCreatePgSearchHelper creates and returns a search helper with the given options, or panics if the
// search helper could not be created.
func (h ForResourceHelper) MustCreatePgSearchHelper() SearchHelper {
	searchHelper, err := NewPgSearchHelper(h.resourceMD, h.ScopeChecker)
	utils.CrashOnError(err)
	return searchHelper
}
