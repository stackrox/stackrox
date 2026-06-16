package datastore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSeedFirstInstall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatSignatureIntegration.GetId()
	mockStore.EXPECT().Get(gomock.Any(), id).Return(nil, false, nil).Times(1)
	mockStore.EXPECT().Upsert(gomock.Any(), signatures.DefaultRedHatSignatureIntegration).Return(nil).Times(1)

	seedRedHatDefaultSignatureIntegration(mockStore)
}

func TestSeedSubsequentStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	id := signatures.DefaultRedHatSignatureIntegration.GetId()
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

func TestSeedKeyBundleFileCreatesFile(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "subdir", "bundle.json")
	t.Setenv(env.RedHatSigningKeyBundleFilePath.EnvVar(), bundlePath)

	seedKeyBundleFile()

	data, err := os.ReadFile(bundlePath)
	require.NoError(t, err)

	var bundle keyBundle
	require.NoError(t, json.Unmarshal(data, &bundle))
	assert.NotEmpty(t, bundle.Keys, "seeded bundle should contain at least one key")

	expectedKeys := signatures.DefaultRedHatSignatureIntegration.GetCosign().GetPublicKeys()
	require.Len(t, bundle.Keys, len(expectedKeys))
	for i, expected := range expectedKeys {
		assert.Equal(t, expected.GetName(), bundle.Keys[i].Name)
	}

	// Verify the seeded file passes the watcher's parseKeyBundle validation,
	// ensuring the PEM is normalized and the format is correct end-to-end.
	parsed, err := parseKeyBundle(data)
	require.NoError(t, err, "seeded bundle must be valid per parseKeyBundle")
	assert.Len(t, parsed.Keys, len(expectedKeys))
}

func TestSeedKeyBundleFileSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "bundle.json")
	t.Setenv(env.RedHatSigningKeyBundleFilePath.EnvVar(), bundlePath)

	existing := []byte(`{"keys": [{"name": "custom", "pem": "custom-data"}]}`)
	require.NoError(t, os.WriteFile(bundlePath, existing, 0600))

	seedKeyBundleFile()

	data, err := os.ReadFile(bundlePath)
	require.NoError(t, err)
	assert.Equal(t, existing, data, "existing file should not be overwritten")
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
