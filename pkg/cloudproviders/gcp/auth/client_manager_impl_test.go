package auth

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokensource"
	authMocks "github.com/stackrox/rox/pkg/cloudproviders/gcp/auth/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestTokenManager asserts the credentials get is called twice when we expire the token.
func TestTokenManager(t *testing.T) {
	t.Parallel()
	controller := gomock.NewController(t)

	mockCredManager := authMocks.NewMockCredentialsManager(controller)
	// Return error here so we don't have to mock the google credentials token source.
	dummyErr := errors.New("dummy error")
	mockCredManager.EXPECT().GetCredentials(gomock.Any()).Return(nil, dummyErr).Times(2)

	ts := tokensource.NewReuseTokenSourceWithForceRefresh(&CredentialManagerTokenSource{credManager: mockCredManager})
	manager := &stsTokenManagerImpl{credManager: mockCredManager, tokenSource: ts}

	_, err := ts.Token()
	assert.ErrorIs(t, err, dummyErr)
	manager.expireToken()
	_, err = ts.Token()
	assert.ErrorIs(t, err, dummyErr)
}
