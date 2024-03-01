package authproviders

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/tokens"
	userPkg "github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	providersPath    = "providers"
	loginPath        = "login"
	sessionPath      = "session"
	tokenRefreshPath = "tokenrefresh"
	logoutPath       = "logout"
)

// Query parameters for roxctl authorization.
const (
	TokenQueryParameter             = "token"
	RefreshTokenQueryParameter      = "refreshToken"
	ExpiresAtQueryParameter         = "expiresAt"
	AuthorizeCallbackQueryParameter = "authorizeCallback"
)

const (
	testQueryParameter        = "test"
	errorQueryParameter       = "error"
	stateQueryParameter       = "state"
	clientStateQueryParameter = "clientState"
	typeQueryParameter        = "type"
	userQueryParameter        = "user"
)

func (r *registryImpl) URLPathPrefix() string {
	return r.urlPathPrefix
}

func (r *registryImpl) errorURL(err error, typ string, clientState string, testMode bool) *url.URL {
	return &url.URL{
		Path: r.redirectURL,
		Fragment: url.Values{
			testQueryParameter:  {strconv.FormatBool(testMode)},
			errorQueryParameter: {err.Error()},
			typeQueryParameter:  {typ},
			stateQueryParameter: {clientState},
		}.Encode(),
	}
}

func (r *registryImpl) tokenURL(rawToken, typ, clientState string) *url.URL {
	return &url.URL{
		Path: r.redirectURL,
		Fragment: url.Values{
			TokenQueryParameter: {rawToken},
			typeQueryParameter:  {typ},
			stateQueryParameter: {clientState},
		}.Encode(),
	}
}

func (r *registryImpl) userMetadataURL(user *v1.AuthStatus, typ, clientState string, testMode bool) *url.URL {
	var buf bytes.Buffer
	if err := new(jsonpb.Marshaler).Marshal(&buf, user); err != nil {
		return r.errorURL(err, typ, clientState, testMode)
	}

	return &url.URL{
		Path: r.redirectURL,
		Fragment: url.Values{
			testQueryParameter:  {strconv.FormatBool(testMode)},
			userQueryParameter:  {base64.RawURLEncoding.EncodeToString(buf.Bytes())},
			typeQueryParameter:  {typ},
			stateQueryParameter: {clientState},
		}.Encode(),
	}
}

func (r *registryImpl) providersURLPrefix() string {
	return path.Join(r.urlPathPrefix, providersPath) + "/"
}

func (r *registryImpl) loginURLPrefix() string {
	return path.Join(r.urlPathPrefix, loginPath) + "/"
}

func (r *registryImpl) sessionURLPrefix() string {
	return path.Join(r.urlPathPrefix, sessionPath) + "/"
}

func (r *registryImpl) tokenRefreshPath() string {
	return path.Join(r.sessionURLPrefix(), tokenRefreshPath)
}

func (r *registryImpl) logoutPath() string {
	return path.Join(r.sessionURLPrefix(), logoutPath)
}

func (r *registryImpl) loginURL(providerID string) string {
	return path.Join(r.loginURLPrefix(), providerID)
}

func (r *registryImpl) error(w http.ResponseWriter, err error, typ, clientState string, testMode bool) {
	if httpErr, ok := err.(httputil.HTTPError); ok {
		http.Error(w, httpErr.Error(), httpErr.HTTPStatusCode())
		return
	}
	w.Header().Set("Location", r.errorURL(err, typ, clientState, testMode).String())
	w.WriteHeader(http.StatusSeeOther)
}

func (r *registryImpl) initHTTPMux() {
	r.HandleFunc(r.providersURLPrefix(), r.providersHTTPHandler)
	r.HandleFunc(r.loginURLPrefix(), r.loginHTTPHandler)
	r.HandleFunc(r.tokenRefreshPath(), httputil.RESTHandler(r.tokenRefreshEndpoint))
	r.HandleFunc(r.logoutPath(), httputil.RESTHandler(r.logoutEndpoint))
}

func (r *registryImpl) loginHTTPHandler(w http.ResponseWriter, req *http.Request) {
	prefix := r.loginURLPrefix()
	if !strings.HasPrefix(req.URL.Path, prefix) {
		log.Errorf("UNEXPECTED: received HTTP request for invalid URL %v", req.URL)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	providerID := req.URL.Path[len(prefix):]
	clientState := req.URL.Query().Get(clientStateQueryParameter)
	testMode, _ := strconv.ParseBool(req.URL.Query().Get(testQueryParameter))
	authorizeRoxctlCallbackURL := req.URL.Query().Get(AuthorizeCallbackQueryParameter)
	state, err := idputil.AttachStateOrEmpty(clientState, testMode, authorizeRoxctlCallbackURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Attaching state: %v", err), http.StatusBadRequest)
		return
	}
	clientState = state

	provider := r.getAuthProvider(providerID)
	if provider == nil {
		http.Error(w, fmt.Sprintf("Unknown auth provider ID %q", providerID), http.StatusNotFound)
		return
	}

	backend, err := provider.GetOrCreateBackend(req.Context())
	if err != nil {
		r.error(w, errors.Wrap(err, "auth provider is unavailable"), provider.Type(), "", testMode)
		return
	}

	ri := requestinfo.FromContext(req.Context())
	loginURL, err := backend.LoginURL(clientState, &ri)
	if err != nil {
		log.Warnf("could not obtain the login URL for %s: %v", providerID, err)
		http.Error(w, fmt.Sprintf("could not get login URL: %v", err), http.StatusInternalServerError)
		return
	}
	if loginURL == "" {
		log.Warnf("empty login URL for %s", providerID)
		http.Error(w, "empty login URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", loginURL)
	w.WriteHeader(http.StatusSeeOther)
}

// TokenRefreshResponse holds the HTTP response from the token refresh endpoint.
type TokenRefreshResponse struct {
	Token  string    `json:"token,omitempty"`
	Expiry time.Time `json:"expiry,omitempty"`
}

func (r *registryImpl) tokenRefreshEndpoint(req *http.Request) (interface{}, error) {
	refreshTokenCookie, err := req.Cookie(RefreshTokenCookieName)
	if err != nil {
		return nil, httputil.Errorf(http.StatusBadRequest, "could not obtain refresh token cookie: %v", err)
	}

	var cookieData refreshTokenCookieData
	if err := cookieData.Decode(refreshTokenCookie.Value); err != nil {
		return nil, httputil.Errorf(http.StatusBadRequest, "unparseable data in refresh token cookie: %v", err)
	}

	provider, providerBackend, err := r.resolveProviderAndBackend(req.Context(), cookieData.ProviderType, cookieData.ProviderID)
	if err != nil {
		return nil, httputil.Errorf(http.StatusBadRequest, "refresh token cookie references invalid auth provider %q: %v", cookieData.ProviderID, err)
	}

	if providerBackend == nil {
		return nil, httputil.Errorf(http.StatusInternalServerError, "auth provider %q is not currently active", provider.ID())
	}

	refreshTokenEnabledBackend, _ := providerBackend.(RefreshTokenEnabledBackend)
	if refreshTokenEnabledBackend == nil {
		return nil, httputil.Errorf(http.StatusBadRequest, "auth provider backend of type %q does not support refresh tokens", provider.Type())
	}

	authResp, err := refreshTokenEnabledBackend.RefreshAccessToken(req.Context(), cookieData.RefreshTokenData)
	if err != nil {
		return nil, httputil.Errorf(http.StatusInternalServerError, "failed to obtain new access token for refresh token: %v", err)
	}

	token, newRefreshCookie, err := r.issueTokenForResponse(req.Context(), provider, authResp)
	if err != nil {
		return nil, httputil.Errorf(http.StatusInternalServerError, "failed to issue Rox token: %v", err)
	}

	if newRefreshCookie != nil {
		httputil.SetCookie(httputil.ResponseHeaderFromContext(req.Context()), newRefreshCookie)
	}

	// Set the access token cookie for now.
	httputil.SetCookie(httputil.ResponseHeaderFromContext(req.Context()), AccessTokenCookie(token))

	return &TokenRefreshResponse{
		Token:  token.Token,
		Expiry: token.Expiry(),
	}, nil
}

func (r *registryImpl) providersHTTPHandler(w http.ResponseWriter, req *http.Request) {
	prefix := r.providersURLPrefix()
	if !strings.HasPrefix(req.URL.Path, prefix) {
		log.Errorf("UNEXPECTED: received HTTP request for invalid URL %v", req.URL)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	relativePath := req.URL.Path[len(prefix):]
	parts := strings.SplitN(relativePath, "/", 2)
	if len(parts) == 0 {
		log.Debugf("Could not split URL path %q", req.URL.Path[len(prefix):])
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	typ := parts[0]
	factory := r.getFactory(typ)
	if factory == nil {
		log.Debugf("Factory with type %q not found", typ)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	providerID, clientState, err := factory.ProcessHTTPRequest(w, req)
	clientState, mode := idputil.ParseClientState(clientState)
	testMode := mode == idputil.TestAuthMode

	var provider Provider
	if err == nil {
		provider = r.getAuthProvider(providerID)
		if provider == nil {
			err = fmt.Errorf("invalid auth provider ID %q", providerID)
		} else if provider.Type() != parts[0] {
			err = fmt.Errorf("auth provider %s is of invalid type %s", provider.Name(), provider.Type())
		}
	}
	if err != nil {
		r.error(w, err, typ, "", testMode)
		return
	}

	backend, err := provider.GetOrCreateBackend(req.Context())
	if err != nil {
		r.error(w, err, typ, "", testMode)
		return
	}

	authResp, err := backend.ProcessHTTPRequest(w, req)
	if err != nil {
		log.Errorf("error processing HTTP request for provider %s of type %s: %v",
			provider.Name(), provider.Type(), err)
		r.error(w, err, typ, clientState, testMode)
		return
	}

	if authResp == nil || authResp.Claims == nil {
		r.error(w, errox.NoCredentials.CausedBy("authentication response is empty"), typ, clientState, testMode)
		return
	}

	if provider.AttributeVerifier() != nil {
		if err := provider.AttributeVerifier().Verify(authResp.Claims.Attributes); err != nil {
			r.error(w, errox.NoCredentials.CausedBy(err), typ, clientState, testMode)
			return
		}
	}

	// We need all access for retrieving roles.
	user, err := CreateRoleBasedIdentity(sac.WithAllAccess(req.Context()), provider, authResp)
	if err != nil {
		r.error(w, errors.Wrap(err, "cannot create role based identity"), typ, clientState, testMode)
		return
	}

	if testMode {
		user.IdpToken = authResp.IdpToken
		w.Header().Set("Location", r.userMetadataURL(user, typ, clientState, testMode).String())
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	userInfo := user.GetUserInfo()
	if userInfo == nil {
		err := errox.NotAuthorized.CausedBy("failed to get user info")
		r.error(w, err, typ, clientState, testMode)
		return
	}

	userRoles := userInfo.GetRoles()
	if len(userRoles) == 0 {
		r.error(w, auth.ErrNoValidRole, typ, clientState, testMode)
		return
	}

	var tokenInfo *tokens.TokenInfo
	var refreshCookie *http.Cookie
	tokenInfo, refreshCookie, err = r.issueTokenForResponse(req.Context(), provider, authResp)
	if err != nil {
		r.error(w, err, typ, clientState, testMode)
		return
	}

	userPkg.LogSuccessfulUserLogin(log, user)

	if tokenInfo == nil {
		// Assume the ProcessHTTPRequest already took care of writing a response.
		return
	}

	if mode == idputil.AuthorizeRoxctlMode {
		callbackURL, err := url.Parse(clientState)
		if err != nil {
			r.error(w, errox.InvalidArgs.New("invalid callback URL for roxctl authorization"), typ,
				clientState, false)
			return
		}
		// Verify the callback URL again before doing the final redirect, ensuring we _only_ redirect to localhost and
		// no unauthorized third-party.
		if !netutil.IsLocalHost(callbackURL.Hostname()) {
			r.error(w, errox.InvalidArgs.New("roxctl authorization has to specify localhost / "+
				"127.0.0.1 as callback URL"), typ, clientState, false)
		}
		qp := callbackURL.Query()
		qp.Set(TokenQueryParameter, tokenInfo.Token)
		qp.Set(ExpiresAtQueryParameter, tokenInfo.Expiry().Format(time.RFC3339))
		if refreshCookie != nil {
			qp.Set(RefreshTokenQueryParameter, refreshCookie.Value)
		}
		callbackURL.RawQuery = qp.Encode()
		w.Header().Set("Location", callbackURL.String())
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	w.Header().Set("Location", r.tokenURL(tokenInfo.Token, typ, clientState).String())
	if refreshCookie != nil {
		http.SetCookie(w, refreshCookie)
	}
	http.SetCookie(w, AccessTokenCookie(tokenInfo))

	w.WriteHeader(http.StatusSeeOther)
}

func (r *registryImpl) logoutEndpoint(req *http.Request) (interface{}, error) {
	if req.Method != http.MethodPost {
		return nil, httputil.NewError(http.StatusMethodNotAllowed, "only POST requests are allowed")
	}

	// Whatever happens, make sure the cookie gets cleared.
	clearCookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Path:     r.sessionURLPrefix(),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	httputil.SetCookie(httputil.ResponseHeaderFromContext(req.Context()), clearCookie)

	// Whatever happens, make sure the access token cookie gets cleared.
	httputil.SetCookie(httputil.ResponseHeaderFromContext(req.Context()), clearAccessTokenCookie())

	cookieData, err := cookieDataFromRequest(req)
	if cookieData == nil && err == nil {
		return nil, httputil.NewError(http.StatusBadRequest, "you are not currently logged in")
	} else if err != nil {
		return nil, httputil.Errorf(http.StatusBadRequest, "failed to decode or parse cookie: %v", err)
	}

	if cookieData.RefreshToken == "" {
		return nil, nil // not using refresh tokens
	}

	provider, backend, err := r.resolveProviderAndBackend(req.Context(), cookieData.ProviderType, cookieData.ProviderID)
	if err != nil {
		return nil, httputil.Errorf(http.StatusBadRequest, "Failed to resolve provider backend for revoking refresh token: %v", err)
	}
	backendWithRefreshTokens, ok := backend.(RefreshTokenEnabledBackend)
	if !ok {
		return nil, httputil.Errorf(http.StatusBadRequest, "Failed to revoke refresh token: provider %s of type %s does not support refresh tokens", provider.Name(), provider.Type())
	}

	if err := backendWithRefreshTokens.RevokeRefreshToken(req.Context(), cookieData.RefreshTokenData); err != nil {
		return nil, httputil.Errorf(http.StatusInternalServerError, "Failed to revoke refresh token of %s auth provider %s: %v", provider.Type(), provider.Name(), err)
	}
	return nil, nil
}
