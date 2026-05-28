package datastore

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func redHatIntegrationMatcher() gomock.Matcher {
	return gomock.Cond(func(x any) bool {
		si, ok := x.(*storage.SignatureIntegration)
		return ok && si.GetId() == signatures.DefaultRedHatSignatureIntegration.GetId()
	})
}

func validBundleJSON() []byte {
	return []byte(fmt.Sprintf(`{"keys": [{"name": "test-key-1", "pem": %q}]}`, testPublicKeyPEM))
}

func validBundleJSON2Keys() []byte {
	return []byte(fmt.Sprintf(`{"keys": [{"name": "test-key-1", "pem": %q}, {"name": "test-key-2", "pem": %q}]}`,
		testPublicKeyPEM, testPublicKeyPEM2))
}

func TestHandlerValidBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(1)

	handler := keyBundleHandler(mockStore)
	err := handler(validBundleJSON())
	assert.NoError(t, err)
}

func TestHandlerTwoKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(1)

	handler := keyBundleHandler(mockStore)
	err := handler(validBundleJSON2Keys())
	assert.NoError(t, err)
}

func TestHandlerInvalidBundleReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	handler := keyBundleHandler(mockStore)
	err := handler([]byte(`{"keys": []}`))
	assert.NoError(t, err, "parse errors must return nil to suppress retry")
}

func TestHandlerMalformedJSONReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	handler := keyBundleHandler(mockStore)
	err := handler([]byte(`{not json`))
	assert.NoError(t, err, "parse errors must return nil to suppress retry")
}

func TestHandlerUpsertErrorReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().
		Upsert(gomock.Any(), redHatIntegrationMatcher()).
		Return(errors.New("transient DB error")).
		Times(1)

	handler := keyBundleHandler(mockStore)
	err := handler(validBundleJSON())
	require.Error(t, err, "upsert errors must be returned to enable retry")
	assert.Contains(t, err.Error(), "transient DB error")
}

func TestHandlerUpsertRetryOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	firstCall := mockStore.EXPECT().
		Upsert(gomock.Any(), redHatIntegrationMatcher()).
		Return(errors.New("transient DB error")).
		Times(1)
	mockStore.EXPECT().
		Upsert(gomock.Any(), redHatIntegrationMatcher()).
		Return(nil).
		Times(1).
		After(firstCall)

	handler := keyBundleHandler(mockStore)

	err := handler(validBundleJSON())
	assert.Error(t, err, "first call should fail")

	err = handler(validBundleJSON())
	assert.NoError(t, err, "second call should succeed")
}
