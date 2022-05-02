package sac

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

type orScopeChecker struct {
	scopeCheckers []ScopeChecker
}

// NewOrScopeChecker returns a new scope checker wrapping the given scope checkers,
// each result will be combined in a disjunction (OR).
func NewOrScopeChecker(scopeCheckers ...ScopeChecker) ScopeChecker {
	return orScopeChecker{
		scopeCheckers: scopeCheckers,
	}
}

func (s orScopeChecker) Core() ScopeCheckerCore {
	if len(s.scopeCheckers) != 0 {
		return s.scopeCheckers[0].Core()
	}
	return nil
}

func (s orScopeChecker) SubScopeChecker(keys ...ScopeKey) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].SubScopeChecker(keys...)
	}
	return s
}

func (s orScopeChecker) PerformChecks(ctx context.Context) error {
	var performChecksErrs *multierror.Error
	for _, checker := range s.scopeCheckers {
		err := checker.PerformChecks(ctx)
		// Short-circuit on the first non-error perform check.
		if err != nil {
			performChecksErrs = multierror.Append(performChecksErrs, err)
		}
	}
	if performChecksErrs == nil {
		return nil
	}
	return performChecksErrs
}

func (s orScopeChecker) TryAllowed(subScopeKeys ...ScopeKey) TryAllowedResult {
	result := Deny
	for _, checker := range s.scopeCheckers {
		if res := checker.TryAllowed(subScopeKeys...); res == Allow {
			return res
		} else if res == Unknown {
			result = Unknown
		}
	}
	return result
}

func (s orScopeChecker) Allowed(ctx context.Context, subScopeKeys ...ScopeKey) (bool, error) {
	var allowedErrs *multierror.Error
	for _, checker := range s.scopeCheckers {
		allowed, err := checker.Allowed(ctx, subScopeKeys...)
		// Short-circuit on the first allowed check result.
		if err != nil {
			allowedErrs = multierror.Append(allowedErrs, err)
		} else if allowed {
			return allowed, nil
		}
	}
	if allowedErrs != nil {
		return false, allowedErrs
	}
	return false, nil
}

func (s orScopeChecker) TryAnyAllowed(subScopeKeyss [][]ScopeKey) TryAllowedResult {
	result := Deny
	for _, checker := range s.scopeCheckers {
		if res := checker.TryAnyAllowed(subScopeKeyss); res == Allow {
			return res
		} else if res == Unknown {
			result = Unknown
		}
	}
	return result
}

func (s orScopeChecker) AnyAllowed(ctx context.Context, subScopeKeyss [][]ScopeKey) (bool, error) {
	var anyAllowedErrs *multierror.Error
	for _, checker := range s.scopeCheckers {
		allowed, err := checker.AnyAllowed(ctx, subScopeKeyss)
		// Short-circuit on the first allowed check result.
		if err != nil {
			anyAllowedErrs = multierror.Append(anyAllowedErrs, err)
		} else if allowed {
			return allowed, nil
		}
	}
	if anyAllowedErrs != nil {
		return false, anyAllowedErrs
	}
	return false, nil
}

func (s orScopeChecker) TryAllAllowed(subScopeKeyss [][]ScopeKey) TryAllowedResult {
	result := Deny
	for _, checker := range s.scopeCheckers {
		if res := checker.TryAllAllowed(subScopeKeyss); res == Allow {
			return res
		} else if res == Unknown {
			result = Unknown
		}
	}
	return result
}

func (s orScopeChecker) AllAllowed(ctx context.Context, subScopeKeyss [][]ScopeKey) (bool, error) {
	var allAllowedErrs *multierror.Error
	for _, checker := range s.scopeCheckers {
		allowed, err := checker.AllAllowed(ctx, subScopeKeyss)
		// Short-circuit on the first allowed check result.
		if err != nil {
			allAllowedErrs = multierror.Append(allAllowedErrs, err)
		} else if allowed {
			return allowed, nil
		}
	}
	if allAllowedErrs != nil {
		return false, allAllowedErrs
	}
	return false, nil
}

func (s orScopeChecker) ForClusterScopedObject(obj ClusterScopedObject) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].ForClusterScopedObject(obj)
	}
	return s
}

func (s orScopeChecker) ForNamespaceScopedObject(obj NamespaceScopedObject) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].ForNamespaceScopedObject(obj)
	}
	return s
}

func (s orScopeChecker) AccessMode(am storage.Access) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].AccessMode(am)
	}
	return s
}

func (s orScopeChecker) Resource(resource permissions.ResourceHandle) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].Resource(resource)
	}
	return s
}

func (s orScopeChecker) ClusterID(clusterID string) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].ClusterID(clusterID)
	}
	return s
}

func (s orScopeChecker) Namespace(namespace string) ScopeChecker {
	for i := range s.scopeCheckers {
		s.scopeCheckers[i] = s.scopeCheckers[i].Namespace(namespace)
	}
	return s
}

func (s orScopeChecker) Check(ctx context.Context, pred ScopePredicate) (bool, error) {
	var checkErrs *multierror.Error
	for _, checker := range s.scopeCheckers {
		allowed, err := checker.Check(ctx, pred)
		if err != nil {
			checkErrs = multierror.Append(checkErrs, err)
		} else if allowed {
			return allowed, nil
		}
	}

	return false, checkErrs
}

func (s orScopeChecker) EffectiveAccessScope(
	resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	var effectiveAccessScopeErrs *multierror.Error
	var root = effectiveaccessscope.DenyAllEffectiveAccessScope()
	for _, checker := range s.scopeCheckers {
		eas, err := checker.EffectiveAccessScope(resource)
		if err != nil {
			effectiveAccessScopeErrs = multierror.Append(effectiveAccessScopeErrs, err)
		} else if eas != nil {
			root.Merge(eas)
		}
	}
	if effectiveAccessScopeErrs != nil {
		return nil, effectiveAccessScopeErrs
	}
	return root, nil
}
