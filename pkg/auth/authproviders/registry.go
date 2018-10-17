package authproviders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// authProviderTokenTTL is the cap for all auth tokens issued for external auth providers.
	authProviderTokenTTL = 30 * 24 * time.Hour
)

var (
	log = logging.LoggerForModule()
)

// NewStoreBackedRegistry creates a new auth provider registry that is backed by a store. It also can handle HTTP requests,
// where every incoming HTTP request URL is expected to refer to a path under `urlPathPrefix`. The redirect URL for
// clients upon successful/failed authentication is `clientRedirectURL`.
func NewStoreBackedRegistry(urlPathPrefix string, redirectURL string, store Store, tokenIssuerFactory tokens.IssuerFactory, defaultRoleMapper permissions.RoleMapper) (Registry, error) {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	registry := &storeBackedRegistry{
		urlPathPrefix: urlPathPrefix,
		redirectURL:   redirectURL,
		store:         store,
		issuerFactory: tokenIssuerFactory,

		backendFactories: make(map[string]BackendFactory),
		providers:        make(map[string]*authProvider),
		providersByName:  make(map[string]*authProvider),

		defaultRoleMapper: defaultRoleMapper,
	}

	if err := registry.init(); err != nil {
		return nil, err
	}
	return registry, nil
}

type storeBackedRegistry struct {
	urlPathPrefix string
	redirectURL   string

	store         Store
	issuerFactory tokens.IssuerFactory

	backendFactories map[string]BackendFactory
	providers        map[string]*authProvider
	providersByName  map[string]*authProvider
	mutex            sync.RWMutex

	defaultRoleMapper permissions.RoleMapper
}

// createAuthProviderAsync is called asynchronously when a new auth provider factory is registered, and providers have
// already been read from the database but couldn't be instantiated due to the factory missing.
func createAuthProviderAsync(factory BackendFactory, typ string, provider *authProvider) {
	backend, effectiveConfig, err := factory.CreateAuthProviderBackend(context.Background(), provider.ID(), provider.baseInfo.UiEndpoint, provider.baseInfo.Config)
	if err != nil {
		log.Errorf("Failed to create auth provider of type %s: %v", typ, err)
		return
	}
	provider.setBackend(backend, effectiveConfig)
}

func (r *storeBackedRegistry) RegisterBackendFactory(typ string, factoryCreator BackendFactoryCreator) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.backendFactories[typ] != nil {
		return fmt.Errorf("backend factory for type %s is already registered", typ)
	}
	pathPrefix := fmt.Sprintf("%s%s/", r.urlPathPrefix, typ)
	factory := factoryCreator(pathPrefix)
	if factory == nil {
		return errors.New("factory creator returned nil factory")
	}
	r.backendFactories[typ] = factory

	for _, provider := range r.providers {
		if provider.Type() != typ || provider.Backend() != nil {
			continue
		}
		go createAuthProviderAsync(factory, typ, provider)
	}

	return nil
}

func (r *storeBackedRegistry) validateNameNoLock(name string) error {
	if name == "" {
		return errors.New("name must not be empty")
	}
	if r.providersByName[name] != nil {
		return fmt.Errorf("name %q is already taken", name)
	}
	return nil
}

func (r *storeBackedRegistry) init() error {
	providerDefs, err := r.store.GetAllAuthProviders()
	if err != nil {
		return err
	}

	r.providers = make(map[string]*authProvider, len(providerDefs))
	for _, def := range providerDefs {
		provider := r.createFromStoredDef(context.Background(), def)
		r.registerProvider(provider)
	}
	return nil
}

func (r *storeBackedRegistry) createFromStoredDef(ctx context.Context, def *v1.AuthProvider) *authProvider {
	provider, err := r.createProvider(ctx, def.GetId(), def.GetType(), def.GetName(), def.GetUiEndpoint(), def.GetEnabled(), def.GetConfig())
	if err != nil {
		log.Errorf("Could not instantiate auth provider for stored configuration: %v", err)
		return &authProvider{
			baseInfo:   *def,
			registry:   r,
			roleMapper: r.defaultRoleMapper,
		}
	}

	return provider
}

func (r *storeBackedRegistry) getFactory(typ string) BackendFactory {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.backendFactories[typ]
}

func (r *storeBackedRegistry) createProvider(ctx context.Context, id, typ, name, uiEndpoint string, enabled bool, config map[string]string) (*authProvider, error) {
	factory := r.getFactory(typ)
	if factory == nil {
		return nil, fmt.Errorf("unknown auth provider type %s", typ)
	}
	backend, effectiveConfig, err := factory.CreateAuthProviderBackend(ctx, id, uiEndpoint, config)
	if err != nil {
		return nil, err
	}
	provider := &authProvider{
		backend: backend,
		baseInfo: v1.AuthProvider{
			Id:         id,
			Name:       name,
			Type:       typ,
			UiEndpoint: uiEndpoint,
			Enabled:    enabled,
			Config:     effectiveConfig,
		},
		registry:   r,
		roleMapper: r.defaultRoleMapper,
	}
	return provider, nil
}

func (r *storeBackedRegistry) addProvider(provider *authProvider) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.providers[provider.ID()] = provider
}

func (r *storeBackedRegistry) registerProvider(provider *authProvider) {
	r.addProvider(provider)

	issuer, err := r.issuerFactory.CreateIssuer(provider, tokens.WithTTL(authProviderTokenTTL))
	if err != nil {
		log.Errorf("UNEXPECTED: failed to create issuer for newly created auth provider: %v", err)
	}
	provider.issuer = issuer
}

func (r *storeBackedRegistry) CreateAuthProvider(ctx context.Context, typ, name, uiEndpoint string, enabled bool, config map[string]string) (AuthProvider, error) {
	id := uuid.NewV4().String()
	newProvider, err := r.createProvider(ctx, id, typ, name, uiEndpoint, enabled, config)
	if err != nil {
		return nil, err
	}

	if err := r.store.AddAuthProvider(&newProvider.baseInfo); err != nil {
		return nil, err
	}

	r.registerProvider(newProvider)

	return newProvider, nil
}

func (r *storeBackedRegistry) UpdateAuthProvider(ctx context.Context, id string, name *string, enabled *bool) (AuthProvider, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	provider := r.providers[id]
	if provider == nil {
		return nil, fmt.Errorf("provider with ID %s not found", id)
	}

	if name != nil {
		if err := r.validateNameNoLock(*name); err != nil {
			return nil, fmt.Errorf("invalid name %q", *name)
		}
	}

	modified, baseInfo, oldName := provider.update(name, enabled)
	if !modified {
		return provider, nil
	}

	if oldName != "" {
		delete(r.providersByName, oldName)
		r.providersByName[baseInfo.Name] = provider
	}

	if err := r.store.UpdateAuthProvider(&baseInfo); err != nil {
		return nil, err
	}

	return provider, nil
}

func (r *storeBackedRegistry) getAuthProvider(id string) *authProvider {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.providers[id]
}

func (r *storeBackedRegistry) GetAuthProvider(ctx context.Context, id string) AuthProvider {
	return r.getAuthProvider(id)
}

func (r *storeBackedRegistry) GetAuthProviders(ctx context.Context, name, typ *string) []AuthProvider {
	var result []AuthProvider

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, provider := range r.providers {
		if typ != nil && *typ != provider.Type() {
			continue
		}
		if name != nil && provider.Name() != *name {
			continue
		}
		result = append(result, provider)
	}

	return result
}

func (r *storeBackedRegistry) DeleteAuthProvider(ctx context.Context, id string) error {
	if err := r.store.RemoveAuthProvider(id); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.providers, id)
	return nil
}

func (r *storeBackedRegistry) recordSuccess(id string) error {
	return r.store.RecordAuthSuccess(id)
}

func (r *storeBackedRegistry) HasUsableProviders() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, provider := range r.providers {
		if provider.Enabled() && provider.Validated() {
			return true
		}
	}
	return false
}

func (r *storeBackedRegistry) ExchangeToken(ctx context.Context, externalToken, typ, state string) (string, string, error) {
	factory := r.getFactory(typ)
	if factory == nil {
		return "", "", status.Errorf(codes.InvalidArgument, "invalid auth provider type %q", typ)
	}

	providerID, err := factory.ResolveProvider(state)
	if err != nil {
		return "", "", err
	}
	provider := r.getAuthProvider(providerID)
	if provider == nil {
		return "", "", status.Errorf(codes.NotFound, "could not locate auth provider %q", providerID)
	}

	claim, opts, clientState, err := provider.Backend().ExchangeToken(ctx, externalToken, state)
	if err != nil {
		return "", "", err
	}
	token, err := provider.issuer.Issue(tokens.RoxClaims{ExternalUser: claim}, opts...)
	if err != nil {
		return "", "", err
	}
	return token.Token, clientState, nil
}
