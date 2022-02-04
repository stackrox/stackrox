package sac

import (
	"context"

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

func (s errScopeCheckerCore) NeedsPostFiltering() bool {
	// The sac flow when no post filtering is needed should rely on the
	// EffectiveAccessScope function, which will surface the error.
	return false
}

// ErrorAccessScopeCheckerCore returns an access scope checker that always returns an error.
func ErrorAccessScopeCheckerCore(err error) ScopeCheckerCore {
	return errScopeCheckerCore{
		err: err,
	}
}

func (s errScopeCheckerCore) EffectiveAccessScope(_ context.Context) (*effectiveaccessscope.ScopeTree, error) {
	return effectiveaccessscope.RestrictedEffectiveAccessScope(), s.err
}
