package datastore

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)

	w.checkAndUpsert()

	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))

	w.checkAndUpsert()
}

func TestWatcherInvalidFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"keys": []}`), 0600))

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)
	w.checkAndUpsert()

	assert.NotEqual(t, [sha256.Size]byte{}, w.lastHash)
}

func TestWatcherFileDoesNotExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "nonexistent.json")

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)
	w.checkAndUpsert()

	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)
}

func TestWatcherFileDeletedResetsHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(2)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)

	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))
	w.checkAndUpsert()
	assert.NotEqual(t, [sha256.Size]byte{}, w.lastHash)

	require.NoError(t, os.Remove(filePath))
	w.checkAndUpsert()
	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)

	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))
	w.checkAndUpsert()
}

func TestWatcherFileChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(2)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)

	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))
	w.checkAndUpsert()
	firstHash := w.lastHash

	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON2Keys()), 0600))
	w.checkAndUpsert()
	assert.NotEqual(t, firstHash, w.lastHash)
}

func TestWatcherFileUnchanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().Upsert(gomock.Any(), redHatIntegrationMatcher()).Return(nil).Times(1)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	require.NoError(t, os.WriteFile(filePath, []byte(validBundleJSON()), 0600))

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)
	w.checkAndUpsert()
	w.checkAndUpsert()
}

func TestWatcherUpsertRetryOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	content := []byte(validBundleJSON())
	require.NoError(t, os.WriteFile(filePath, content, 0600))

	firstCall := mockStore.EXPECT().
		Upsert(gomock.Any(), redHatIntegrationMatcher()).
		Return(errors.New("transient DB error")).
		Times(1)
	mockStore.EXPECT().
		Upsert(gomock.Any(), redHatIntegrationMatcher()).
		Return(nil).
		Times(1).
		After(firstCall)

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)

	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)

	w.checkAndUpsert()
	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)

	w.checkAndUpsert()
	assert.Equal(t, sha256.Sum256(content), w.lastHash)
}

func TestWatcherClampsInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	w := newKeyBundleWatcher("/nonexistent", time.Millisecond, mockStore)
	assert.GreaterOrEqual(t, w.interval, minWatchInterval)

	w = newKeyBundleWatcher("/nonexistent", minWatchInterval, mockStore)
	assert.Equal(t, minWatchInterval, w.interval)

	longInterval := 2 * minWatchInterval
	w = newKeyBundleWatcher("/nonexistent", longInterval, mockStore)
	assert.Equal(t, longInterval, w.interval)
}

func TestWatcherOversizedFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	oversizedContent := []byte(strings.Repeat("x", maxBundleFileSize+1))
	require.NoError(t, os.WriteFile(filePath, oversizedContent, 0600))

	w := newKeyBundleWatcher(filePath, 24*time.Hour, mockStore)
	w.checkAndUpsert()

	firstHash := w.lastHash
	assert.NotEqual(t, [sha256.Size]byte{}, firstHash)

	w.checkAndUpsert()
	assert.Equal(t, firstHash, w.lastHash)
}
