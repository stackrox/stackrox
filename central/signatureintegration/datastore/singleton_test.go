package datastore

import (
	"testing"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSeedFirstInstall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatSignatureIntegration.GetId()
	mockStore.EXPECT().Get(gomock.Any(), id).Return(nil, false, nil).Times(1)
	mockStore.EXPECT().Upsert(gomock.Any(), signatures.DefaultRedHatSignatureIntegration).Return(nil).Times(1)

	seedRedHatSignatureIntegration(mockStore)
}

func TestStartKeyBundleWatcherDisabled(t *testing.T) {
	t.Setenv("ROX_DISABLE_REDHAT_SIGNING_KEY_BUNDLE_WATCHER", "true")

	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	old := bundleWatcher
	defer func() { bundleWatcher = old }()
	bundleWatcher = nil

	startKeyBundleWatcher(mockStore)
	assert.Nil(t, bundleWatcher)
}

func TestSeedSubsequentStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatSignatureIntegration.GetId()
	mockStore.EXPECT().Get(gomock.Any(), id).Return(signatures.DefaultRedHatSignatureIntegration, true, nil).Times(1)
	// Upsert must NOT be called — integration already exists.

	seedRedHatSignatureIntegration(mockStore)
}
