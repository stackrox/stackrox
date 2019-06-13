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
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
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
	scopeMap expiringcache.Cache
}

func newEnricher() *Enricher {
	return &Enricher{scopeMap: expiringcache.NewExpiringCacheOrPanic(5000, env.PermissionTimeout.DurationSetting(), time.Minute)}
}

// RootScopeCheckerCoreContextEnricher enriches the given context with a root scope checker which can be used to check a
// user's permissions.  If SAC is disabled we will instead enrich with an AllowAllAccessScopeChecker and skip caching
func (se *Enricher) RootScopeCheckerCoreContextEnricher(ctx context.Context) (context.Context, error) {
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return sac.WithGlobalAccessScopeChecker(ctx, sac.DenyAllAccessScopeChecker()), nil
	}
	if id.Service() != nil {
		return sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker()), nil
	}

	// If SAC is configured create and cache a RootScopeCheckerCore with a Client object.  The Client object will
	// encapsulate the actual connected/disconnected state of the plugin
	clientManager := AuthPluginClientManagerSingleton()
	client := clientManager.GetClient()
	if client == nil {
		return sac.WithGlobalAccessScopeChecker(ctx, scopeCheckerForIdentity(id)), nil
	}

	principal := idToPrincipal(id)
	principalJSON, err := json.Marshal(principal)
	if err != nil {
		return nil, err
	}
	rsc := se.scopeMap.Get(principalJSON).(sac.ScopeCheckerCore)
	if rsc == nil {
		rsc = sac.NewRootScopeCheckerCore(NewRequestTracker(client, datastore.Singleton(), principal))
		// Not locking here can cause multiple root contexts to be created for one user.  This will have correct results
		// and be eventually consistent but it will be slightly inefficient.
		se.scopeMap.Add(principalJSON, rsc)
	}
	return sac.WithGlobalAccessScopeChecker(ctx, rsc), nil
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
