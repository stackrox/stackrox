package transitional

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	resources2 "github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sync"
)

type permissionUseRecorder struct {
	perms permissions.PermissionMap
	mutex sync.Mutex
}

func newPermissionUseRecorder() *permissionUseRecorder {
	return &permissionUseRecorder{
		perms: make(permissions.PermissionMap),
	}
}

func (r *permissionUseRecorder) RecordPermissionUse(resource permissions.Resource, am storage.Access) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.perms.Add(resource, am)
}

func (r *permissionUseRecorder) UsedPermissions() permissions.PermissionMap {
	result := make(permissions.PermissionMap)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	for resource, am := range r.perms {
		result[resource] = am
	}
	return result
}

type permissionRecordingSCC struct {
	wrapped sac.ScopeCheckerCore

	rec *permissionUseRecorder

	am  *storage.Access
	res *permissions.Resource
}

func newPermissionRecordingSCC(wrapped sac.ScopeCheckerCore) *permissionRecordingSCC {
	return &permissionRecordingSCC{
		wrapped: wrapped,
		rec:     newPermissionUseRecorder(),
	}
}

func (s *permissionRecordingSCC) SubScopeChecker(key sac.ScopeKey) sac.ScopeCheckerCore {
	subScopeChecker := &permissionRecordingSCC{
		wrapped: s.wrapped.SubScopeChecker(key),
		rec:     s.rec,
		am:      s.am,
		res:     s.res,
	}

	if s.am == nil { // global level
		if k, ok := key.(sac.AccessModeScopeKey); ok {
			subScopeChecker.am = &[]storage.Access{storage.Access(k)}[0]
		} else {
			return sac.DenyAllAccessScopeChecker()
		}
	} else if s.res == nil { // access mode-level
		if k, ok := key.(sac.ResourceScopeKey); ok {
			subScopeChecker.res = &[]permissions.Resource{permissions.Resource(k)}[0]
		} else {
			return sac.DenyAllAccessScopeChecker()
		}
	}

	return subScopeChecker
}

func (s *permissionRecordingSCC) TryAllowed() sac.TryAllowedResult {
	res := s.wrapped.TryAllowed()
	if res == sac.Unknown {
		return sac.Unknown
	}

	am := storage.Access_READ_WRITE_ACCESS
	if s.am != nil {
		am = *s.am
	}
	resources := resources2.ListAll()
	if s.res != nil {
		resources = []permissions.Resource{*s.res}
	}

	for _, resource := range resources {
		s.rec.RecordPermissionUse(resource, am)
	}

	return res
}

func (s *permissionRecordingSCC) PerformChecks(ctx context.Context) error {
	return s.wrapped.PerformChecks(ctx)
}

func (s *permissionRecordingSCC) UsedPermissions() permissions.PermissionMap {
	return s.rec.UsedPermissions()
}

func (s *permissionRecordingSCC) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return s.wrapped.EffectiveAccessScope(resource)
}
