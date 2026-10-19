package datastore

import (
	"testing"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSeedFirstInstall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatIntegrationID
	mockStore.EXPECT().Get(gomock.Any(), id).Return(nil, false, nil).Times(1)
	mockStore.EXPECT().Upsert(gomock.Any(), signatures.DefaultRedHatSignatureIntegration).Return(nil).Times(1)

	seedRedHatDefaultSignatureIntegration(mockStore)
}

func TestSeedSubsequentStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatIntegrationID
	mockStore.EXPECT().Get(gomock.Any(), id).Return(signatures.DefaultRedHatSignatureIntegration, true, nil).Times(1)

	seedRedHatDefaultSignatureIntegration(mockStore)
}

func TestStartKeyBundleUpdaterOfflineMode(t *testing.T) {
	t.Setenv(env.OfflineModeEnv.EnvVar(), "true")
	t.Setenv(env.RedHatSigningKeyBundleURL.EnvVar(), "https://example.com/keys.json")

	old := bundleUpdater
	defer func() { bundleUpdater = old }()
	bundleUpdater = nil

	startKeyBundleUpdater()
	assert.Nil(t, bundleUpdater, "updater should not start in offline mode")
}

func TestStartKeyBundleUpdaterOnlineWithURL(t *testing.T) {
	t.Setenv(env.OfflineModeEnv.EnvVar(), "false")
	t.Setenv(env.RedHatSigningKeyBundleURL.EnvVar(), "https://example.com/keys.json")

	old := bundleUpdater
	defer func() {
		if bundleUpdater != nil {
			bundleUpdater.Stop()
		}
		bundleUpdater = old
	}()
	bundleUpdater = nil

	startKeyBundleUpdater()
	assert.NotNil(t, bundleUpdater, "updater should start in online mode with URL configured")
}

func TestStartKeyBundleUpdaterOnlineWithoutURL(t *testing.T) {
	t.Setenv(env.OfflineModeEnv.EnvVar(), "false")
	t.Setenv(env.RedHatSigningKeyBundleURL.EnvVar(), "")

	old := bundleUpdater
	defer func() { bundleUpdater = old }()
	bundleUpdater = nil

	startKeyBundleUpdater()
	assert.Nil(t, bundleUpdater, "updater should not start without URL")
}
