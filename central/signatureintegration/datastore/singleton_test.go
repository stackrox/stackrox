package datastore

import (
	"fmt"
	"os"
	"testing"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func writeTempBundle(t *testing.T) {
	t.Helper()
	bundleData := fmt.Sprintf(`{"keys": [{"name": "test-key", "pem": %q}]}`, testPublicKeyPEM)
	dir := t.TempDir()
	bundlePath := dir + "/bundle.json"
	if err := os.WriteFile(bundlePath, []byte(bundleData), 0600); err != nil {
		t.Fatalf("failed to write temp bundle: %v", err)
	}
	old := signatures.RedHatKeyBundlePath
	signatures.RedHatKeyBundlePath = bundlePath
	t.Cleanup(func() { signatures.RedHatKeyBundlePath = old })
}

func TestSeedFirstInstall(t *testing.T) {
	writeTempBundle(t)

	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatIntegrationID
	mockStore.EXPECT().Get(gomock.Any(), id).Return(nil, false, nil).Times(1)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Cond(func(x any) bool {
		si, ok := x.(*storage.SignatureIntegration)
		return ok && si.GetId() == id
	})).Return(nil).Times(1)

	seedRedHatDefaultSignatureIntegration(mockStore)
}

func TestSeedSubsequentStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatIntegrationID
	mockStore.EXPECT().Get(gomock.Any(), id).Return(&storage.SignatureIntegration{Id: id}, true, nil).Times(1)

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
