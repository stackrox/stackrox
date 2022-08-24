package sac

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/utils"
)

type errScopeCheckerCore struct {
	err error
}

func (s errScopeCheckerCore) SubScopeChecker(_ ScopeKey) ScopeCheckerCore {
	return s
}

func (s errScopeCheckerCore) TryAllowed() TryAllowedResult {
	logging.LoggerForModule().Error("This should not be called", s.err)
	utils.Must(s.err)
	return Deny
}

// ErrorAccessScopeCheckerCore returns an access scope checker that always returns an error.
func ErrorAccessScopeCheckerCore(err error) ScopeCheckerCore {
	return errScopeCheckerCore{
		err: err,
	}
}

func (s errScopeCheckerCore) EffectiveAccessScope(_ permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return effectiveaccessscope.DenyAllEffectiveAccessScope(), s.err
}
