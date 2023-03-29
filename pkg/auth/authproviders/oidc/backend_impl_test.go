package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc/internal/endpoint"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var (
	_ authproviders.RefreshTokenEnabledBackend = (*backendImpl)(nil)
)

func TestMerge(t *testing.T) {
	for _, testCase := range []struct {
		desc           string
		oldConfig      map[string]string
		newConfig      map[string]string
		expectedConfig map[string]string
	}{
		{
			"old config with client secret, new config wants to use client secret but is empty",
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "SECRET",
			},
			map[string]string{
				DontUseClientSecretConfigKey: "false",
			},
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "SECRET",
			},
		},
		{
			"old config with client secret, new config wants to use client secret and specifies a new one",
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "SECRET",
			},
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "NEWSECRET",
			},
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "NEWSECRET",
			},
		},
		{
			"old config with no client secret, new config wants to use client secret",
			map[string]string{
				DontUseClientSecretConfigKey: "true",
			},
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "NEWSECRET",
			},
			map[string]string{
				DontUseClientSecretConfigKey: "false",
				ClientSecretConfigKey:        "NEWSECRET",
			},
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			f := &factory{}
			merged := f.MergeConfig(c.newConfig, c.oldConfig)
			assert.Equal(t, c.expectedConfig, merged)
		})
	}
}

// ideally this would be const...
var (
	allResponseTypes = []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"}
	allResponseModes = []string{"query", "fragment", "form_post"}
	allScopes        = []string{"openid", "profile", "offline_access", "name", "given_name", "family_name", "nickname",
		"email", "email_verified", "picture", "created_at", "identities", "phone", "address"}
)

type mockOAuth2Token struct {
	idToken string
}

func (m mockOAuth2Token) GetAccessToken() string {
	return "mock-access-token"
}

func (m mockOAuth2Token) GetRefreshToken() string {
	return "mock-refresh-token"
}

func (m mockOAuth2Token) GetExtra(string) interface{} {
	return m.idToken
}

type claims struct {
	name  string
	email string
	uid   string
}

func (c claims) serialize(nonce string) string {
	return strings.Join([]string{c.name, c.email, c.uid, nonce}, ":")
}

type wantBackend struct {
	responseMode    string
	responseTypes   []string
	config          map[string]string
	baseOauthConfig *oauth2.Config
}

type responseValueProvider interface {
	serialize(nonce string) string
}

type literalValue struct {
	value string
}

func (v literalValue) serialize(string) string {
	return v.value
}

func TestBackend(t *testing.T) {
	const mockAccessToken = "mock-access-token"
	const mockAuthorizationCode = "mock-authz-code"
	transientError := errors.New("simulated transient error")

	// Claims supplied by the method which is supposed to be selected by the given test case
	suppliedClaims := claims{
		uid:   "mock-uid",
		name:  "Mock Name",
		email: "mock@e-mail.com",
	}
	// Claims supplied by the method which is supposed to NOT be selected by the given test case
	alternativeSuppliedClaims := claims{
		uid:   "mock-uid2",
		name:  "Mock Name2",
		email: "mock2@e-mail.com",
	}
	// The claims expected to be returned in the AuthResponse from processIDPResponse
	wantProcessIDPResponseAuthResponseClaims := tokens.ExternalUserClaim{
		UserID:   suppliedClaims.uid,
		FullName: suppliedClaims.name,
		Email:    suppliedClaims.email,
		Attributes: map[string][]string{
			authproviders.EmailAttribute:  {suppliedClaims.email},
			authproviders.NameAttribute:   {suppliedClaims.name},
			authproviders.UseridAttribute: {suppliedClaims.uid},
		},
	}

	tests := map[string]struct {
		config                      map[string]string
		oidcProvider                oidcProvider
		wantBackend                 *wantBackend
		wantBackendErr              error
		issueNonce                  bool
		idpResponseTemplate         map[string]responseValueProvider
		exchangedTokenClaims        *claims
		wantProcessIDPResponseError string
		assertInsecureClient        bool
	}{
		"no client id": {
			config: map[string]string{
				ClientIDConfigKey:     "",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "some-issuer",
				ModeConfigKey:         "post",
			},
			wantBackendErr: errNoClientIDProvided,
		},
		"bad issuer": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "",
				ModeConfigKey:         "post",
			},
			wantBackendErr: endpoint.ErrNoIssuerProvided,
		},
		"transient backend error": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "post",
			},
			oidcProvider:   nil,
			wantBackendErr: transientError,
		},
		"no client secret and no confirmation": {
			config: map[string]string{
				ClientIDConfigKey:            "testclientid",
				ClientSecretConfigKey:        "",
				DontUseClientSecretConfigKey: "false",
				IssuerConfigKey:              "test-issuer",
				ModeConfigKey:                "post",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackendErr: errPleaseSpecifyClientSecret,
		},
		"no client secret and no confirmation in query mode": {
			config: map[string]string{
				ClientIDConfigKey:            "testclientid",
				ClientSecretConfigKey:        "",
				DontUseClientSecretConfigKey: "false",
				IssuerConfigKey:              "test-issuer",
				ModeConfigKey:                "query",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackendErr: errQueryWithoutClientSecret,
		},
		"insecure client mode form post": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "https+insecure://test-issuer",
				ModeConfigKey:         "post",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			assertInsecureClient: true,
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https+insecure://test-issuer",
					ModeConfigKey:         "post",
				},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"code": literalValue{mockAuthorizationCode},
			},
			exchangedTokenClaims: &suppliedClaims,
		},
		"mode form post": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "https://test-issuer",
				ModeConfigKey:         "post",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "post",
				},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"code": literalValue{mockAuthorizationCode},
			},
			exchangedTokenClaims: &suppliedClaims,
		},
		"mode form post and mismatching idp response": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "https://test-issuer",
				ModeConfigKey:         "post",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: []string{"code"},
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "post",
				},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
			wantProcessIDPResponseError: "1 error occurred:\n\t* 'code' field not found in response data\n\n",
		},
		"mode form post with bad nonce": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "https://test-issuer",
				ModeConfigKey:         "post",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "post",
				},
			},
			issueNonce: false,
			idpResponseTemplate: map[string]responseValueProvider{
				"code": literalValue{mockAuthorizationCode},
			},
			exchangedTokenClaims:        &suppliedClaims,
			wantProcessIDPResponseError: "1 error occurred:\n\t* ID token verification failed: invalid token\n\n",
		},
		"mode query": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "query",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "query",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "query",
				},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"code": literalValue{mockAuthorizationCode},
			},
			exchangedTokenClaims: &suppliedClaims,
		},
		"mode fragment with access_token only": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "https://test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "fragment",
				},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
		},
		"mode fragment with access_token only fails on userinfo endpoint failure": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "https://test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
				userInfoShouldFail:         true,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "fragment",
				},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
			wantProcessIDPResponseError: "2 errors occurred:\n\t* fetching user info with access token: fetching updated userinfo: simulated UserInfo endpoint failure\n\t* no id_token field found in response\n\n",
		},
		"mode fragment with id_token only": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token": suppliedClaims,
			},
		},
		"mode fragment with id_token only and bad nonce": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: false,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token": suppliedClaims,
			},
			wantProcessIDPResponseError: "2 errors occurred:\n\t* no access_token field found in response\n\t* id token verification failed: invalid token\n\n",
		},
		"mode fragment with both token and id_token": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token":     alternativeSuppliedClaims,
				"access_token": literalValue{mockAccessToken},
			},
		},
		"mode fragment with both token and id_token and invalid expires_in": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token":     alternativeSuppliedClaims,
				"access_token": literalValue{mockAccessToken},
				"expires_in":   literalValue{"garbage"},
			},
		},
		"mode fragment with both token and id_token and long expires_in": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token":     alternativeSuppliedClaims,
				"access_token": literalValue{mockAccessToken},
				"expires_in":   literalValue{"604800"}, // a week
			},
		},
		"mode fragment with both token and id_token and short expires_in": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: alternativeSuppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token":     suppliedClaims,
				"access_token": literalValue{mockAccessToken},
				"expires_in":   literalValue{"20"},
			},
		},
		"mode fragment with both token and id_token, and userinfo endpoint failure": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "fragment",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: alternativeSuppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
				userInfoShouldFail:         true,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"id_token":     suppliedClaims,
				"access_token": literalValue{mockAccessToken},
			},
		},
		"legacy no mode setting, equal to fragment, with access token only": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "fragment",
				},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
		},
		"mode auto, error due to no response types": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: []string{"code"},
				responseModesSupported: []string{"blah"},
			},
			wantBackendErr: errors.New("automatically determining response mode: could not determine a suitable response mode, supported modes are: blah"),
		},
		"mode auto, with client secret, form post mode result": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "post",
				},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"code": literalValue{mockAuthorizationCode},
			},
			exchangedTokenClaims: &suppliedClaims,
		},
		"mode auto, with client secret, non-code and non-post mode result": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     []string{"token", "id_token", "token id_token"},
				responseModesSupported:     []string{"query", "fragment"},
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "fragment",
				},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
		},
		"mode auto, with client secret, code with non-post mode result": {
			config: map[string]string{
				ClientIDConfigKey:     "testclientid",
				ClientSecretConfigKey: "testsecret",
				IssuerConfigKey:       "test-issuer",
				ModeConfigKey:         "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     []string{"code", "token", "id_token", "token id_token"},
				responseModesSupported:     []string{"query", "fragment"},
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "query",
				responseTypes: []string{"code"},
				config: map[string]string{
					ClientIDConfigKey:     "testclientid",
					ClientSecretConfigKey: "testsecret",
					IssuerConfigKey:       "https://test-issuer",
					ModeConfigKey:         "query",
				},
				baseOauthConfig: &oauth2.Config{
					ClientID:     "testclientid",
					ClientSecret: "testsecret",
					Endpoint: oauth2.Endpoint{
						AuthURL:  "fake-auth-url",
						TokenURL: "fake-token-url",
					},
					RedirectURL: "",
					Scopes:      []string{"openid", "profile", "email", "offline_access"},
				},
			},
			issueNonce: true,
			idpResponseTemplate: map[string]responseValueProvider{
				"code": literalValue{mockAuthorizationCode},
			},
			exchangedTokenClaims: &suppliedClaims,
		},
		"mode auto, no client secret": {
			config: map[string]string{
				ClientIDConfigKey:            "testclientid",
				ClientSecretConfigKey:        "",
				DontUseClientSecretConfigKey: "true",
				IssuerConfigKey:              "https://test-issuer",
				ModeConfigKey:                "",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
		},
		"mode auto, with client secret, disable offline_access scope": {
			config: map[string]string{
				ClientIDConfigKey:                  "testclientid",
				ClientSecretConfigKey:              "testsecret",
				IssuerConfigKey:                    "https://test-issuer",
				ModeConfigKey:                      "",
				DisableOfflineAccessScopeConfigKey: "true",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported:     allResponseTypes,
				responseModesSupported:     allResponseModes,
				claimsFromUserInfoEndpoint: suppliedClaims,
				userInfoAssertAccessToken:  mockAccessToken,
			},
			wantBackend: &wantBackend{
				responseMode:  "fragment",
				responseTypes: []string{"token", "id_token"},
				baseOauthConfig: &oauth2.Config{
					ClientID:     "testclientid",
					ClientSecret: "testsecret",
					Endpoint: oauth2.Endpoint{
						AuthURL:  "fake-auth-url",
						TokenURL: "fake-token-url",
					},
					RedirectURL: "",
					Scopes:      []string{"openid", "profile", "email"},
				},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"access_token": literalValue{mockAccessToken},
			},
		},
		"unauthorized client error from idp": {
			config: map[string]string{
				ClientIDConfigKey:            "testclientid",
				ClientSecretConfigKey:        "testsecret",
				DontUseClientSecretConfigKey: "true",
				IssuerConfigKey:              "https://test-issuer",
				ModeConfigKey:                "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				// Typical response from KeyCloak when we request implicit or hybrid flow from a confidential-only client config.
				"error":             literalValue{"unauthorized_client"},
				"error_description": literalValue{"Client is not allowed to initiate browser login with given response_type. Implicit flow is disabled for the client."},
			},
			wantProcessIDPResponseError: "Identity provider claims that this authentication provider configuration is not authorized to request an authorization code or access token using this method. " +
				"Additional information from the provider follows. Client is not allowed to initiate browser login with given response_type. Implicit flow is disabled for the client.",
		},
		"error from idp without description": {
			config: map[string]string{
				ClientIDConfigKey:            "testclientid",
				ClientSecretConfigKey:        "testsecret",
				DontUseClientSecretConfigKey: "true",
				IssuerConfigKey:              "https://test-issuer",
				ModeConfigKey:                "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"error": literalValue{"code1"},
			},
			wantProcessIDPResponseError: "Identity provider returned a \"code1\" error.",
		},
		"error from idp with description": {
			config: map[string]string{
				ClientIDConfigKey:            "testclientid",
				ClientSecretConfigKey:        "testsecret",
				DontUseClientSecretConfigKey: "true",
				IssuerConfigKey:              "https://test-issuer",
				ModeConfigKey:                "auto",
			},
			oidcProvider: &mockOIDCProvider{
				responseTypesSupported: allResponseTypes,
				responseModesSupported: allResponseModes,
			},
			wantBackend: &wantBackend{
				responseMode:  "form_post",
				responseTypes: []string{"code"},
			},
			idpResponseTemplate: map[string]responseValueProvider{
				"error":             literalValue{"code2"},
				"error_description": literalValue{"Blah blah blah."},
			},
			wantProcessIDPResponseError: "Identity provider returned a \"code2\" error. Additional information from the provider follows. Blah blah blah.",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := NewFactory("/callback/path").(*factory)

			if tt.oidcProvider != nil {
				tt.oidcProvider.(*mockOIDCProvider).t = t
				f.providerFactoryFunc = func(ctx context.Context, _ string) (oidcProvider, error) {
					client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
					assert.True(t, ok, "context passed to provider creation must contain an HTTP client")
					if client.Transport != nil || tt.assertInsecureClient {
						transport, ok := client.Transport.(*http.Transport)
						assert.Truef(t, ok, "cannot check client transport of %T", client.Transport)
						assert.Equal(t, tt.assertInsecureClient, transport.TLSClientConfig.InsecureSkipVerify)
					}
					return tt.oidcProvider, nil
				}
			} else {
				f.providerFactoryFunc = func(ctx context.Context, issuer string) (oidcProvider, error) {
					return nil, transientError
				}
			}
			// generate nonce that will be checked when verifying token
			nonce := "forged-nonce"
			if tt.issueNonce {
				var err error
				nonce, err = f.noncePool.IssueNonce()
				assert.NoError(t, err, "generating a nonce failed")
			}
			// provide a mock exchange function if using the token endpoint
			if tt.exchangedTokenClaims != nil {
				f.oauthExchange = func(ctx context.Context, oauthCfg *oauth2.Config, code string) (oauth2Token, error) {
					require.Equal(t, mockAuthorizationCode, code, "unexpected authorization code passed to OAuth2 exchange function")
					return mockOAuth2Token{idToken: tt.exchangedTokenClaims.serialize(nonce)}, nil
				}
			} else {
				f.oauthExchange = func(ctx context.Context, oauthCfg *oauth2.Config, code string) (oauth2Token, error) {
					t.Fatal("exchange function should not be called")
					return nil, nil
				}
			}

			// create backend and perform related assertions
			backendInterface, err := f.CreateBackend(context.TODO(), "abcde-12345", []string{"endpoint1", "endpoint2"}, tt.config, nil)
			gotBackend := backendInterface.(*backendImpl)
			require.Equal(t, fmt.Sprint(tt.wantBackendErr), fmt.Sprint(err), "Unexpected newBackend() error")
			tt.wantBackend.assertMatches(t, gotBackend)
			if gotBackend == nil {
				return
			}

			// call processIDPResponse and perform related assertions
			idpResponseData := url.Values{}
			for name, provider := range tt.idpResponseTemplate {
				idpResponseData[name] = []string{provider.serialize(nonce)}
			}
			authResp, err := gotBackend.processIDPResponse(context.TODO(), idpResponseData)
			if tt.wantProcessIDPResponseError != "" {
				assert.EqualError(t, err, tt.wantProcessIDPResponseError)
			} else {
				require.NoError(t, err, "processIDPResponse returned error")
				assert.Equal(t, &wantProcessIDPResponseAuthResponseClaims, authResp.Claims, "unexpected auth response from processIDPResponse")
			}
		})
	}
}

// assertMatches makes sure the interesting fields of backendImpl returned from code under test match our expectations.
// Unfortunately that struct is too big and complex to just "reflect.DeepEquals" it, so we hand-pick individual fields.
func (want *wantBackend) assertMatches(t *testing.T, got *backendImpl) {
	require.Equalf(t, want == nil, got == nil, "newBackend() backend %v, want %v", got, want)
	if got == nil {
		return
	}
	assert.Equal(t, want.responseMode, got.responseMode, "unexpected responseMode")
	assert.Truef(t, got.responseTypes.Unfreeze().Equal(set.NewStringSet(want.responseTypes...)), "responseTypes got = %v, want %v",
		got.responseTypes.ElementsString(" "),
		want.responseTypes)
	if want.config != nil {
		assert.Equal(t, want.config, got.config, "unexpected config")
	}
	if want.baseOauthConfig != nil {
		assert.Equal(t, *want.baseOauthConfig, got.baseOauthConfig, "unexpected baseOauthConfig")
	}
}

type mockOIDCProvider struct {
	responseTypesSupported     []string
	responseModesSupported     []string
	claimsFromUserInfoEndpoint claims
	userInfoAssertAccessToken  string
	userInfoShouldFail         bool
	t                          *testing.T
}

func (m *mockOIDCProvider) Claims(v *extraDiscoveryInfo) error {
	info := extraDiscoveryInfo{
		ScopesSupported:        allScopes,
		RevocationEndpoint:     "https://sr-dev.auth0.com/oauth/revoke",
		ResponseTypesSupported: m.responseTypesSupported,
		ResponseModesSupported: m.responseModesSupported,
	}
	// test JSON round-trip rather than assign directly
	infoBytes, err := json.Marshal(info)
	require.NoError(m.t, err)
	return json.Unmarshal(infoBytes, v)
}

func (m *mockOIDCProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  "fake-auth-url",
		TokenURL: "fake-token-url",
	}
}

type mockOIDCUserInfo struct {
	claims claims
}

func (m mockOIDCUserInfo) Claims(v interface{}) error {
	switch u := v.(type) {
	case *userInfoType:
		u.UID = m.claims.uid
		u.Name = m.claims.name
		u.EMail = m.claims.email
	case map[string]interface{}, *map[string]interface{}:
		return nil
	default:
		return errors.Errorf("unsupported type %T", v)
	}
	return nil
}

func (m *mockOIDCProvider) UserInfo(_ context.Context, tokenSource oauth2.TokenSource) (oidcUserInfo, error) {
	token, err := tokenSource.Token()
	require.NoError(m.t, err) // our code provides a static token source, which should never fail
	require.Equal(m.t, m.userInfoAssertAccessToken, token.AccessToken)
	if m.userInfoShouldFail {
		return nil, errors.New("simulated UserInfo endpoint failure")
	}
	return mockOIDCUserInfo{claims: m.claimsFromUserInfoEndpoint}, nil
}

func (m *mockOIDCProvider) Verifier(*oidc.Config) oidcIDTokenVerifier {
	return mockVerifier{}
}

type mockVerifier struct {
}

func (m mockVerifier) Verify(_ context.Context, token string) (oidcIDToken, error) {
	// undo claims.serialize
	s := strings.Split(token, ":")
	return mockOIDCToken{name: s[0], email: s[1], uid: s[2], nonce: s[3]}, nil
}

type mockOIDCToken struct {
	name  string
	email string
	uid   string
	nonce string
}

func (m mockOIDCToken) GetNonce() string {
	return m.nonce
}

func (m mockOIDCToken) Claims(v interface{}) error {
	switch u := v.(type) {
	case *userInfoType:
		u.UID = m.uid
		u.Name = m.name
		u.EMail = m.email
	case map[string]interface{}, *map[string]interface{}:
		return nil
	default:
		return errors.Errorf("unsupported type %T", v)
	}
	return nil
}

func (m mockOIDCToken) GetExpiry() time.Time {
	return time.Now().Add(time.Minute * 2)
}
