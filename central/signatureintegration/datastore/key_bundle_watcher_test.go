package datastore

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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

func validBundleJSON() string {
	return fmt.Sprintf(`{"keys": [{"name": "test-key-1", "pem": %q}]}`, testPublicKeyPEM)
}

func validBundleJSON2Keys() string {
	return fmt.Sprintf(`{"keys": [{"name": "test-key-1", "pem": %q}, {"name": "test-key-2", "pem": %q}]}`,
		testPublicKeyPEM, testPublicKeyPEM2)
}

func TestWatcherFileAppears(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(1)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	w := newKeyBundleWatcher(filePath, 50*time.Millisecond, mockStore)

	// File does not exist yet — no upsert expected on first check.
	w.checkAndUpsert()

	// Write valid bundle file.
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))

	// Now the file exists — upsert should be called.
	w.checkAndUpsert()
}

func TestWatcherInvalidFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	// Upsert must NOT be called for an invalid bundle.

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"keys": []}`), 0600))

	w := newKeyBundleWatcher(filePath, 50*time.Millisecond, mockStore)
	w.checkAndUpsert()

	// Hash is updated even on parse failure to avoid log spam from repeated attempts.
	assert.NotEqual(t, [sha256.Size]byte{}, w.lastHash)
}

func TestWatcherFileDoesNotExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "nonexistent.json")

	w := newKeyBundleWatcher(filePath, 50*time.Millisecond, mockStore)
	w.checkAndUpsert()

	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)
}

func TestWatcherFileDeletedResetsHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(2)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	w := newKeyBundleWatcher(filePath, 50*time.Millisecond, mockStore)

	// Write and process file.
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))
	w.checkAndUpsert()
	assert.NotEqual(t, [sha256.Size]byte{}, w.lastHash)

	// Delete file — hash should be reset.
	require.NoError(t, os.Remove(filePath))
	w.checkAndUpsert()
	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)

	// Re-create with same content — should upsert again since hash was reset.
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))
	w.checkAndUpsert()
}

func TestWatcherFileChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(2)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	w := newKeyBundleWatcher(filePath, 50*time.Millisecond, mockStore)

	// First version.
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))
	w.checkAndUpsert()
	firstHash := w.lastHash

	// Updated version with two keys.
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON2Keys()), 0600))
	w.checkAndUpsert()
	assert.NotEqual(t, firstHash, w.lastHash)
}

func TestWatcherFileUnchanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	// Upsert should only be called once — the second check has the same hash.
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(1)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))

	w := newKeyBundleWatcher(filePath, 50*time.Millisecond, mockStore)
	w.checkAndUpsert()
	w.checkAndUpsert()
}

func TestWatcherStartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).AnyTimes()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))

	w := &keyBundleWatcher{
		filePath: filePath,
		interval: 50 * time.Millisecond,
		siStore:  mockStore,
		stopSig:  concurrency.NewSignal(),
	}

	w.Start()
	time.Sleep(100 * time.Millisecond)
	w.Stop()
	select {
	case <-w.stopSig.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop within timeout")
	}
}
