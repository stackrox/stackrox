package user

import (
	"context"
	"net/http"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	logger = logging.LoggerForModule()
)

// AuthenticatorProvider gives us access to authenticators.
type AuthenticatorProvider interface {
	GetAuthenticators() map[string]authproviders.Authenticator
	RecordAuthSuccess(id string) error
}

// An AuthInterceptor provides gRPC interceptors that authenticates users.
type AuthInterceptor struct {
	authenticatorProvider AuthenticatorProvider
}

// NewAuthInterceptor creates a new AuthInterceptor.
func NewAuthInterceptor(authenticatorProvider AuthenticatorProvider) *AuthInterceptor {
	return &AuthInterceptor{authenticatorProvider: authenticatorProvider}
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

func (a *AuthInterceptor) authToken(ctx context.Context) context.Context {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return a.retrieveToken(ctx, meta)
}

func (a *AuthInterceptor) retrieveToken(ctx context.Context, headers map[string][]string) (newCtx context.Context) {
	authenticators := a.authenticatorProvider.GetAuthenticators()
	userIdentity := a.getUserIdentity(headers, authenticators)
	newCtx = authn.NewAuthConfigurationContext(ctx, authn.AuthConfiguration{
		ProviderConfigured: a.countEnabledAndValidated(authenticators) > 0,
	})

	if userIdentity != nil {
		return authn.NewUserContext(newCtx, *userIdentity)
	}
	return newCtx
}

func (a *AuthInterceptor) countEnabledAndValidated(authenticators map[string]authproviders.Authenticator) (enabled int) {
	for _, p := range authenticators {
		if p.Enabled() && p.Validated() {
			enabled++
		}
	}
	return enabled
}

func (a *AuthInterceptor) getUserIdentity(headers map[string][]string, authenticators map[string]authproviders.Authenticator) *authn.UserIdentity {
	for id, authenticator := range authenticators {
		if !authenticator.Enabled() {
			continue
		}

		user, expiration, err := authenticator.User(headers)
		if err != nil {
			logger.Debugf("user auth error: %s", err)
			continue
		}

		if !authenticator.Validated() {
			if err := a.authenticatorProvider.RecordAuthSuccess(id); err != nil {
				logger.Errorf("Failed to update auth provider status for auth %s with "+
					"loginURL %s: %s", id, authenticator.LoginURL(), err)
			}
		}

		return &authn.UserIdentity{
			User:         user,
			AuthProvider: authenticator,
			Expiration:   expiration,
		}
	}
	return nil
}
