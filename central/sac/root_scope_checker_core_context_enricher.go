package sac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/userpass"
	"github.com/stackrox/rox/central/sac/authorizer"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	enricher *Enricher
)

func initialize() {
	enricher = &Enricher{}
}

// GetEnricher returns the singleton Enricher object.
func GetEnricher() *Enricher {
	once.Do(initialize)
	return enricher
}

// Enricher returns a object which will enrich a context with a cached root scope checker core
type Enricher struct {
}

// getRootScopeCheckerCore adds a scope checker to the context for later use in
// scope checking.
func (se *Enricher) getRootScopeCheckerCore(ctx context.Context) (context.Context, observe.ScopeCheckerCoreType, error) {
	// Check the id of the context and decide scope checker to use.
	id := authn.IdentityFromContextOrNil(ctx)
	if id == nil {
		// User identity not found => deny all.
		return sac.WithGlobalAccessScopeChecker(ctx, sac.DenyAllAccessScopeChecker()), observe.ScopeCheckerDenyForNoID, nil
	}
	if id.Service() != nil || userpass.IsLocalAdmin(id) {
		// Admin => allow all.
		return sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker()), observe.ScopeCheckerAllowAdminAndService, nil
	}
	if len(id.Roles()) == 0 {
		// User has no valid role => deny all.
		return sac.WithGlobalAccessScopeChecker(ctx, sac.DenyAllAccessScopeChecker()), observe.ScopeCheckerDenyForNoID, nil
	}
	scopeChecker, err := authorizer.NewBuiltInScopeChecker(ctx, id.Roles())
	if err != nil {
		return nil, observe.ScopeCheckerNone, errors.Wrap(err, "creating scoped authorizer for identity")
	}
	return sac.WithGlobalAccessScopeChecker(ctx, scopeChecker), observe.ScopeCheckerBuiltIn, nil
}

// GetPreAuthContextEnricher returns a contextutil.ContextUpdater which adds a
// scope checker to the context for later use in scope checking. It also enables
// authorization tracing on demand by injecting an instance of a specific struct
// into the context.
func (se *Enricher) GetPreAuthContextEnricher(authzTraceSink observe.AuthzTraceSink) contextutil.ContextUpdater {
	return func(ctx context.Context) (context.Context, error) {
		// Collect authz trace iff it is turned on globally. An alternative
		// could be per-request collection triggered by a specific request
		// header and the `DebugLogs` permission of the associated principal.
		var trace *observe.AuthzTrace
		if authzTraceSink.IsActive() {
			trace = observe.NewAuthzTrace()
			ctx = observe.ContextWithAuthzTrace(ctx, trace)
		}

		ctxWithSCC, sccType, err := se.getRootScopeCheckerCore(ctx)
		if err != nil {
			return nil, err
		}
		trace.RecordScopeCheckerCoreType(sccType)
		return ctxWithSCC, nil
	}
}
