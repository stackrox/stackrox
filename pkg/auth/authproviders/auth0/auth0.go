package auth0

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokenbased"
	"github.com/stackrox/rox/pkg/jwt"
)

const (
	cacheExpiry = 5 * time.Minute
)

func init() {
	authproviders.Register("auth0", newFromAPI)
}

// A Config sets up an Auth0 integration.
type config struct {
	// Domain identifies the Auth0 API server to contact (e.g., your_tenant.auth0.com or your_tenant.eu.auth0.com.)
	Domain string
	// ClientID is the identifier of the API client.
	ClientID string
	// Audience is the identifier of this service that will be embedded in the token.
	// Audience is recommended but will default to a Prevent-specific value if unset.
	Audience string
	// Endpoint is the server on which the redirect URL is to be found.
	Endpoint string
	// Enabled says whether the integration is enabled.
	Enabled bool
	// Validated says whether the integration has worked at some point.
	Validated bool
}

type cacheElem struct {
	profile    *auth0Profile
	expiration time.Time
}

// Auth0 integrates with the Auth0 /authorize API to get access tokens.
type auth0 struct {
	config config

	cacheLock    *sync.Mutex
	profileCache map[string]cacheElem
}

// Validate checks the provided Config for errors.
func (c config) Validate() error {
	if c.Domain == "" {
		return errors.New("domain is required")
	}
	if c.ClientID == "" {
		return errors.New("client_id is required")
	}
	return nil
}

// NewAuth0 creates a new Auth0 integration from an API object.
func newAuth0(cfg config) *auth0 {
	return &auth0{
		config:       cfg,
		cacheLock:    new(sync.Mutex),
		profileCache: make(map[string]cacheElem),
	}
}

// NewFromAPI creates a new Auth0 integration from an API object.
func newFromAPI(a *v1.AuthProvider) (authproviders.AuthProvider, error) {
	cfg := config{
		Domain:    a.Config["domain"],
		ClientID:  a.Config["client_id"],
		Audience:  a.Config["audience"],
		Endpoint:  a.GetUiEndpoint(),
		Enabled:   a.GetEnabled(),
		Validated: a.GetValidated(),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return newAuth0(cfg), nil
}

func (c config) audience() string {
	if c.Audience != "" {
		return c.Audience
	}
	return "https://prevent.stackrox.io"
}

// Enabled says if the integration is enabled.
func (a auth0) Enabled() bool {
	return a.config.Enabled
}

// Validated says if the integration is has been successfully used before.
func (a auth0) Validated() bool {
	return a.config.Validated
}

func (a auth0) issuer() string {
	u := url.URL{
		Scheme: "https",
		Host:   a.config.Domain,
		Path:   "/",
	}
	return u.String()
}

func (a auth0) jwksURL() string {
	u := url.URL{
		Scheme: "https",
		Host:   a.config.Domain,
		Path:   "/.well-known/jwks.json",
	}
	return u.String()
}

func (a auth0) redirectURL() string {
	u := url.URL{
		Scheme: "https",
		Host:   a.config.Endpoint,
		Path:   "/auth/response/oidc",
	}
	return u.String()
}

func (a auth0) authorizeParams() url.Values {
	return url.Values{
		"response_type": []string{"token"},
		"client_id":     []string{a.config.ClientID},
		"redirect_uri":  []string{a.redirectURL()},
		//"state": []string{}, // TODO(cg): "Recommended: This value must be used by the client to prevent CSRF attacks." Serve it from the UI?
		"audience":    []string{a.config.audience()},
		"scope":       []string{"openid profile"},
		"access_type": []string{"offline"}, // This is used for certain IdPs. See Auth0 docs: https://auth0.com/docs/api/authentication#social.
	}
}

// LoginURL generates the URL the user should be sent to to authenticate.
func (a auth0) LoginURL() string {
	u := url.URL{
		Scheme:   "https",
		Host:     a.config.Domain,
		Path:     "/authorize",
		RawQuery: a.authorizeParams().Encode(),
	}
	return u.String()
}

// RefreshURL generates the URL that the browser should refresh in the background
// to extend the user's access.
func (a auth0) RefreshURL() string {
	params := a.authorizeParams()
	// Use "Silent Authentication". https://auth0.com/docs/api-auth/tutorials/silent-authentication
	params.Add("prompt", "none")
	u := url.URL{
		Scheme:   "https",
		Path:     "/authorize",
		Host:     a.config.Domain,
		RawQuery: params.Encode(),
	}
	return u.String()
}

// User validates the user, if possible, based on the headers.
func (a auth0) Parse(headers map[string][]string, roleMapper tokenbased.RoleMapper) (identity tokenbased.Identity, err error) {
	jwks := jwt.NewJWKSGetter(a.jwksURL())
	validator := jwt.NewRS256Validator(jwks, a.issuer(), a.config.audience())
	accessToken, claims, err := validator.ValidateFromHeaders(headers)
	if err != nil {
		return nil, fmt.Errorf("token validation: %s", err)
	}
	email, err := a.getProfile(accessToken)
	if err != nil {
		return nil, fmt.Errorf("user profile retrieval: %s", err)
	}
	role, exists := roleMapper.Role(email)
	if !exists {
		return nil, fmt.Errorf("couldn't find role for email: %s", email)
	}
	return tokenbased.NewIdentity(email, role, claims.Expiry.Time()), nil
}

func (a auth0) getCachedProfile(token string) (*auth0Profile, bool) {
	a.cacheLock.Lock()
	defer a.cacheLock.Unlock()
	cache, ok := a.profileCache[token]
	if !ok {
		return nil, false
	}
	if cache.expiration.Before(time.Now()) {
		delete(a.profileCache, token)
		return nil, false
	}
	return cache.profile, true
}

func (a auth0) addCachedProfile(t string, p *auth0Profile) {
	a.cacheLock.Lock()
	defer a.cacheLock.Unlock()
	a.profileCache[t] = cacheElem{
		profile:    p,
		expiration: time.Now().Add(cacheExpiry),
	}
}

func (a auth0) getProfile(token string) (email string, err error) {
	if profile, ok := a.getCachedProfile(token); ok {
		return profile.Name, nil
	}
	c := http.Client{
		Timeout: 5 * time.Second,
	}
	h := http.Header{}
	h.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	r := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   a.config.Domain,
			Path:   "/userinfo",
		},
		Header: h,
	}
	resp, err := c.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("Failed to retrieve profile due to exceeding API limits")
	}
	var prof auth0Profile
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&prof); err != nil {
		return "", fmt.Errorf("profile decoding: %s", err)
	}
	a.addCachedProfile(token, &prof)
	return prof.Name, nil
}

type auth0Profile struct {
	// See https://auth0.com/docs/user-profile/normalized/oidc.
	// TODO(cg): Others are also returned. Add them if we need them.
	Name string `json:"name"`
}
