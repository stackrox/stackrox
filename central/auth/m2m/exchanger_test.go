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
	halfAnHour           = "30m"
	oneMinute            = "1m"
	thirtyMinutes        = "30m"
	oneHour              = "1h"
	oneHourThirtyMinutes = "1h30m"
	twoHours             = "2h"
	notADuration         = "60"

	configID     = "test-config-id"
	testConfigID = "test-config-id"
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

// newOIDCDiscoveryServer serves a minimal OIDC discovery document, so that creating the
// OIDC provider within tokenVerifierFromConfigData succeeds without reaching out to a
// real issuer. The server is automatically closed when the test finishes.
func newOIDCDiscoveryServer(t *testing.T) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":   server.URL,
				"jwks_uri": server.URL + "/keys",
			})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newM2MConfig(
	issuer string,
	tokenExpirationDuration string,
	configType storage.AuthMachineToMachineConfig_Type,
	audience string,
) *storage.AuthMachineToMachineConfig {
	return &storage.AuthMachineToMachineConfig{
		Id:                      testConfigID,
		Type:                    configType,
		Issuer:                  issuer,
		TokenExpirationDuration: tokenExpirationDuration,
		Audience:                audience,
	}
}

// expectSuccessfulIssuerCreation prepares the given issuer factory mock to successfully
// create an issuer for the given issuer URL.
func expectSuccessfulIssuerCreation(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory, issuerURL string) {
	issuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(source tokens.Source, _ ...tokens.Option) (tokens.Issuer, error) {
			assert.Equal(t, issuerURL, source.ID())
			return tokensMocks.NewMockIssuer(gomock.NewController(t)), nil
		})
}

func TestNewTokenExchanger(t *testing.T) {
	oidcDiscoveryServer := newOIDCDiscoveryServer(t)

	cases := map[string]struct {
		configs              []*storage.AuthMachineToMachineConfig
		prepareIssuerFactory func(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory)
		expectedTTL          time.Duration
		expectedError        error
		expectedErrorTexts   []string
	}{
		"Multiple consistent configs successfully create an exchanger": {
			configs: []*storage.AuthMachineToMachineConfig{
				newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-1"),
				newM2MConfig(oidcDiscoveryServer.URL, halfAnHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-2"),
			},
			prepareIssuerFactory: func(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory) {
				expectSuccessfulIssuerCreation(t, issuerFactory, oidcDiscoveryServer.URL)
			},
			// The exchanger is expected to use the smallest TTL across all configs.
			expectedTTL: 30 * time.Minute,
		},
		"Single config list successfully creates an exchanger": {
			configs: []*storage.AuthMachineToMachineConfig{
				newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-1"),
			},
			prepareIssuerFactory: func(t *testing.T, issuerFactory *tokensMocks.MockIssuerFactory) {
				expectSuccessfulIssuerCreation(t, issuerFactory, oidcDiscoveryServer.URL)
			},
			expectedTTL: 1 * time.Hour,
		},
		"Empty config list is rejected": {
			configs:            nil,
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"creating token exchanger", "no auth machine to machine configuration"},
		},
		"Issuer mismatch is rejected": {
			configs: []*storage.AuthMachineToMachineConfig{
				newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-1"),
				newM2MConfig(oidcDiscoveryServer.URL+"/other", oneHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-2"),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"creating token exchanger", "issuer mismatch"},
		},
		"Type mismatch is rejected": {
			configs: []*storage.AuthMachineToMachineConfig{
				newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-1"),
				newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GITHUB_ACTIONS, "audience-2"),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"creating token exchanger", "type mismatch"},
		},
		"Issuer and type mismatches are both rejected": {
			configs: []*storage.AuthMachineToMachineConfig{
				newM2MConfig(oidcDiscoveryServer.URL, oneHour, storage.AuthMachineToMachineConfig_GENERIC, "audience-1"),
				newM2MConfig(oidcDiscoveryServer.URL+"/other", oneHour, storage.AuthMachineToMachineConfig_GITHUB_ACTIONS, "audience-2"),
			},
			expectedError:      errox.InvalidArgs,
			expectedErrorTexts: []string{"creating token exchanger", "issuer mismatch", "type mismatch"},
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

			exchanger, err := newTokenExchanger(it.Context(), c.configs, roleDS, issuerFactory)

			expectError := c.expectedError != nil || len(c.expectedErrorTexts) > 0
			if c.expectedError != nil {
				assert.ErrorIs(it, err, c.expectedError)
			}
			for _, expectedErrorText := range c.expectedErrorTexts {
				assert.ErrorContains(it, err, expectedErrorText)
			}
			if !expectError {
				assert.NoError(it, err)
				assert.NotNil(it, exchanger)
				assert.NotNil(it, exchanger.Provider())
				assert.Equal(it, c.expectedTTL, exchanger.GetTokenExpirationDuration())
				assert.Equal(it, c.configs[0].GetType(), exchanger.GetType())
				assert.Len(it, exchanger.Configs(), len(c.configs))
				for i, config := range c.configs {
					protoassert.Equal(it, config, exchanger.Configs()[i])
				}
				protoassert.Equal(it, c.configs[0], exchanger.Config())
			} else {
				assert.Nil(it, exchanger)
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

func TestConfig(t *testing.T) {
	newConfig := func(id string) *storage.AuthMachineToMachineConfig {
		return &storage.AuthMachineToMachineConfig{
			Id:                      id,
			Type:                    storage.AuthMachineToMachineConfig_GENERIC,
			Issuer:                  "https://issuer.example.com",
			TokenExpirationDuration: oneHour,
		}
	}

	cases := map[string]struct {
		configs        []*storage.AuthMachineToMachineConfig
		expectedConfig *storage.AuthMachineToMachineConfig
	}{
		"No config returns nil": {
			configs:        nil,
			expectedConfig: nil,
		},
		"Single config returns that config": {
			configs:        []*storage.AuthMachineToMachineConfig{newConfig("config-1")},
			expectedConfig: newConfig("config-1"),
		},
		"Multiple configs return the first config": {
			configs: []*storage.AuthMachineToMachineConfig{
				newConfig("config-1"),
				newConfig("config-2"),
				newConfig("config-3"),
			},
			expectedConfig: newConfig("config-1"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			exchanger := &machineToMachineTokenExchanger{configs: c.configs}
			if c.expectedConfig == nil {
				assert.Nil(t, exchanger.Config())
				return
			}
			protoassert.Equal(t, c.expectedConfig, exchanger.Config())
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
		prepareVerifier       func(t *testing.T, verifier *MocktokenVerifier)
		prepareRoleDS         func(t *testing.T, roleDS *roleMocks.MockDataStore)
		prepareIssuer         func(t *testing.T, issuer *tokensMocks.MockIssuer)
		prepareClaimExtractor func(t *testing.T, ctrl *gomock.Controller) claimExtractor
		expectedToken         string
		expectedError         error
		expectedErrorText     string
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
		"Claim extractor failure when extracting ID token claims is propagated": {
			prepareVerifier: func(t *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), gomock.Any()).
					Return(newIDToken("test-subject"), nil)
				verifier.EXPECT().Audience().Return("")
			},
			prepareClaimExtractor: func(_ *testing.T, ctrl *gomock.Controller) claimExtractor {
				claimExtractor := NewMockclaimExtractor(ctrl)
				claimExtractor.EXPECT().ExtractClaims(gomock.Any()).Return(nil, assert.AnError)
				return claimExtractor
			},
			expectedError:     errox.NoCredentials,
			expectedErrorText: "extracting claims from ID token",
		},
		"Claim extractor failure when extracting Rox claims is propagated": {
			prepareVerifier: func(t *testing.T, verifier *MocktokenVerifier) {
				verifier.EXPECT().VerifyIDToken(gomock.Any(), gomock.Any()).
					Return(newIDToken("test-subject"), nil)
				verifier.EXPECT().Audience().Return("")
			},
			prepareRoleDS: func(t *testing.T, roleDS *roleMocks.MockDataStore) {
				roleDS.EXPECT().GetAndResolveRole(gomock.Any(), "Admin").
					Return(&testResolvedRole{name: "Admin"}, nil)
			},
			prepareClaimExtractor: func(_ *testing.T, ctrl *gomock.Controller) claimExtractor {
				claimExtractor := NewMockclaimExtractor(ctrl)
				claimExtractor.EXPECT().ExtractClaims(gomock.Any()).
					Return(map[string][]string{"sub": {"test-subject"}}, nil)
				claimExtractor.EXPECT().ExtractRoxClaims(gomock.Any()).
					Return(tokens.RoxClaims{}, assert.AnError)
				return claimExtractor
			},
			expectedError:     errox.NoCredentials,
			expectedErrorText: "creating claims for Central token",
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
			// Unless a case provides its own claim extractor (e.g. to simulate failures),
			// the one derived from the config type is used.
			roxClaimExtractor := newClaimExtractorForConfigType(config.GetType())
			if c.prepareClaimExtractor != nil {
				roxClaimExtractor = c.prepareClaimExtractor(it, ctrl)
			}
			exchanger := &machineToMachineTokenExchanger{
				configs:           []*storage.AuthMachineToMachineConfig{config},
				verifiers:         []tokenVerifier{verifier},
				provider:          newProvider(config.GetId(), config.GetType().String(), providerRoleMapper),
				issuer:            issuer,
				roleDS:            roleDS,
				roxClaimExtractor: roxClaimExtractor,
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
