package oidc

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/oauth2"
)

const (
	fragmentCallbackURLPath = "/auth/response/oidc"

	nonceTTL     = 1 * time.Minute
	nonceByteLen = 20

	issuerConfigKey              = "issuer"
	clientIDConfigKey            = "client_id"
	clientSecretConfigKey        = "client_secret"
	dontUseClientSecretConfigKey = "do_not_use_client_secret"
	modeConfigKey                = "mode"
)

type nonceVerificationSetting int

const (
	verifyNonce nonceVerificationSetting = iota
	dontVerifyNonce
)

type backendImpl struct {
	id                 string
	idTokenVerifier    *oidc.IDTokenVerifier
	noncePool          cryptoutils.NoncePool
	defaultUIEndpoint  string
	allowedUIEndpoints set.StringSet

	provider        *provider
	baseRedirectURL url.URL
	baseOauthConfig oauth2.Config
	baseOptions     []oauth2.AuthCodeOption
	formPostMode    bool

	config map[string]string
}

func (p *backendImpl) OnEnable(provider authproviders.Provider) {
}

func (p *backendImpl) OnDisable(provider authproviders.Provider) {
}

func (p *backendImpl) ExchangeToken(ctx context.Context, token, state string) (*authproviders.AuthResponse, string, error) {
	responseValues := make(url.Values, 2)
	responseValues.Set("state", state)
	responseValues.Set("id_token", token)

	return p.processIDPResponse(ctx, responseValues)
}

func (p *backendImpl) RefreshAccessToken(ctx context.Context, refreshToken string) (*authproviders.AuthResponse, error) {
	token, err := p.baseOauthConfig.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	}).Token()
	if err != nil {
		return nil, errors.Wrap(err, "refreshing access token")
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		return nil, errors.New("did not receive an identity token in exchange for the refresh token")
	}

	return p.verifyIDToken(ctx, rawIDToken, dontVerifyNonce)
}

func (p *backendImpl) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if p.provider.RevocationEndpoint == "" {
		return errors.New("provider does not expose a token revocation endpoint")
	}

	revokeTokenData := url.Values{
		"token":           []string{refreshToken},
		"token_type_hint": []string{"refresh_token"},
	}
	resp, err := p.baseOauthConfig.PostRawRequest(ctx, p.provider.RevocationEndpoint, revokeTokenData)
	if err != nil {
		return errors.Wrap(err, "transport error making token revocation request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil
	}

	respBytes, err := ioutils.ReadAtMost(resp.Body, 1024)
	errMsg := fmt.Sprintf("server returned status %s, first 1024 bytes of the response: %s", resp.Status, respBytes)
	if err != nil {
		errMsg = fmt.Sprintf("%s. Additionally, there was an error reading the response body: %v", errMsg, err)
	}
	return errors.New(errMsg)
}

func (p *backendImpl) LoginURL(clientState string, ri *requestinfo.RequestInfo) string {
	return p.loginURL(clientState, ri)
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func (p *backendImpl) verifyIDToken(ctx context.Context, rawIDToken string, nonceVerification nonceVerificationSetting) (*authproviders.AuthResponse, error) {
	idToken, err := p.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	if nonceVerification != dontVerifyNonce && !p.noncePool.ConsumeNonce(idToken.Nonce) {
		return nil, errors.New("invalid token")
	}

	var userInfo userInfoType
	if err := idToken.Claims(&userInfo); err != nil {
		return nil, err
	}

	claim := userInfoToExternalClaims(&userInfo)
	return &authproviders.AuthResponse{
		Claims:     claim,
		Expiration: idToken.Expiry,
	}, nil
}

func newBackend(ctx context.Context, id string, uiEndpoints []string, callbackURLPath string, config map[string]string) (*backendImpl, error) {
	if len(uiEndpoints) == 0 {
		return nil, errors.New("OIDC requires a default UI endpoint")
	}

	issuer := config[issuerConfigKey]
	if issuer == "" {
		return nil, errors.New("no issuer provided")
	}

	if strings.HasPrefix(issuer, "http://") {
		return nil, errors.New("unencrypted http is not allowed for OIDC issuers")
	}
	if !strings.HasPrefix(issuer, "https://") {
		issuer = "https://" + issuer
	}

	oidcCfg := oidc.Config{
		ClientID: config[clientIDConfigKey],
	}

	if oidcCfg.ClientID == "" {
		return nil, errors.New("no client ID provided")
	}

	oidcProvider, issuer, err := createOIDCProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	provider := wrapProvider(oidcProvider)

	p := &backendImpl{
		id: id,
		noncePool: cryptoutils.NewThreadSafeNoncePool(
			cryptoutils.NewNonceGenerator(nonceByteLen, rand.Reader), nonceTTL),
		defaultUIEndpoint:  uiEndpoints[0],
		allowedUIEndpoints: set.NewStringSet(uiEndpoints...),
		provider:           provider,
	}

	p.baseRedirectURL = url.URL{
		Scheme: "https",
	}

	mode := strings.ToLower(config[modeConfigKey])
	switch mode {
	case "", "fragment":
		mode = "fragment"
		p.baseRedirectURL.Path = fragmentCallbackURLPath
		p.baseOptions = append(p.baseOptions, oauth2.SetAuthURLParam("response_mode", "fragment"))
	case "post":
		p.baseRedirectURL.Path = callbackURLPath
		p.baseOptions = append(p.baseOptions, oauth2.SetAuthURLParam("response_mode", "form_post"))
		p.formPostMode = true
	default:
		return nil, errors.Errorf("invalid mode %q", mode)
	}

	responseType := "id_token"
	clientSecret := config[clientSecretConfigKey]
	if clientSecret != "" {
		if !features.RefreshTokens.Enabled() {
			return nil, errors.New("setting a client secret is not supported yet")
		}

		if mode != "post" {
			return nil, errors.Errorf("mode %q cannot be used with a client secret", mode)
		}
		responseType = "code"
	} else if config[dontUseClientSecretConfigKey] == "false" {
		return nil, errors.New("please specify a client secret, or explicitly opt-out of client secret usage")
	}

	p.baseOptions = append(p.baseOptions, oauth2.SetAuthURLParam("response_type", responseType))

	p.idTokenVerifier = oidcProvider.Verifier(&oidcCfg)

	p.baseOauthConfig = oauth2.Config{
		ClientID:     oidcCfg.ClientID,
		ClientSecret: clientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	if features.RefreshTokens.Enabled() && clientSecret != "" && provider.SupportsScope(oidc.ScopeOfflineAccess) {
		p.baseOauthConfig.Scopes = append(p.baseOauthConfig.Scopes, oidc.ScopeOfflineAccess)
	}

	p.config = map[string]string{
		issuerConfigKey:       issuer,
		clientIDConfigKey:     oidcCfg.ClientID,
		clientSecretConfigKey: clientSecret,
		modeConfigKey:         mode,
	}

	return p, nil
}

func (p *backendImpl) Config(redact bool) map[string]string {
	configCopy := maputil.CloneStringStringMap(p.config)
	if redact && configCopy[clientSecretConfigKey] != "" {
		configCopy[clientSecretConfigKey] = "*****"
	}
	return configCopy
}

func (p *backendImpl) MergeConfigInto(newCfg map[string]string) map[string]string {
	mergedCfg := maputil.CloneStringStringMap(newCfg)
	// This handles the case where the client sends an "unchanged" client secret. In that case,
	// we will take the client secret from the stored config and put it into the merged config.
	// We only put secret into the merged config if the new config says it wants to use a client secret, AND the client
	// secret is not specified in the request.
	if mergedCfg[dontUseClientSecretConfigKey] == "false" && mergedCfg[clientSecretConfigKey] == "" {
		mergedCfg[clientSecretConfigKey] = p.config[clientSecretConfigKey]
	}
	return mergedCfg
}

func (p *backendImpl) useCodeFlow() bool {
	return p.baseOauthConfig.ClientSecret != "" && p.formPostMode
}

func (p *backendImpl) oauthCfgForRequest(ri *requestinfo.RequestInfo) *oauth2.Config {
	redirectURL := p.baseRedirectURL
	if p.allowedUIEndpoints.Contains(ri.Hostname) {
		redirectURL.Host = ri.Hostname
		// Allow HTTP only if the client did not use TLS and the host is localhost.
		if !ri.ClientUsedTLS && netutil.IsLocalEndpoint(redirectURL.Host) {
			redirectURL.Scheme = "http"
		}
	} else {
		redirectURL.Host = p.defaultUIEndpoint
	}

	oauthCfg := p.baseOauthConfig
	oauthCfg.RedirectURL = redirectURL.String()

	return &oauthCfg
}

func (p *backendImpl) loginURL(clientState string, ri *requestinfo.RequestInfo) string {
	nonce, err := p.noncePool.IssueNonce()
	if err != nil {
		log.Error("UNEXPECTED: could not issue nonce")
		return ""
	}

	state := idputil.MakeState(p.id, clientState)
	options := make([]oauth2.AuthCodeOption, len(p.baseOptions)+1)
	copy(options, p.baseOptions)
	options[len(p.baseOptions)] = oidc.Nonce(nonce)

	redirectURL := p.baseRedirectURL
	if p.allowedUIEndpoints.Contains(ri.Hostname) {
		redirectURL.Host = ri.Hostname
		// Allow HTTP only if the client did not use TLS and the host is localhost.
		if !ri.ClientUsedTLS && netutil.IsLocalEndpoint(redirectURL.Host) {
			redirectURL.Scheme = "http"
		}
	} else {
		redirectURL.Host = p.defaultUIEndpoint
	}

	return p.oauthCfgForRequest(ri).AuthCodeURL(state, options...)
}

func (p *backendImpl) processIDPResponseForImplicitFlow(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, string, error) {
	_, clientState := idputil.SplitState(responseData.Get("state"))

	rawIDToken := responseData.Get("id_token")
	if rawIDToken == "" {
		return nil, clientState, errors.New("required form fields not found")
	}

	authResp, err := p.verifyIDToken(ctx, rawIDToken, verifyNonce)
	if err != nil {
		return nil, clientState, errors.Wrap(err, "id token verification failed")
	}

	return authResp, clientState, nil
}

func (p *backendImpl) processIDPResponseForCodeFlow(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, string, error) {
	_, clientState := idputil.SplitState(responseData.Get("state"))

	code := responseData.Get("code")
	if code == "" {
		return nil, clientState, errors.New("required form fields not found")
	}

	ri := requestinfo.FromContext(ctx)
	oauthCfg := p.oauthCfgForRequest(&ri)

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, clientState, errors.Wrap(err, "failed to obtain ID token for code")
	}

	rawIDToken, _ := token.Extra("id_token").(string) // needs to be present thanks to `openid` scope
	if rawIDToken == "" {
		return nil, clientState, errors.New("response from server did not contain ID token in violation of OIDC spec")
	}

	authResp, err := p.verifyIDToken(ctx, rawIDToken, verifyNonce)
	if err != nil {
		return nil, clientState, errors.Wrap(err, "ID token verification failed")
	}

	authResp.RefreshToken = token.RefreshToken

	return authResp, clientState, nil
}

func (p *backendImpl) processIDPResponse(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, string, error) {
	if p.useCodeFlow() {
		return p.processIDPResponseForCodeFlow(ctx, responseData)
	}
	return p.processIDPResponseForImplicitFlow(ctx, responseData)
}

func (p *backendImpl) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, string, error) {
	// Form data is guaranteed to be parsed thanks to factory.ProcessHTTPRequest
	return p.processIDPResponse(r.Context(), r.Form)
}

func (p *backendImpl) Validate(ctx context.Context, claims *tokens.Claims) error {
	return nil
}

// Helpers
///////////

// UserInfo is an internal helper struct to unmarshal OIDC token info into.
type userInfoType struct {
	Name   string   `json:"name"`
	EMail  string   `json:"email"`
	UID    string   `json:"sub"`
	Groups []string `json:"groups"`
}

func userInfoToExternalClaims(userInfo *userInfoType) *tokens.ExternalUserClaim {
	claim := &tokens.ExternalUserClaim{
		UserID:   userInfo.UID,
		FullName: userInfo.Name,
		Email:    userInfo.EMail,
	}

	// If no user id, substitute email.
	if claim.UserID == "" {
		claim.UserID = userInfo.EMail
	}

	// Add all fields as attributes.
	claim.Attributes = make(map[string][]string)
	if claim.UserID != "" {
		claim.Attributes["userid"] = []string{claim.UserID}
	}
	if claim.FullName != "" {
		claim.Attributes["name"] = []string{claim.FullName}
	}
	if claim.Email != "" {
		claim.Attributes["email"] = []string{claim.Email}
	}

	// If using non-standard group information add them.
	if len(userInfo.Groups) > 0 {
		claim.Attributes["groups"] = userInfo.Groups
	}
	return claim
}
