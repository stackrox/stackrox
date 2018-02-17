package user

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	logger = logging.LoggerForModule()
)

// AuthLister contains the storage-access functions that this
// interceptor requires.
type AuthLister interface {
	GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error)
}

// An AuthInterceptor provides gRPC interceptors that authenticates users.
type AuthInterceptor struct {
	db        AuthLister
	providers map[string]authproviders.Authenticator
	lock      sync.RWMutex
}

// NewAuthInterceptor creates a new AuthInterceptor.
func NewAuthInterceptor(storage AuthLister) *AuthInterceptor {
	return &AuthInterceptor{
		db:        storage,
		providers: make(map[string]authproviders.Authenticator),
	}
}

// UpdateProvider updates the in-memory set of auth providers.
func (a *AuthInterceptor) UpdateProvider(id string, provider authproviders.Authenticator) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.providers[id] = provider
}

// RemoveProvider removes a provider from the set of auth providers.
func (a *AuthInterceptor) RemoveProvider(id string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	delete(a.providers, id)
}

// UnaryInterceptor parses authentication metadata to maintain the time for
// a cluster's sensor has last contacted this API server.
// Naturally, it should be called after authentication metadata is parsed.
func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return a.authUnary
}

// StreamInterceptor parses authentication metadata to maintain the time for
// a cluster's sensor has last contacted this API server.
// Naturally, it should be called after authentication metadata is parsed.
func (a *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return a.authStream
}

// HTTPInterceptor is an interceptor for http handlers
func (a *AuthInterceptor) HTTPInterceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Lowercase all of the header keys due to grpc metadata keys being lowercased
		newHeaders := make(map[string][]string)
		for k, v := range req.Header {
			newHeaders[strings.ToLower(k)] = v
		}
		ctx := a.retrieveToken(req.Context(), newHeaders)
		h.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (a *AuthInterceptor) authUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(a.authToken(ctx), req)
}

func (a *AuthInterceptor) authStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	newStream := &authn.StreamWithContext{
		ServerStream:    stream,
		ContextOverride: a.authToken(stream.Context()),
	}
	return handler(srv, newStream)
}

func (a *AuthInterceptor) retrieveToken(ctx context.Context, headers map[string][]string) (newCtx context.Context) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	newCtx = authn.NewAuthConfigurationContext(ctx, authn.AuthConfiguration{
		ProviderConfigured: a.countEnabled() > 0,
	})
	for _, p := range a.providers {
		if !p.Enabled() {
			continue
		}
		user, expiration, err := p.User(headers)
		if err != nil {
			logger.Debugf("User auth error: %s", err)
			continue
		}

		return authn.NewUserContext(newCtx, authn.UserIdentity{
			User:         user,
			AuthProvider: p,
			Expiration:   expiration,
		})
	}
	return newCtx
}

func (a *AuthInterceptor) authToken(ctx context.Context) context.Context {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return a.retrieveToken(ctx, meta)
}

func (a *AuthInterceptor) countEnabled() (enabled int) {
	for _, p := range a.providers {
		if p.Enabled() {
			enabled++
		}
	}
	return enabled
}
