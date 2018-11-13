package oidc

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"golang.org/x/oauth2"
)

const (
	fragmentCallbackURLPath = "/auth/response/oidc"

	nonceTTL     = 1 * time.Minute
	nonceByteLen = 20

	issuerConfigKey   = "issuer"
	clientIDConfigKey = "client_id"
	modeConfigKey     = "mode"
)

type provider struct {
	idTokenVerifier *oidc.IDTokenVerifier
	noncePool       cryptoutils.NoncePool
	loginURL        func(string) string
}

func (p *provider) ExchangeToken(ctx context.Context, externalRawToken, state string) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	claim, opts, err := p.verifyIDToken(ctx, externalRawToken)
	_, clientState := splitState(state)
	if err != nil {
		return nil, nil, clientState, err
	}
	return claim, opts, clientState, nil
}

func (p *provider) LoginURL(clientState string) string {
	return p.loginURL(clientState)
}

func (p *provider) RefreshURL() string {
	return ""
}

func (p *provider) verifyIDToken(ctx context.Context, rawIDToken string) (*tokens.ExternalUserClaim, []tokens.Option, error) {
	idToken, err := p.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, err
	}

	if !p.noncePool.ConsumeNonce(idToken.Nonce) {
		return nil, nil, errors.New("invalid token")
	}

	var userInfo userInfoType
	if err := idToken.Claims(&userInfo); err != nil {
		return nil, nil, err
	}

	claim := userInfoToExternalClaims(&userInfo)
	return claim, []tokens.Option{tokens.WithExpiry(idToken.Expiry)}, nil
}

// The go-oidc library has two annoying characteristics when it comes to creating provider instances:
// - The context is passed on to the remoteKeySource that is being created. Hence, we can't use a short-lived context
//   (such as the request context), as otherwise subsequent verifications will fail because the keys have not been
//   retrieved.
// - The check for the issuer is done strictly, not even tolerating a trailing slash (which makes it very hard to omit
//   the `https://` prefix, as is common).
// We therefore add a wrapper method that calls `oidc.NewProvider` with the background context and writes the result to
// a channel, and retries in case of an error with a trailing slash added or removed.
//
type createOIDCProviderResult struct {
	issuer   string
	provider *oidc.Provider
	err      error
}

func createOIDCProviderAsync(issuer string, resultC chan<- createOIDCProviderResult) {
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		if strings.HasSuffix(issuer, "/") {
			issuer = strings.TrimSuffix(issuer, "/")
		} else {
			issuer = issuer + "/"
		}
		provider, err = oidc.NewProvider(context.Background(), issuer)
	}
	resultC <- createOIDCProviderResult{issuer: issuer, provider: provider, err: err}
}

func createOIDCProvider(ctx context.Context, issuer string) (*oidc.Provider, string, error) {
	resultC := make(chan createOIDCProviderResult, 1)
	go createOIDCProviderAsync(issuer, resultC)
	select {
	case res := <-resultC:
		return res.provider, res.issuer, res.err
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
}

func newProvider(ctx context.Context, id string, uiEndpoint string, callbackURLPath string, config map[string]string) (*provider, map[string]string, error) {
	issuer := config[issuerConfigKey]
	if issuer == "" {
		return nil, nil, errors.New("no issuer provided")
	}

	if strings.HasPrefix(issuer, "http://") {
		return nil, nil, errors.New("unencrypted http is not allowed for OIDC issuers")
	}
	if !strings.HasPrefix(issuer, "https://") {
		issuer = "https://" + issuer
	}

	oidcCfg := oidc.Config{
		ClientID: config[clientIDConfigKey],
	}

	if oidcCfg.ClientID == "" {
		return nil, nil, errors.New("no client ID provided")
	}

	redirectURL := &url.URL{
		Scheme: "https",
		Host:   uiEndpoint,
	}

	options := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("response_type", "id_token"),
	}
	mode := strings.ToLower(config[modeConfigKey])
	switch mode {
	case "", "fragment":
		mode = "fragment"
		redirectURL.Path = fragmentCallbackURLPath
		options = append(options, oauth2.SetAuthURLParam("response_mode", "fragment"))
	case "post":
		redirectURL.Path = callbackURLPath
		options = append(options, oauth2.SetAuthURLParam("response_mode", "form_post"))
	default:
		return nil, nil, fmt.Errorf("invalid mode %q", mode)
	}

	oidcProvider, issuer, err := createOIDCProvider(ctx, issuer)
	if err != nil {
		return nil, nil, err
	}

	p := &provider{
		noncePool: cryptoutils.NewThreadSafeNoncePool(
			cryptoutils.NewNonceGenerator(nonceByteLen, rand.Reader), nonceTTL),
	}

	p.idTokenVerifier = oidcProvider.Verifier(&oidcCfg)

	oauthCfg := oauth2.Config{
		ClientID:    oidcCfg.ClientID,
		Endpoint:    oidcProvider.Endpoint(),
		RedirectURL: redirectURL.String(),
		Scopes:      []string{oidc.ScopeOpenID, "profile"},
	}

	p.loginURL = func(clientState string) string {
		nonce, err := p.noncePool.IssueNonce()
		if err != nil {
			log.Errorf("UNEXPECTED: could not issue nonce")
			return ""
		}
		state := makeState(id, clientState)
		newOptions := make([]oauth2.AuthCodeOption, len(options)+1)
		copy(newOptions, options)
		newOptions[len(options)] = oidc.Nonce(nonce)
		return oauthCfg.AuthCodeURL(state, newOptions...)
	}

	effectiveConfig := map[string]string{
		issuerConfigKey:   issuer,
		clientIDConfigKey: oidcCfg.ClientID,
		modeConfigKey:     mode,
	}

	return p, effectiveConfig, nil
}

func (p *provider) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	// Form data is guaranteed to be parsed thanks to factory.ProcessHTTPRequest
	rawIDToken := r.FormValue("id_token")
	if rawIDToken == "" {
		return nil, nil, "", errors.New("required form fields not found")
	}

	_, clientState := splitState(r.FormValue("state"))

	userClaim, opts, err := p.verifyIDToken(r.Context(), rawIDToken)
	if err != nil {
		return nil, nil, clientState, fmt.Errorf("id token verification failed: %v", err)
	}

	return userClaim, opts, clientState, nil
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
