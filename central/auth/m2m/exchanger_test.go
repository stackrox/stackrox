package m2m

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/protoassert"
	"go.uber.org/mock/gomock"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

const (
	oneMinute            = "1m"
	thirtyMinutes        = "30m"
	oneHour              = "1h"
	oneHourThirtyMinutes = "1h30m"
	twoHours             = "2h"
	notADuration         = "60"

	configID = "test-config-id"
)

func TestValidateConfigConsistency(t *testing.T) {
	newConfig := func(issuer string, configType storage.AuthMachineToMachineConfig_Type) *storage.AuthMachineToMachineConfig {
		return &storage.AuthMachineToMachineConfig{
			Issuer: issuer,
			Type:   configType,
		}
	}

	cases := map[string]struct {
		configs            []*storage.AuthMachineToMachineConfig
		expectedError      error
		expectedErrorTexts []string
	}{
		"No config is rejected": {
			configs:       nil,
			expectedError: errox.InvalidArgs,
		},
		"Single config is valid": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
			},
		},
		"Multiple consistent configs are valid": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
			},
		},
		"Issuer mismatch is rejected": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
				newConfig("https://other-issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"issuer mismatch"},
		},
		"Type mismatch is rejected": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GITHUB_ACTIONS),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"type mismatch"},
		},
		"Issuer and type mismatches are both reported": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
				newConfig("https://other-issuer.example.com", storage.AuthMachineToMachineConfig_GITHUB_ACTIONS),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"issuer mismatch", "type mismatch"},
		},
		"Only the first config is used as reference for mismatch reporting": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
				newConfig("https://issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
				newConfig("https://third-issuer.example.com", storage.AuthMachineToMachineConfig_GENERIC),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"issuer mismatch", "https://issuer.example.com", "https://third-issuer.example.com"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateConfigConsistency(c.configs)

			if c.expectedError == nil && len(c.expectedErrorTexts) == 0 {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, c.expectedError)
			for _, expectedErrorText := range c.expectedErrorTexts {
				assert.ErrorContains(t, err, expectedErrorText)
			}
		})
	}
}

func TestNewTokenExchanger(t *testing.T) {
	// Serve a minimal OIDC discovery document, so that creating the OIDC provider within
	// tokenVerifierFromConfigData succeeds without reaching out to a real issuer.
	var oidcDiscoveryServer *httptest.Server
	oidcDiscoveryServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":   oidcDiscoveryServer.URL,
				"jwks_uri": oidcDiscoveryServer.URL + "/keys",
			})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(oidcDiscoveryServer.Close)

	newM2MConfig := func(
		issuer string,
		tokenExpirationDuration string,
		configType storage.AuthMachineToMachineConfig_Type,
	) *storage.AuthMachineToMachineConfig {
		return &storage.AuthMachineToMachineConfig{
			Id:                      configID,
			Type:                    configType,
			Issuer:                  issuer,
			TokenExpirationDuration: tokenExpirationDuration,
		}
	}

	const oneHour = "1h"
	const notADuration = "60"

	cases := map[string]struct {
		config               *storage.AuthMachineToMachineConfig
		prepareIssuerFactory func(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory)
		expectedError        error
		expectedErrorText    string
	}{
		"Valid config (generic) successfully creates an exchanger": {
			config: newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC),
			prepareIssuerFactory: func(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory) {
				issuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).
					DoAndReturn(func(source tokens.Source, _ ...tokens.Option) (tokens.Issuer, error) {
						assert.Equal(t, oidcDiscoveryServer.URL, source.ID())
						return tokensMocks.NewMockIssuer(gomock.NewController(t)), nil
					})
			},
		},
		"Valid config (k8s service account) successfully creates an exchanger": {
			config: newM2MConfig(
				oidcDiscoveryServer.URL,
				oneHour,
				storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
			),
			prepareIssuerFactory: func(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory) {
				issuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).
					DoAndReturn(func(source tokens.Source, _ ...tokens.Option) (tokens.Issuer, error) {
						assert.Equal(t, oidcDiscoveryServer.URL, source.ID())
						return tokensMocks.NewMockIssuer(gomock.NewController(t)), nil
					})
			},
		},
		"Invalid token expiration duration is rejected": {
			config: newM2MConfig(
				oidcDiscoveryServer.URL,
				notADuration,
				storage.AuthMachineToMachineConfig_GENERIC,
			),
			expectedError:     errox.InvalidArgs,
			expectedErrorText: "parsing token expiration duration",
		},
		"Issuer without valid OIDC discovery endpoint is rejected": {
			config: newM2MConfig(
				oidcDiscoveryServer.URL+"/nonexistent",
				oneHour,
				storage.AuthMachineToMachineConfig_GENERIC,
			),
			expectedErrorText: "creating OIDC provider for issuer",
		},
		"Issuer factory failure is propagated": {
			config: newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC),
			prepareIssuerFactory: func(_ *testing.T, issuerFactory *tokensMocks.MockIssuerFactory) {
				issuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
			expectedErrorText: "creating token issuer",
		},
	}

	for name, c := range cases {
		t.Run(name, func(it *testing.T) {
			ctrl := gomock.NewController(t)
			issuerFactory := tokensMocks.NewMockIssuerFactory(ctrl)
			if c.prepareIssuerFactory != nil {
				c.prepareIssuerFactory(t, issuerFactory)
			}
			roleDS := roleMocks.NewMockDataStore(ctrl)

			exchanger, err := newTokenExchanger(it.Context(), c.config, roleDS, issuerFactory)

			expectError := c.expectedError != nil || c.expectedErrorText != ""
			if c.expectedError != nil {
				assert.ErrorIs(it, err, c.expectedError)
			}
			if c.expectedErrorText != "" {
				assert.ErrorContains(it, err, c.expectedErrorText)
			}
			if !expectError {
				assert.NoError(it, err)
				assert.NotNil(it, exchanger)
				protoassert.Equal(it, c.config, exchanger.Config())
				assert.NotNil(it, exchanger.Provider())
			}
		})
	}
}

func TestGetTTLFromConfigs(t *testing.T) {
	newConfig := func(tokenExpirationDuration string) *storage.AuthMachineToMachineConfig {
		return &storage.AuthMachineToMachineConfig{
			TokenExpirationDuration: tokenExpirationDuration,
		}
	}

	cases := map[string]struct {
		configs       []*storage.AuthMachineToMachineConfig
		expectedTTL   time.Duration
		expectedError error
	}{
		"No config is rejected": {
			configs:       nil,
			expectedError: errox.InvalidArgs,
		},
		"Valid single config with minutes": {
			configs:     []*storage.AuthMachineToMachineConfig{newConfig(oneMinute)},
			expectedTTL: 1 * time.Minute,
		},
		"Valid single config with hours": {
			configs:     []*storage.AuthMachineToMachineConfig{newConfig(oneHour)},
			expectedTTL: 1 * time.Hour,
		},
		"Valid single config with compound duration": {
			configs:     []*storage.AuthMachineToMachineConfig{newConfig(oneHourThirtyMinutes)},
			expectedTTL: 90 * time.Minute,
		},
		"Invalid single config with raw integer value": {
			// notADuration = "60"
			configs:       []*storage.AuthMachineToMachineConfig{newConfig(notADuration)},
			expectedError: errox.InvalidArgs,
		},
		"Invalid single config with empty duration": {
			configs:       []*storage.AuthMachineToMachineConfig{newConfig("")},
			expectedError: errox.InvalidArgs,
		},
		"Invalid single config with unknown unit": {
			configs:       []*storage.AuthMachineToMachineConfig{newConfig("1w")},
			expectedError: errox.InvalidArgs,
		},
		"Multiple valid configs return the smallest TTL": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig(twoHours),
				newConfig(thirtyMinutes),
				newConfig(oneHour),
			},
			expectedTTL: 30 * time.Minute,
		},
		"Multiple valid configs with a repeated TTL": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig(oneHour),
				newConfig(oneHour),
			},
			expectedTTL: 1 * time.Hour,
		},
		"Multiple configs with an invalid later config are rejected": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig(oneHour),
				newConfig(notADuration),
			},
			expectedError: errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(it *testing.T) {
			ttl, err := getTTLFromConfigs(c.configs)
			assert.ErrorIs(it, err, c.expectedError)
			// In error cases, expectedTTL is unset and thus holds the zero value,
			// which is what the function returns alongside the error.
			assert.Equal(it, c.expectedTTL, ttl)
		})
	}
}

func TestExchangeToken(t *testing.T) {
	newIDToken := func(subject string) *IDToken {
		return &IDToken{
			Subject:  subject,
			Audience: []string{"test-audience"},
			Claims: func(into any) error {
				claims, ok := into.(*map[string]interface{})
				if !ok {
					return assert.AnError
				}
				*claims = map[string]interface{}{"email": "sample@stackrox.io"}
				return nil
			},
		}
	}

	newExchangerConfig := func() *storage.AuthMachineToMachineConfig {
		return &storage.AuthMachineToMachineConfig{
			Id:   configID,
			Type: storage.AuthMachineToMachineConfig_GENERIC,
			Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{Key: "sub", ValueExpression: "test-.*", Role: "Admin"},
			},
		}
	}

	cases := map[string]struct {
		prepareVerifier   func(t *testing.T, verifier *MocktokenVerifier)
		prepareRoleDS     func(t *testing.T, roleDS *roleMocks.MockDataStore)
		prepareIssuer     func(t *testing.T, issuer *tokensMocks.MockIssuer)
		expectedToken     string
		expectedError     error
		expectedErrorText string
	}{
		"Valid ID token is exchanged for a Central token": {
			prepareVerifier: func(t *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), "test-raw-id-token").
					Return(newIDToken("test-subject"), nil) // #nosec G101 -- test data
				verifier.EXPECT().Audience().Return("")
			},
			prepareRoleDS: func(t *testing.T, roleDS *roleMocks.MockDataStore) {
				roleDS.EXPECT().GetAndResolveRole(gomock.Any(), "Admin").
					Return(&testResolvedRole{name: "Admin"}, nil)
			},
			prepareIssuer: func(t *testing.T, issuer *tokensMocks.MockIssuer) {
				issuer.EXPECT().Issue(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, claims tokens.RoxClaims, _ ...tokens.Option) (*tokens.TokenInfo, error) {
						assert.Equal(t, "test-subject|sample@stackrox.io", claims.ExternalUser.UserID)
						return &tokens.TokenInfo{Token: "test-rox-token"}, nil // #nosec G101 -- test data
					})
			},
			expectedToken: "test-rox-token",
		},
		"Invalid ID token is rejected": {
			prepareVerifier: func(_ *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			},
			expectedError:     errox.NoCredentials,
			expectedErrorText: "ID token is invalid",
		},
		"ID token with invalid claims is rejected": {
			prepareVerifier: func(t *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), gomock.Any()).
					Return(&IDToken{
						Subject: "test-subject",
						Claims:  func(any) error { return assert.AnError },
					}, nil)
				verifier.EXPECT().Audience().Return("")
			},
			expectedError:     errox.NoCredentials,
			expectedErrorText: "extracting claims from ID token",
		},
		"ID token without matching role mapping is rejected": {
			prepareVerifier: func(t *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), gomock.Any()).
					Return(newIDToken("unknown-subject"), nil)
				verifier.EXPECT().Audience().Return("")
			},
			expectedError:     errox.NoCredentials,
			expectedErrorText: "resolving roles for id token",
		},
		"Token issuing failure is propagated": {
			prepareVerifier: func(t *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), gomock.Any()).
					Return(newIDToken("test-subject"), nil)
				verifier.EXPECT().Audience().Return("")
			},
			prepareRoleDS: func(t *testing.T, roleDS *roleMocks.MockDataStore) {
				roleDS.EXPECT().GetAndResolveRole(gomock.Any(), "Admin").
					Return(&testResolvedRole{name: "Admin"}, nil)
			},
			prepareIssuer: func(_ *testing.T, issuer *tokensMocks.MockIssuer) {
				issuer.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			},
			expectedError:     errox.NoCredentials,
			expectedErrorText: "issuing Central token",
		},
	}

	for name, c := range cases {
		t.Run(name, func(it *testing.T) {
			ctrl := gomock.NewController(it)
			verifier := NewMocktokenVerifier(ctrl)
			if c.prepareVerifier != nil {
				c.prepareVerifier(it, verifier)
			}
			roleDS := roleMocks.NewMockDataStore(ctrl)
			if c.prepareRoleDS != nil {
				c.prepareRoleDS(it, roleDS)
			}
			issuer := tokensMocks.NewMockIssuer(ctrl)
			if c.prepareIssuer != nil {
				c.prepareIssuer(it, issuer)
			}

			config := newExchangerConfig()
			mappings := parseMappings(config.GetMappings(), "")
			providerRoleMapper := newRoleMapperFromParsedMappings(mappings, roleDS)
			exchanger := &machineToMachineTokenExchanger{
				configs:           []*storage.AuthMachineToMachineConfig{config},
				verifiers:         []tokenVerifier{verifier},
				provider:          newProvider(config.GetId(), config.GetType().String(), providerRoleMapper),
				issuer:            issuer,
				roleDS:            roleDS,
				roxClaimExtractor: newClaimExtractorForConfigType(config.GetType()),
				mappings:          mappings,
			}

			token, err := exchanger.ExchangeToken(it.Context(), "test-raw-id-token")

			if c.expectedError != nil {
				assert.ErrorIs(it, err, c.expectedError)
			}
			if c.expectedErrorText != "" {
				assert.ErrorContains(it, err, c.expectedErrorText)
			}
			if c.expectedError == nil && c.expectedErrorText == "" {
				assert.NoError(it, err)
				assert.Equal(it, c.expectedToken, token)
			}
		})
	}
}
