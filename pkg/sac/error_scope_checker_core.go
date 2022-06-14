package sac

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

type errScopeCheckerCore struct {
	err error
}

func (s errScopeCheckerCore) SubScopeChecker(key ScopeKey) ScopeCheckerCore {
	return s
}

func (s errScopeCheckerCore) TryAllowed() TryAllowedResult {
	// Return `Unknown` to indicate to the caller that `PerformChecks` must be called,
	// which will yield an error.
	return Unknown
}

func (s errScopeCheckerCore) PerformChecks(ctx context.Context) error {
	return s.err
}

// ErrorAccessScopeCheckerCore returns an access scope checker that always returns an error.
func ErrorAccessScopeCheckerCore(err error) ScopeCheckerCore {
	return errScopeCheckerCore{
		err: err,
	}
}

func (s errScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return effectiveaccessscope.DenyAllEffectiveAccessScope(), s.err
}
