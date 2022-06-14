package sac

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/stackrox/central/auth/userpass"
	"github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/sac/authorizer"
	"github.com/stackrox/stackrox/pkg/auth/permissions/utils"
	"github.com/stackrox/stackrox/pkg/contextutil"
	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/expiringcache"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
	"github.com/stackrox/stackrox/pkg/sac"
	sacClient "github.com/stackrox/stackrox/pkg/sac/client"
	"github.com/stackrox/stackrox/pkg/sac/observe"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once     sync.Once
	enricher *Enricher
)

func initialize() {
	enricher = newEnricher()
}

// GetEnricher returns the singleton Enricher object.
func GetEnricher() *Enricher {
	once.Do(initialize)
	return enricher
}

// Enricher returns a object which will enrich a context with a cached root scope checker core
type Enricher struct {
	// In a perfect world we would clear this cache when SAC gets disabled
	cacheLock     sync.Mutex
	clientCaches  expiringcache.Cache
	clientManager AuthPluginClientManger
}

func newEnricher() *Enricher {
	return &Enricher{
		clientCaches:  newConfiguredCache(),
		clientManager: AuthPluginClientManagerSingleton(),
	}
}

// getRootScopeCheckerCore adds a scope checker to the context for later use in
// scope checking. There are four possibilities currently:
//   1. User identity maps to a special case: (a) deny all or (b) allow all =>
//      add the corresponding trivial scope checker.
//   2. Built-in scoped authorizer must be used => use the scope checker
//      constructed from resolved roles associated with the identity.
//   3. Auth plugin is detected => use the scope checker created around the
//      auth plugin client in use at the time of request.
//   4. Unrecoverable error => nil context.
func (se *Enricher) getRootScopeCheckerCore(ctx context.Context) (context.Context, observe.ScopeCheckerCoreType, error) {
	client := se.clientManager.GetClient()

	// Check the id of the context and decide scope checker to use.
	id := authn.IdentityFromContextOrNil(ctx)
	if id == nil {
		// 1a. User identity not found => deny all.
		return sac.WithGlobalAccessScopeChecker(ctx, sac.DenyAllAccessScopeChecker()), observe.ScopeCheckerDenyForNoID, nil
	}
	if id.Service() != nil || userpass.IsLocalAdmin(id) {
		// 1b. Admin => allow all.
		return sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker()), observe.ScopeCheckerAllowAdminAndService, nil
	}
	if len(id.Roles()) == 0 {
		// 1c. User has no valid role => deny all.
		return sac.WithGlobalAccessScopeChecker(ctx, sac.DenyAllAccessScopeChecker()), observe.ScopeCheckerDenyForNoID, nil
	}
	if client == nil {
		// 2. Built-in scoped authorizer must be used.
		scopeChecker, err := authorizer.NewBuiltInScopeChecker(ctx, id.Roles())
		if err != nil {
			return nil, observe.ScopeCheckerNone, errors.Wrap(err, "creating scoped authorizer for identity")
		}
		return sac.WithGlobalAccessScopeChecker(ctx, scopeChecker), observe.ScopeCheckerBuiltIn, nil
	}
	ctx = sac.SetContextPluginScopedAuthzEnabled(ctx)

	// Get the principal and the cache key for it.
	principal, idCacheKey, err := idToPrincipalAndCacheKey(id)
	if err != nil {
		return nil, observe.ScopeCheckerNone, err
	}

	// 3. If we have a scope checker cached for the user, use that,
	// otherwise generate a new one and add it to the cache.
	cacheForClient := se.cacheForClient(client)
	rsc, _ := cacheForClient.Get(idCacheKey).(sac.ScopeCheckerCore)
	if rsc == nil {
		rsc = sac.NewRootScopeCheckerCore(NewRequestTracker(client, datastore.Singleton(), principal))
		// Not locking here can cause multiple root contexts to be created for
		// one user. This will have correct results and be eventually consistent
		// but it will be slightly inefficient.
		cacheForClient.Add(idCacheKey, rsc)
	}
	return sac.WithGlobalAccessScopeChecker(ctx, rsc), observe.ScopeCheckerPlugin, nil
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

func (se *Enricher) cacheForClient(client sacClient.Client) expiringcache.Cache {
	se.cacheLock.Lock()
	defer se.cacheLock.Unlock()

	clientCache, _ := se.clientCaches.Get(client).(expiringcache.Cache)
	if clientCache == nil {
		clientCache = newConfiguredCache()
		se.clientCaches.Add(client, clientCache)
	}
	return clientCache
}

func idToPrincipalAndCacheKey(id authn.Identity) (*payload.Principal, string, error) {
	// Generate a JSON body for the user we are using the auth plugin for.
	principal := idToPrincipal(id)
	principalJSONBytes, err := json.Marshal(principal)
	if err != nil {
		return nil, "", err
	}
	return principal, string(principalJSONBytes), nil
}

func idToPrincipal(id authn.Identity) *payload.Principal {
	externalAuthProvider := id.ExternalAuthProvider()
	var authProvider payload.AuthProviderInfo
	// TODO joseph do something here for, e.g., API tokens
	if externalAuthProvider == nil {
		authProvider = payload.AuthProviderInfo{
			Type: "",
			ID:   "",
			Name: "",
		}
	} else {
		authProvider = payload.AuthProviderInfo{
			Type: externalAuthProvider.Type(),
			ID:   externalAuthProvider.ID(),
			Name: externalAuthProvider.Name(),
		}
	}
	attributes := make(map[string]interface{}, len(id.Attributes()))
	for k, v := range id.Attributes() {
		attributes[k] = v
	}

	return &payload.Principal{AuthProvider: authProvider, Attributes: attributes, Roles: utils.RoleNames(id.Roles())}
}

func newConfiguredCache() expiringcache.Cache {
	return expiringcache.NewExpiringCache(env.PermissionTimeout.DurationSetting())
}
