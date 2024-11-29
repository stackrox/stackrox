package sac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// ErrUnexpectedScopeKey is returned when scope key does not match expected level.
	ErrUnexpectedScopeKey = errors.New("unexpected scope key")
	// ErrUnknownResource is returned when resource is unknown.
	ErrUnknownResource = errors.New("unknown resource")

	log = logging.LoggerForModule()
)

type globalResourceFullAccessScopeCheckerCore struct {
	access   storage.Access
	resource permissions.Resource
	wrapped  ScopeCheckerCore
}

func (scc *globalResourceFullAccessScopeCheckerCore) SubScopeChecker(scopeKey ScopeKey) ScopeCheckerCore {
	log.Info("Resource Full access SCC for (", scc.access.String(), ",", scc.resource.GetResource(), ") drilling from global to", scopeKey.String())
	scope, ok := scopeKey.(AccessModeScopeKey)
	if !ok {
		utils.Must(errors.Wrapf(ErrUnexpectedScopeKey, "at global level checked encountered sub key %q", scopeKey))
		return DenyAllAccessScopeChecker()
	}
	subWrapped := scc.wrapped.SubScopeChecker(scopeKey)
	access := storage.Access(scope)
	if access <= scc.access {
		subScopeCheckerCore := &globalResourceFullAccessScopeCheckerCore{
			access:   scc.access,
			resource: scc.resource,
			wrapped:  subWrapped,
		}
		return &accessResourceFullAccessScopeCheckerCore{subScopeCheckerCore}
	}
	return subWrapped
}

func (scc *globalResourceFullAccessScopeCheckerCore) Allowed() bool { return false }

func (scc *globalResourceFullAccessScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return scc.
		SubScopeChecker(AccessModeScopeKey(resource.Access)).
		SubScopeChecker(ResourceScopeKey(resource.Resource.Resource)).
		EffectiveAccessScope(resource)
}

type accessResourceFullAccessScopeCheckerCore struct {
	*globalResourceFullAccessScopeCheckerCore
}

func (scc *accessResourceFullAccessScopeCheckerCore) SubScopeChecker(scopeKey ScopeKey) ScopeCheckerCore {
	log.Info("Resource Full access SCC for (", scc.access.String(), ",", scc.resource.GetResource(), ") drilling from access to", scopeKey.String())
	scope, ok := scopeKey.(ResourceScopeKey)
	if !ok {
		utils.Must(errors.Wrapf(ErrUnexpectedScopeKey, "at access level checked encountered sub key %q", scopeKey))
		return DenyAllAccessScopeChecker()
	}
	res := permissions.Resource(scope.String())
	resource, ok := resources.MetadataForResource(res)
	if !ok {
		resource, ok = resources.MetadataForInternalResource(res)
	}
	if !ok {
		utils.Must(errors.Wrapf(ErrUnknownResource, "on scope key %q", scopeKey))
		return DenyAllAccessScopeChecker()
	}
	if scc.resource.GetResource() == resource.GetResource() ||
		scc.resource.GetResource() == resource.ReplacingResource.GetResource() {
		log.Info("Resource Full access SCC for (", scc.access.String(), ",", scc.resource.GetResource(), ") - matching resource -> full access")
		return AllowAllAccessScopeChecker()
	}
	log.Info("Resource Full access SCC for (", scc.access.String(), ",", scc.resource.GetResource(), ") - no match -> defaulting to wrapped scc")
	return scc.wrapped.SubScopeChecker(scopeKey)
}

func (scc *accessResourceFullAccessScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	if scc.access < resource.Access {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	return scc.
		SubScopeChecker(ResourceScopeKey(resource.Resource.Resource)).
		EffectiveAccessScope(resource)
}

// WithUnrestrictedResourceRead returns a context that allows unrestricted read to the target resource on top of
// the current context access scopes.
func WithUnrestrictedResourceRead(ctx context.Context, resource permissions.ResourceMetadata) context.Context {
	return withUnrestrictedResourceAccess(ctx, storage.Access_READ_ACCESS, resource)
}

// WithUnrestrictedResourceReadWrite returns a context that allows unrestricted read and write to the target resource
// on top of the current context access scopes.
func WithUnrestrictedResourceReadWrite(ctx context.Context, resource permissions.ResourceMetadata) context.Context {
	return withUnrestrictedResourceAccess(ctx, storage.Access_READ_WRITE_ACCESS, resource)
}

func withUnrestrictedResourceAccess(ctx context.Context, access storage.Access, resource permissions.ResourceMetadata) context.Context {
	log.Info("withUnrestrictedResourceAccess for (", access.String(), ",", resource.String(), ")")
	wrappedScopeCheckerCore := globalAccessScopeCheckerCore(ctx)
	wrappedResourceAccessScope, err := wrappedScopeCheckerCore.EffectiveAccessScope(permissions.ResourceWithAccess{Access: access, Resource: resource})
	// If the existing access scope for the target resource is already unrestricted, don't wrap.
	if err == nil && wrappedResourceAccessScope != nil && wrappedResourceAccessScope.State == effectiveaccessscope.Included {
		log.Info("Access already unrestricted for resource", resource.String())
		return ctx
	}
	newScopeCheckerCore := &globalResourceFullAccessScopeCheckerCore{
		access:   access,
		resource: resource.GetResource(),
		wrapped:  wrappedScopeCheckerCore,
	}
	return WithGlobalAccessScopeChecker(ctx, newScopeCheckerCore)
}
