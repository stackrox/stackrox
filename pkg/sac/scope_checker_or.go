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

func (s orScopeChecker) SubScopeChecker(keys ...ScopeKey) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].SubScopeChecker(keys...))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
}

func (s orScopeChecker) IsAllowed(subScopeKeys ...ScopeKey) bool {
	for _, checker := range s.scopeCheckers {
		// Short-circuit on the first allowed check result.
		if checker.IsAllowed(subScopeKeys...) {
			return true
		}
	}
	return false
}

func (s orScopeChecker) AllAllowed(subScopeKeyss [][]ScopeKey) bool {
	for _, checker := range s.scopeCheckers {
		// Short-circuit on the first allowed check result.
		if checker.AllAllowed(subScopeKeyss) {
			return true
		}
	}
	return false
}

func (s orScopeChecker) ForClusterScopedObject(obj ClusterScopedObject) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].ForClusterScopedObject(obj))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
}

func (s orScopeChecker) ForNamespaceScopedObject(obj NamespaceScopedObject) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].ForNamespaceScopedObject(obj))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
}

func (s orScopeChecker) AccessMode(am storage.Access) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].AccessMode(am))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
}

func (s orScopeChecker) Resource(resource permissions.ResourceHandle) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].Resource(resource))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
}

func (s orScopeChecker) ClusterID(clusterID string) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].ClusterID(clusterID))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
}

func (s orScopeChecker) Namespace(namespace string) ScopeChecker {
	var checkers []ScopeChecker
	for i := range s.scopeCheckers {
		checkers = append(checkers, s.scopeCheckers[i].Namespace(namespace))
	}
	return &orScopeChecker{
		scopeCheckers: checkers,
	}
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

	return false, checkErrs.ErrorOrNil()
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
