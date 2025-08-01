package auth

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokensource"
	authMocks "github.com/stackrox/rox/pkg/cloudproviders/gcp/auth/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestTokenManager asserts the credentials get is called twice when we expire the token.
func TestTokenManager(t *testing.T) {
	controller := gomock.NewController(t)

	mockCredManager := authMocks.NewMockCredentialsManager(controller)
	// Return error here so we don't have to mock the google credentials token source.
	dummyErr := errors.New("dummy error")
	mockCredManager.EXPECT().GetCredentials(gomock.Any()).Return(nil, dummyErr).Times(2)

	ts := tokensource.NewReuseTokenSourceWithInvalidate(
		&CredentialManagerTokenSource{credManager: mockCredManager},
		time.Minute,
	)
	manager := &stsTokenManagerImpl{credManager: mockCredManager, tokenSource: ts}

	_, err := ts.Token()
	assert.ErrorIs(t, err, dummyErr)
	manager.invalidateToken()
	_, err = ts.Token()
	assert.ErrorIs(t, err, dummyErr)
}
