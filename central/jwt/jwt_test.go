package jwt

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	testRoxIssuerID = "test-issuer"
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
			token: tokenWithIssuer(testRoxIssuerID),
			mockRoxValidator: func(ctx context.Context, token string) (*tokens.TokenInfo, error) {
				assert.Equal(t, tokenWithIssuer(testRoxIssuerID), token)
				return &tokens.TokenInfo{Token: token}, nil
			},
			expectedTokenToPass: tokenWithIssuer(testRoxIssuerID),
		},
		"non-rox issuer with exchanger": {
			token: tokenWithIssuer("other-issuer"),
			mockRoxValidator: func(ctx context.Context, token string) (*tokens.TokenInfo, error) {
				assert.Equal(t, "exchanged-token", token)
				return &tokens.TokenInfo{Token: token}, nil
			},
			setupExchangerSet: func(mes *mocks.MockTokenExchangerSet) {
				mockExchanger := mocks.NewMockTokenExchanger(mockCtrl)
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
				TokenExchangerSet: exchangerSet,
				roxValidator:      c.mockRoxValidator,
				issuerID:          testRoxIssuerID,
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
