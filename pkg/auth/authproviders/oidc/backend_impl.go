package oidc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc/internal/endpoint"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/oauth2"
)

const (
	fragmentCallbackURLPath = "/auth/response/oidc"

	issuerConfigKey              = "issuer"
	clientIDConfigKey            = "client_id"
	clientSecretConfigKey        = "client_secret"
	dontUseClientSecretConfigKey = "do_not_use_client_secret"
	modeConfigKey                = "mode"
	disableOfflineAccessKey      = "disable_offline_access"

	userInfoExpiration = 5 * time.Minute

	orgIDAttribute = "orgid"
	orgAdminGroup  = "org_admin"
)

type nonceVerificationSetting int

const (
	verifyNonce nonceVerificationSetting = iota
	dontVerifyNonce
)

type backendImpl struct {
	id                 string
	idTokenVerifier    oidcIDTokenVerifier
	noncePool          cryptoutils.NoncePool
	defaultUIEndpoint  string
	allowedUIEndpoints set.StringSet

	provider        *informedProvider
	baseRedirectURL url.URL
	baseOauthConfig oauth2.Config
	baseOptions     []oauth2.AuthCodeOption
	oauthExchange   exchangeFunc

	responseMode  string
	responseTypes set.FrozenStringSet

	config map[string]string

	httpClient *http.Client
}

func (p *backendImpl) OnEnable(authproviders.Provider) {
}

func (p *backendImpl) OnDisable(authproviders.Provider) {
}

func (p *backendImpl) ExchangeToken(ctx context.Context, token, state string) (*authproviders.AuthResponse, string, error) {
	var responseValues url.Values
	if strings.HasPrefix(token, "#") {
		var err error
		responseValues, err = url.ParseQuery(token[1:])
		if err != nil {
			return nil, "", errors.Wrap(err, "parsing key/value pairs from token")
		}
	} else {
		responseValues = make(url.Values, 2)
		responseValues.Set("id_token", token)
	}
	responseValues.Set("state", state)

	_, clientState := idputil.SplitState(state)
	authResp, err := p.processIDPResponse(ctx, responseValues)
	return authResp, clientState, err
}

func (p *backendImpl) RefreshAccessToken(ctx context.Context, refreshTokenData authproviders.RefreshTokenData) (*authproviders.AuthResponse, error) {
	switch t := refreshTokenData.Type(); t {
	case "refresh_token":
		return p.refreshWithRefreshToken(ctx, refreshTokenData.RefreshToken)
	case "access_token":
		return p.refreshWithAccessToken(ctx, refreshTokenData.RefreshToken)
	default:
		return nil, errors.Errorf("invalid refresh token type %q", t)
	}
}

func (p *backendImpl) refreshWithRefreshToken(ctx context.Context, refreshToken string) (*authproviders.AuthResponse, error) {
	token, err := p.baseOauthConfig.TokenSource(p.injectHTTPClient(ctx), &oauth2.Token{
		RefreshToken: refreshToken,
	}).Token()
	if err != nil {
		return nil, errors.Wrap(err, "refreshing access token")
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		return nil, errors.New("did not receive an identity token in exchange for the refresh token")
	}

	authResp, err := p.verifyIDToken(ctx, rawIDToken, dontVerifyNonce)
	if err != nil {
		return nil, errors.Wrap(err, "verifying ID token after refresh")
	}
	if token.RefreshToken != "" && token.RefreshToken != refreshToken {
		authResp.RefreshTokenType = "refresh_token"
		authResp.RefreshToken = token.RefreshToken
	}

	return authResp, nil
}

func (p *backendImpl) refreshWithAccessToken(ctx context.Context, accessToken string) (*authproviders.AuthResponse, error) {
	return p.fetchUserInfo(ctx, accessToken)
}

func (p *backendImpl) RevokeRefreshToken(ctx context.Context, refreshTokenData authproviders.RefreshTokenData) error {
	if p.provider.RevocationEndpoint == "" {
		return errors.New("provider does not expose a token revocation endpoint")
	}

	revokeTokenData := url.Values{
		"token":           []string{refreshTokenData.RefreshToken},
		"token_type_hint": []string{refreshTokenData.Type()},
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

func (p *backendImpl) LoginURL(clientState string, ri *requestinfo.RequestInfo) (string, error) {
	return p.loginURL(clientState, ri)
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func (p *backendImpl) verifyIDToken(ctx context.Context, rawIDToken string, nonceVerification nonceVerificationSetting) (*authproviders.AuthResponse, error) {
	idToken, err := p.idTokenVerifier.Verify(p.injectHTTPClient(ctx), rawIDToken)
	if err != nil {
		return nil, err
	}

	if nonceVerification != dontVerifyNonce && !p.noncePool.ConsumeNonce(idToken.GetNonce()) {
		return nil, errors.New("invalid token")
	}

	var userInfo userInfoType
	if err := idToken.Claims(&userInfo); err != nil {
		return nil, err
	}

	externalClaims := userInfoToExternalClaims(&userInfo)
	return &authproviders.AuthResponse{
		Claims:     externalClaims,
		Expiration: idToken.GetExpiry(),
	}, nil
}

func (p *backendImpl) fetchUserInfo(ctx context.Context, rawAccessToken string) (*authproviders.AuthResponse, error) {
	userInfoFromEndpoint, err := p.provider.UserInfo(p.injectHTTPClient(ctx), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: rawAccessToken,
	}))
	if err != nil {
		return nil, errors.Wrap(err, "fetching updated userinfo")
	}

	var userInfo userInfoType
	if err := userInfoFromEndpoint.Claims(&userInfo); err != nil {
		return nil, errors.Wrap(err, "parsing userinfo")
	}

	externalClaims := userInfoToExternalClaims(&userInfo)
	return &authproviders.AuthResponse{
		Claims:     externalClaims,
		Expiration: time.Now().Add(userInfoExpiration),
	}, nil
}

var (
	errNoClientIDProvided        = errors.New("no client ID provided")
	errPleaseSpecifyClientSecret = errors.New("please specify a client secret, or explicitly opt-out of client secret usage")
	errQueryWithoutClientSecret  = errors.New("query response mode can only be used with a client secret")
)

func newBackend(ctx context.Context, id string, uiEndpoints []string, callbackURLPath string, config map[string]string,
	providerFactory providerFactoryFunc, exchange exchangeFunc, noncePool cryptoutils.NoncePool) (*backendImpl, error) {
	if len(uiEndpoints) == 0 {
		return nil, errors.New("OIDC requires a default UI endpoint")
	}

	clientID := config[clientIDConfigKey]
	if clientID == "" {
		return nil, errNoClientIDProvided
	}

	issuerHelper, err := endpoint.NewHelper(config[issuerConfigKey])
	if err != nil {
		return nil, err
	}
	provider, err := createOIDCProvider(ctx, issuerHelper, providerFactory)
	if err != nil {
		return nil, err
	}

	b := &backendImpl{
		id:                 id,
		noncePool:          noncePool,
		defaultUIEndpoint:  uiEndpoints[0],
		allowedUIEndpoints: set.NewStringSet(uiEndpoints...),
		provider:           provider,
		httpClient:         issuerHelper.HTTPClient(),
		oauthExchange:      exchange,
	}

	b.baseRedirectURL = url.URL{
		Scheme: "https",
	}

	clientSecret := config[clientSecretConfigKey]

	mode := strings.ToLower(config[modeConfigKey])
	if mode == "auto" {
		mode, err = provider.SelectResponseMode(clientSecret != "")
		if err != nil {
			return nil, errors.Wrap(err, "automatically determining response mode")
		}
		// Nasty back-and-forth between our value and the one used by OIDC.
		if mode == "form_post" {
			mode = "post"
		}
	} else if mode == "" {
		mode = "fragment" // legacy setting
	}

	if clientSecret == "" && config[dontUseClientSecretConfigKey] == "false" {
		if mode == "query" {
			return nil, errQueryWithoutClientSecret
		}
		return nil, errPleaseSpecifyClientSecret
	}

	var responseMode string
	switch mode {
	case "fragment":
		b.baseRedirectURL.Path = fragmentCallbackURLPath
		responseMode = "fragment"
	case "query":
		b.baseRedirectURL.Path = callbackURLPath
		responseMode = "query"
	case "post":
		b.baseRedirectURL.Path = callbackURLPath
		responseMode = "form_post"
	default:
		return nil, errors.Errorf("invalid mode %q", mode)
	}

	if !provider.SupportsResponseMode(responseMode) {
		return nil, errors.Errorf("invalid response mode %q, supported modes: %s", responseMode, strings.Join(provider.ResponseModesSupported, ", "))
	}
	b.baseOptions = append(b.baseOptions, oauth2.SetAuthURLParam("response_mode", responseMode))
	b.responseMode = responseMode

	responseType, err := provider.SelectResponseType(responseMode, clientSecret != "")
	if err != nil {
		return nil, errors.Wrap(err, "determining response type")
	}
	b.responseTypes = set.NewFrozenStringSet(strings.Split(responseType, " ")...)

	b.baseOptions = append(b.baseOptions, oauth2.SetAuthURLParam("response_type", responseType))

	b.idTokenVerifier = provider.Verifier(&oidc.Config{ClientID: clientID})

	disableOfflineAccess := config[disableOfflineAccessKey] == "true"
	b.baseOauthConfig, err = createBaseOAuthConfig(
		clientID,
		clientSecret,
		provider.Endpoint(),
		issuerHelper,
		!disableOfflineAccess && provider.SupportsScope(oidc.ScopeOfflineAccess),
	)
	if err != nil {
		return nil, err
	}

	b.config = map[string]string{
		issuerConfigKey:       issuerHelper.Issuer(),
		clientIDConfigKey:     clientID,
		clientSecretConfigKey: clientSecret,
		modeConfigKey:         mode,
	}
	if disableOfflineAccess {
		b.config[disableOfflineAccessKey] = "true"
	}

	return b, nil
}

func createBaseOAuthConfig(clientID string, clientSecret string, endpoint oauth2.Endpoint, helper *endpoint.Helper, offlineAccessSupported bool) (oauth2.Config, error) {
	baseConfig := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	if clientSecret != "" && offlineAccessSupported {
		baseConfig.Scopes = append(baseConfig.Scopes, oidc.ScopeOfflineAccess)
	}

	var err error
	baseConfig.Endpoint.AuthURL, err = helper.AdjustAuthURL(baseConfig.Endpoint.AuthURL)
	if err != nil {
		return oauth2.Config{}, err
	}
	return baseConfig, nil
}

func (p *backendImpl) Config() map[string]string {
	return p.config
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

func (p *backendImpl) loginURL(clientState string, ri *requestinfo.RequestInfo) (string, error) {
	state := idputil.MakeState(p.id, clientState)
	options := make([]oauth2.AuthCodeOption, len(p.baseOptions), len(p.baseOptions)+1)
	copy(options, p.baseOptions)

	if p.responseTypes.Contains("code") || p.responseTypes.Contains("id_token") {
		// A nonce parameter may only be specified if we can hope to get an id_token (either through
		// code flow, or through implicit flow with id_tokens).
		nonce, err := p.noncePool.IssueNonce()
		if err != nil {
			log.Error("UNEXPECTED: could not issue nonce")
			return "", err
		}

		options = append(options, oidc.Nonce(nonce))
	}

	return p.oauthCfgForRequest(ri).AuthCodeURL(state, options...), nil
}

func (p *backendImpl) processIDPResponseForImplicitFlowWithIDToken(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, error) {
	rawIDToken := responseData.Get("id_token")
	if rawIDToken == "" {
		return nil, errors.New("no id_token field found in response")
	}

	authResp, err := p.verifyIDToken(ctx, rawIDToken, verifyNonce)
	if err != nil {
		return nil, errors.Wrap(err, "id token verification failed")
	}

	return authResp, nil
}

func (p *backendImpl) processIDPResponseForImplicitFlowWithAccessToken(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, error) {
	rawToken := responseData.Get("access_token")
	if rawToken == "" {
		return nil, errors.New("no access_token field found in response")
	}

	authResp, err := p.fetchUserInfo(ctx, rawToken)
	if err != nil {
		return nil, errors.Wrap(err, "fetching user info with access token")
	}

	authResp.RefreshTokenData = authproviders.RefreshTokenData{
		RefreshTokenType: "access_token",
		RefreshToken:     rawToken,
	}

	return authResp, nil
}

func (p *backendImpl) processIDPResponseForCodeFlow(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, error) {
	code := responseData.Get("code")
	if code == "" {
		log.Debugf("Failed to locate 'code' field in IdP response. Response data: %+v", responseData)
		return nil, errors.New("'code' field not found in response data")
	}

	ri := requestinfo.FromContext(ctx)
	oauthCfg := p.oauthCfgForRequest(&ri)

	token, err := p.oauthExchange(p.injectHTTPClient(ctx), oauthCfg, code)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain ID token for code")
	}

	rawIDToken, _ := token.GetExtra("id_token").(string) // needs to be present thanks to `openid` scope
	if rawIDToken == "" {
		return nil, errors.New("response from server did not contain ID token in violation of OIDC spec")
	}

	authResp, err := p.verifyIDToken(ctx, rawIDToken, verifyNonce)
	if err != nil {
		return nil, errors.Wrap(err, "ID token verification failed")
	}

	if token.GetRefreshToken() != "" {
		// we received a proper refresh token
		authResp.RefreshTokenData = authproviders.RefreshTokenData{
			RefreshToken:     token.GetRefreshToken(),
			RefreshTokenType: "refresh_token",
		}
	} else {
		authResp.RefreshTokenData = authproviders.RefreshTokenData{
			RefreshToken:     token.GetAccessToken(),
			RefreshTokenType: "access_token",
		}
	}

	return authResp, nil
}

func (p *backendImpl) processIDPResponse(ctx context.Context, responseData url.Values) (*authproviders.AuthResponse, error) {
	idpError := responseData.Get("error")
	if idpError != "" {
		desc := translateErrorCode(idpError)
		if idpErrorDesc := responseData.Get("error_description"); idpErrorDesc != "" {
			desc = desc + " Additional information from the provider follows. " + idpErrorDesc
		}
		return nil, errors.New(desc)
	}
	now := time.Now()

	var combinedErr error
	if p.responseTypes.Contains("code") {
		authResp, err := p.processIDPResponseForCodeFlow(ctx, responseData)
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		} else {
			return authResp, nil
		}
	}

	// Try to authenticate with both the access and the ID token, such that if necessary, we can select the one
	// that is valid for longer.
	var authRespToken, authRespIDToken *authproviders.AuthResponse
	if p.responseTypes.Contains("token") {
		var err error
		authRespToken, err = p.processIDPResponseForImplicitFlowWithAccessToken(ctx, responseData)
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		}
	}
	if p.responseTypes.Contains("id_token") {
		var err error
		authRespIDToken, err = p.processIDPResponseForImplicitFlowWithIDToken(ctx, responseData)
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		}
	}

	// If we got both a token and ID token response, choose the one that lasts for longer (if the server did
	// not give us an expiration time for the access token, assume it lasts at least as long as the ID token).
	if authRespToken != nil && authRespIDToken != nil {
		expiresInStr := responseData.Get("expires_in")
		if expiresInStr == "" {
			// No expiration for access token, trust it will be valid for long enough.
			return authRespToken, nil
		}
		expiresInSecs, err := strconv.Atoi(expiresInStr)
		if err != nil {
			log.Warnf("unparseable expires_in time %q returned by authentication server", expiresInStr)
			return authRespToken, nil
		}

		accessTokenExpiry := now.Add(time.Second * time.Duration(expiresInSecs))
		// expiration of the AuthResponse will match exp claim of the ID token
		if accessTokenExpiry.Before(authRespIDToken.Expiration) {
			// prefer ID token only if it expires later.
			return authRespIDToken, nil
		}
		return authRespToken, nil
	} else if authRespToken != nil {
		return authRespToken, nil
	} else if authRespIDToken != nil {
		return authRespIDToken, nil
	}

	if combinedErr == nil {
		combinedErr = errors.Errorf("no supported response type: %s", p.responseTypes.ElementsString(", "))
	}

	return nil, combinedErr
}

func translateErrorCode(idpError string) string {
	switch idpError {
	case "unauthorized_client":
		return "Identity provider claims that this authentication provider configuration is not authorized to request an authorization code or access token using this method."
	default:
		return fmt.Sprintf("Identity provider returned a %q error.", idpError)
	}
}

func (p *backendImpl) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	var values url.Values
	switch r.Method {
	case http.MethodGet:
		if p.responseMode != "query" {
			return nil, errors.Errorf("this URL should only be accessed with method %s when using the 'query' response mode, but requested response mode was %q", r.Method, p.responseMode)
		}
		values = r.URL.Query()
	case http.MethodPost:
		if p.responseMode != "form_post" {
			return nil, errors.Errorf("this URL should only be accessed with method %s when using the 'form_post' response mode, but requested response mode was %q", r.Method, p.responseMode)
		}
		// Form data is guaranteed to be parsed thanks to factory.ProcessHTTPRequest
		values = r.Form
	default:
		return nil, errors.Errorf("method %s not allowed for this URL", r.Method)
	}

	return p.processIDPResponse(r.Context(), values)
}

func (p *backendImpl) Validate(context.Context, *tokens.Claims) error {
	return nil
}

// Helpers
///////////

// userInfoType is an internal helper struct to unmarshal OIDC token info into.
type userInfoType struct {
	Name   string   `json:"name"`
	EMail  string   `json:"email"`
	UID    string   `json:"sub"`
	Groups []string `json:"groups"`
	// Every Red Hat SSO user belongs to exactly one organization, claim
	// "account_id" represents that organisation. See more on the claims here:
	// 	https://source.redhat.com/groups/public/it-user/it_user_team_wiki/topic_external_sso_enablements#attributes-needed
	OrgID string `json:"account_id"`
	// Claim "is_org_admin" is the claim that identifies organisation admins within sso.redhat.com.
	IsOrgAdmin bool `json:"is_org_admin"`
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
		claim.Attributes[authproviders.UseridAttribute] = []string{claim.UserID}
	}
	if claim.FullName != "" {
		claim.Attributes[authproviders.NameAttribute] = []string{claim.FullName}
	}
	if claim.Email != "" {
		claim.Attributes[authproviders.EmailAttribute] = []string{claim.Email}
	}

	// If using non-standard group information add them.
	if len(userInfo.Groups) > 0 {
		claim.Attributes[authproviders.GroupsAttribute] = userInfo.Groups
	}

	// Add sso.redhat.com attributes.
	if userInfo.OrgID != "" {
		claim.Attributes[orgIDAttribute] = []string{userInfo.OrgID}
	}
	if userInfo.IsOrgAdmin {
		claim.Attributes[authproviders.GroupsAttribute] = append(claim.Attributes[authproviders.GroupsAttribute], orgAdminGroup)
	}
	return claim
}

func (p *backendImpl) injectHTTPClient(ctx context.Context) context.Context {
	if p.httpClient != nil {
		return oidc.ClientContext(ctx, p.httpClient)
	}
	return ctx
}
