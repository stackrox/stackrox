package service

import (
	"testing"

	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetSource(t *testing.T) {
	// Smoke test the token issuer source creation
	assert.NotNil(t, getTokenSource())
}

func TestGetTokenIssuer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockIssuerFactory := tokensMocks.NewMockIssuerFactory(mockCtrl)
	tokenAuthProvider := getTokenSource()
	mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
	mockIssuerFactory.EXPECT().
		CreateIssuer(tokenAuthProvider).
		Times(1).
		Return(mockIssuer, nil)
	assert.NotPanics(t, func() {
		issuer := getTokenIssuer(mockIssuerFactory)
		assert.Equal(t, mockIssuer, issuer)
	})
}
