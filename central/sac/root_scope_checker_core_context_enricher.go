package sac

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/sac"
	sacClient "github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	enricher *Enricher
)

func initialize() {
	enricher = newEnricher(features.ScopedAccessControl.Enabled())
}

// GetEnricher returns the singleton Enricher object.
func GetEnricher() *Enricher {
	once.Do(initialize)
	return enricher
}

// Enricher returns a object which will enrich a context with a cached root scope checker core
type Enricher struct {
	sacEnabled bool

	// In a perfect world we would clear this cache when SAC gets disabled
	cacheLock     sync.Mutex
	clientCaches  expiringcache.Cache
	clientManager AuthPluginClientManger
}

func newEnricher(sacEnabled bool) *Enricher {
	return &Enricher{
		sacEnabled:    sacEnabled,
		clientCaches:  newConfiguredCache(),
		clientManager: AuthPluginClientManagerSingleton(),
	}
}

// PreAuthContextEnricher adds the client in use at the time of request to the context for use in scope checking.
func (se *Enricher) PreAuthContextEnricher(ctx context.Context) (context.Context, error) {
	if client := se.clientManager.GetClient(); se.sacEnabled && client != nil {
		return sacClient.SetInContext(ctx, client), nil
	}
	return ctx, nil
}

// PostAuthContextEnricher enriches the given context with a root scope checker which can be used to check a
// user's permissions. If SAC is disabled we will instead enrich with an AllowAllAccessScopeChecker and skip caching
func (se *Enricher) PostAuthContextEnricher(ctx context.Context) (context.Context, error) {
	// If SAC is turned off, just allow all access for SAC checks.
	if !se.sacEnabled {
		return sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker()), nil
	}

	// Check the id of the context and decide scope checker to use.
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return sac.WithGlobalAccessScopeChecker(ctx, sac.DenyAllAccessScopeChecker()), nil
	}
	if id.Service() != nil {
		return sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker()), nil
	}
	if basic.IsBasicIdentity(id) {
		return sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker()), nil
	}

	// If no client is present, then just return a scope checker that uses local identity information.
	client := sacClient.GetFromContext(ctx)
	if client == nil {
		return sac.WithGlobalAccessScopeChecker(ctx, scopeCheckerForIdentity(id)), nil
	}

	// Get the principal and the cache key for it.
	principal, idCacheKey, err := idToPrincipalAndCacheKey(id)
	if err != nil {
		return nil, err
	}

	// If we have a scope checker cached for the user, use that, otherwise generate a new one and add it to the cache.
	cacheForClient := se.cacheForClient(client)
	rsc, _ := cacheForClient.Get(idCacheKey).(sac.ScopeCheckerCore)
	if rsc == nil {
		rsc = sac.NewRootScopeCheckerCore(NewRequestTracker(client, datastore.Singleton(), principal))
		// Not locking here can cause multiple root contexts to be created for one user.  This will have correct results
		// and be eventually consistent but it will be slightly inefficient.
		cacheForClient.Add(idCacheKey, rsc)
	}
	return sac.WithGlobalAccessScopeChecker(ctx, rsc), nil
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
	return &payload.Principal{AuthProvider: authProvider, Attributes: attributes}
}

func scopeCheckerForIdentity(id authn.Identity) sac.ScopeCheckerCore {
	var globalAccessModes []storage.Access
	switch id.Role().GlobalAccess {
	case storage.Access_READ_WRITE_ACCESS:
		globalAccessModes = append(globalAccessModes, storage.Access_READ_WRITE_ACCESS)
		fallthrough
	case storage.Access_READ_ACCESS:
		globalAccessModes = append(globalAccessModes, storage.Access_READ_ACCESS)
	}
	if len(globalAccessModes) > 0 {
		return sac.AllowFixedScopes(sac.AccessModeScopeKeys(globalAccessModes...))
	}

	var readResources []permissions.ResourceHandle
	var writeResources []permissions.ResourceHandle

	for resourceName, access := range id.Role().GetResourceToAccess() {
		resource := permissions.Resource(resourceName)
		switch access {
		case storage.Access_READ_WRITE_ACCESS:
			writeResources = append(writeResources, resource)
			fallthrough
		case storage.Access_READ_ACCESS:
			readResources = append(readResources, resource)
		}
	}

	return sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS):       sac.AllowFixedScopes(sac.ResourceScopeKeys(readResources...)),
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(sac.ResourceScopeKeys(writeResources...)),
	}
}

func newConfiguredCache() expiringcache.Cache {
	return expiringcache.NewExpiringCacheOrPanic(5000, env.PermissionTimeout.DurationSetting(), time.Minute)
}
