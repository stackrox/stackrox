package tokenbased

import (
	"context"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokenbased/user"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	logger = logging.LoggerForModule()
)

// AuthProviderAccessor gives us access to auth providers.
type AuthProviderAccessor interface {
	GetParsedAuthProviders() map[string]authproviders.AuthProvider
	RecordAuthSuccess(id string) error
}

// An AuthInterceptor provides gRPC interceptors that authenticates users.
type AuthInterceptor struct {
	authProviderAccessor AuthProviderAccessor
	userRoleMapper       tokenbased.RoleMapper
	apiTokenParser       tokenbased.IdentityParser
}

// NewAuthInterceptor creates a new AuthInterceptor.
func NewAuthInterceptor(authProviderAccessor AuthProviderAccessor, userRoleMapper tokenbased.RoleMapper, apiTokenParser tokenbased.IdentityParser) *AuthInterceptor {
	return &AuthInterceptor{
		authProviderAccessor: authProviderAccessor,
		userRoleMapper:       userRoleMapper,
		apiTokenParser:       apiTokenParser,
	}
}

// UnaryInterceptor parses headers and embeds authentication metadata in the context, if found.
func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return contextutil.UnaryServerInterceptor(a.authToken)
}

// StreamInterceptor parses headers and embeds authenticated metadata in the context, if found.
func (a *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return contextutil.StreamServerInterceptor(a.authToken)
}

// HTTPInterceptor is an interceptor for http handlers
func (a *AuthInterceptor) HTTPInterceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Lowercase all of the header keys due to grpc metadata keys being lowercased
		newHeaders := make(map[string][]string)
		for k, v := range req.Header {
			newHeaders[strings.ToLower(k)] = v
		}
		ctx := a.addAuthInfoToContext(req.Context(), newHeaders)
		h.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (a *AuthInterceptor) authToken(ctx context.Context) (context.Context, error) {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, nil
	}
	return a.addAuthInfoToContext(ctx, meta), nil
}

func (a *AuthInterceptor) addAuthInfoToContext(ctx context.Context, headers map[string][]string) (newCtx context.Context) {
	authProviders := a.authProviderAccessor.GetParsedAuthProviders()

	// Consult the auth providers and try to get a user identity.
	newCtx, found, authProviderID := a.addUserIdentityIfFound(ctx, headers, authProviders)

	// If we did find an identity, mark the auth provider that gave us the identity as validated.
	if found {
		// If authProviderID is not in the map, it's a programming error since it's returned to us by the (private) function we call.
		authProvider := authProviders[authProviderID]
		if !authProvider.Validated() {
			if err := a.authProviderAccessor.RecordAuthSuccess(authProviderID); err != nil {
				logger.Errorf("Failed to update auth provider status for auth %s with "+
					"loginURL %s: %s", authProviderID, authProvider.LoginURL(), err)
			}
		}
	}

	// If we couldn't get a user identity from any of the auth providers, we see if this corresponds
	// to an API token issued by Central.
	if !found {
		newCtx = a.addAPITokenIdentityIfFound(newCtx, headers)
	}

	// Finally, add the auth configuration context. Note that this _has_ to be done last,
	// since we need this to reflect it if any auth providers were validated.
	return authn.NewAuthConfigurationContext(newCtx, authn.AuthConfiguration{
		ProviderConfigured: a.countEnabledAndValidated(authProviders) > 0,
	})
}

func (a *AuthInterceptor) addUserIdentityIfFound(ctx context.Context, headers map[string][]string, authProviders map[string]authproviders.AuthProvider) (newCtx context.Context, found bool, authProviderID string) {
	userIdentity, found, authProviderID := a.getUserIdentity(headers, authProviders)
	if found {
		newCtx = authn.NewTokenBasedIdentityContext(ctx, userIdentity)
		return
	}
	return ctx, false, ""
}

func (a *AuthInterceptor) addAPITokenIdentityIfFound(ctx context.Context, headers map[string][]string) context.Context {
	identity, err := a.apiTokenParser.Parse(headers, nil)
	if err != nil {
		return ctx
	}
	return authn.NewTokenBasedIdentityContext(ctx, identity)
}

func (a *AuthInterceptor) countEnabledAndValidated(authenticators map[string]authproviders.AuthProvider) (enabled int) {
	for _, p := range authenticators {
		if p.Enabled() && p.Validated() {
			enabled++
		}
	}
	return enabled
}

func (a *AuthInterceptor) getUserIdentity(headers map[string][]string, authProviders map[string]authproviders.AuthProvider) (u user.Identity, found bool, authProviderID string) {
	for id, authProvider := range authProviders {
		if !authProvider.Enabled() {
			continue
		}

		identity, err := authProvider.Parse(headers, a.userRoleMapper)
		if err != nil {
			logger.Debugf("user auth error: %s", err)
			continue
		}

		return user.NewIdentity(identity, authProvider), true, id
	}
	return
}
