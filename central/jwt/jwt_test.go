package jwt

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	errTest = errors.New("test error")
)

type mockRoxValidator func(ctx context.Context, token string) (*tokens.TokenInfo, error)

func (v mockRoxValidator) Validate(ctx context.Context, token string) (*tokens.TokenInfo, error) {
	if v != nil {
		return v(ctx, token)
	}
	return nil, nil
}

func tokenWithIssuer(issuer string) string {
	claims := fmt.Sprintf(`{"iss":"%s"}`, issuer)
	return fmt.Sprintf("%s.%s.signature",
		base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)),
		base64.RawURLEncoding.EncodeToString([]byte(claims)))
}

func TestM2MValidator(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)

	cases := map[string]struct {
		token               string
		mockRoxValidator    mockRoxValidator
		setupExchangerSet   func(*mocks.MockTokenExchangerSet)
		expectedTokenToPass string
		expectedErr         error
	}{
		"rox issuer": {
			token: tokenWithIssuer(roxIssuer),
			mockRoxValidator: func(ctx context.Context, token string) (*tokens.TokenInfo, error) {
				assert.Equal(t, tokenWithIssuer(roxIssuer), token)
				return &tokens.TokenInfo{Token: token}, nil
			},
			expectedTokenToPass: tokenWithIssuer(roxIssuer),
		},
		"non-rox issuer with exchanger": {
			token: tokenWithIssuer("other-issuer"),
			mockRoxValidator: func(ctx context.Context, token string) (*tokens.TokenInfo, error) {
				assert.Equal(t, "exchanged-token", token)
				return &tokens.TokenInfo{Token: token}, nil
			},
			setupExchangerSet: func(mes *mocks.MockTokenExchangerSet) {
				mockExchanger := mocks.NewMockTokenExchanger(mockCtrl)
				amtmc := &storage.AuthMachineToMachineConfig{}
				amtmc.SetTokenExpirationDuration("1h")
				mockExchanger.EXPECT().Config().Times(1).Return(amtmc)
				mockExchanger.EXPECT().ExchangeToken(gomock.Any(), tokenWithIssuer("other-issuer")).
					Return("exchanged-token", nil)
				mes.EXPECT().GetTokenExchanger("other-issuer").
					Return(mockExchanger, true)
			},
			expectedTokenToPass: "exchanged-token",
		},
		"non-rox issuer, no exchanger": {
			token: tokenWithIssuer("unknown-issuer"),
			setupExchangerSet: func(mes *mocks.MockTokenExchangerSet) {
				mes.EXPECT().GetTokenExchanger("unknown-issuer").
					Return(nil, false)
			},
			expectedErr: errox.NoCredentials,
		},
		"issuer mismatch exchanger error": {
			token: tokenWithIssuer("failing-issuer"),
			setupExchangerSet: func(mes *mocks.MockTokenExchangerSet) {
				mockExchanger := mocks.NewMockTokenExchanger(mockCtrl)
				amtmc := &storage.AuthMachineToMachineConfig{}
				amtmc.SetTokenExpirationDuration("1h")
				mockExchanger.EXPECT().Config().Times(1).Return(amtmc)
				mockExchanger.EXPECT().ExchangeToken(gomock.Any(), tokenWithIssuer("failing-issuer")).
					Return("", errTest)
				mes.EXPECT().GetTokenExchanger("failing-issuer").
					Return(mockExchanger, true)
			},
			expectedErr: errTest,
		},
		"invalid token": {
			token:       "invalid-token",
			expectedErr: errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			exchangerSet := mocks.NewMockTokenExchangerSet(mockCtrl)
			if c.setupExchangerSet != nil {
				c.setupExchangerSet(exchangerSet)
			}
			exchangerSet.EXPECT().HasExchangersConfigured().Times(1).Return(true)

			validator := &m2mValidator{
				TokenExchangerSet:    exchangerSet,
				roxValidator:         c.mockRoxValidator,
				exchangedTokensCache: make(map[string]expiringcache.Cache[string, string]),
			}

			token, err := validator.Validate(ctx, c.token)

			if assert.ErrorIs(t, err, c.expectedErr) {
				if c.expectedTokenToPass != "" && assert.NotNil(t, token) {
					assert.Equal(t, c.expectedTokenToPass, token.Token)
				}
			}
		})
	}
}

func TestTokenCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mes := mocks.NewMockTokenExchangerSet(mockCtrl)

	roxValidator := func(ctx context.Context, token string) (*tokens.TokenInfo, error) {
		assert.Equal(t, tokenWithIssuer(roxIssuer), token)
		return &tokens.TokenInfo{Token: token}, nil
	}

	validator := &m2mValidator{
		TokenExchangerSet:    mes,
		roxValidator:         mockRoxValidator(roxValidator),
		exchangedTokensCache: make(map[string]expiringcache.Cache[string, string]),
	}

	t.Run("1h expiration", func(t *testing.T) {
		mockExchanger := mocks.NewMockTokenExchanger(mockCtrl)
		amtmc := &storage.AuthMachineToMachineConfig{}
		amtmc.SetTokenExpirationDuration("1h")
		mockExchanger.EXPECT().Config().Times(1).
			Return(amtmc)

		mes.EXPECT().GetTokenExchanger("1h-issuer").
			Times(2).
			Return(mockExchanger, true)
		mockExchanger.EXPECT().ExchangeToken(gomock.Any(), tokenWithIssuer("1h-issuer")).
			Times(1). // Exchange once.
			Return("exchanged-token", nil)

		// Exchange and cache:
		token, err := validator.exchange(context.Background(), "1h-issuer", tokenWithIssuer("1h-issuer"))
		assert.NoError(t, err)
		assert.Equal(t, "exchanged-token", token)

		// No exchange, return cached:
		token, err = validator.exchange(context.Background(), "1h-issuer", tokenWithIssuer("1h-issuer"))
		assert.NoError(t, err)
		assert.Equal(t, "exchanged-token", token)
	})

	t.Run("1m expiration", func(t *testing.T) {
		mockExchanger := mocks.NewMockTokenExchanger(mockCtrl)
		amtmc := &storage.AuthMachineToMachineConfig{}
		amtmc.SetTokenExpirationDuration("1m")
		mockExchanger.EXPECT().Config().Times(1).
			Return(amtmc)

		mes.EXPECT().GetTokenExchanger("1m-issuer").
			Times(2).
			Return(mockExchanger, true)
		mockExchanger.EXPECT().ExchangeToken(gomock.Any(), tokenWithIssuer("1m-issuer")).
			Times(2). // Exchange twice.
			Return("exchanged-token", nil)

		// Exchange and don't cache:
		token, err := validator.exchange(context.Background(), "1m-issuer", tokenWithIssuer("1m-issuer"))
		assert.NoError(t, err)
		assert.Equal(t, "exchanged-token", token)

		// Exchange again:
		token, err = validator.exchange(context.Background(), "1m-issuer", tokenWithIssuer("1m-issuer"))
		assert.NoError(t, err)
		assert.Equal(t, "exchanged-token", token)
	})
}
