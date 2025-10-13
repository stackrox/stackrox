package testutils

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/sac"
	"go.uber.org/mock/gomock"
)

type contextWithAccessMatcher struct {
	pred sac.ScopePredicate
}

func (m contextWithAccessMatcher) Matches(x interface{}) bool {
	ctx, _ := x.(context.Context)
	if ctx == nil {
		return false
	}
	ok, _ := sac.GlobalAccessScopeChecker(ctx).Check(ctx, m.pred)
	return ok
}

func (m contextWithAccessMatcher) String() string {
	return fmt.Sprintf("context satisfying access scope predicate %v", m.pred)
}

// ContextWithAccess returns a matcher that matches contexts with access scopes satisfying the given predicate.
func ContextWithAccess(pred sac.ScopePredicate) gomock.Matcher {
	return contextWithAccessMatcher{
		pred: pred,
	}
}
