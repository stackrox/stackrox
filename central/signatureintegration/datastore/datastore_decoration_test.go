package datastore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// readCtx is an all-access read context suitable for unit tests.
func allAccessCtx() context.Context {
	return sac.WithAllAccess(context.Background())
}

// newDecorationTestDataStore creates a datastoreImpl backed by a mock store for decoration tests.
func newDecorationTestDataStore(t *testing.T) (DataStore, *storeMocks.MockSignatureIntegrationStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	ds := New(mockStore, nil)
	return ds, mockStore
}

// rhIntegrationWithKeys returns a copy of the default Red Hat integration.
func rhIntegrationWithKeys(extraKeys ...*storage.CosignPublicKeyVerification_PublicKey) *storage.SignatureIntegration {
	cloned := signatures.DefaultRedHatSignatureIntegration.CloneVT()
	cloned.Cosign.PublicKeys = append(cloned.Cosign.PublicKeys, extraKeys...)
	return cloned
}

func TestGetSignatureIntegration_DecoratesRedHatIntegration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pub"), []byte(anotherValidPublicKeyPEM), 0o600))

	ds, mockStore := newDecorationTestDataStore(t)
	rhID := signatures.DefaultRedHatSignatureIntegration.GetId()
	mockStore.EXPECT().Get(gomock.Any(), rhID).Return(rhIntegrationWithKeys(), true, nil)

	result, found, err := ds.GetSignatureIntegration(allAccessCtx(), rhID)
	require.NoError(t, err)
	require.True(t, found)

	// Should now have the embedded key plus the extra one from the directory.
	require.Len(t, result.GetCosign().GetPublicKeys(), 2)
	names := map[string]struct{}{}
	for _, k := range result.GetCosign().GetPublicKeys() {
		names[k.GetName()] = struct{}{}
	}
	require.Contains(t, names, "extra.pub")
}

func TestGetSignatureIntegration_DoesNotDecorateNonRedHatIntegration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pub"), []byte(anotherValidPublicKeyPEM), 0o600))

	ds, mockStore := newDecorationTestDataStore(t)

	userIntegration := &storage.SignatureIntegration{
		Id:   GenerateSignatureIntegrationID(),
		Name: "user-created",
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{Name: "my-key", PublicKeyPemEnc: validPublicKeyPEM},
			},
		},
	}
	mockStore.EXPECT().Get(gomock.Any(), userIntegration.GetId()).Return(userIntegration, true, nil)

	result, found, err := ds.GetSignatureIntegration(allAccessCtx(), userIntegration.GetId())
	require.NoError(t, err)
	require.True(t, found)
	// Should only have the original key — no decoration.
	require.Len(t, result.GetCosign().GetPublicKeys(), 1)
	require.Equal(t, "my-key", result.GetCosign().GetPublicKeys()[0].GetName())
}

func TestGetAllSignatureIntegrations_DecoratesOnlyRedHatIntegration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pub"), []byte(anotherValidPublicKeyPEM), 0o600))

	ds, mockStore := newDecorationTestDataStore(t)

	userIntegration := &storage.SignatureIntegration{
		Id:   GenerateSignatureIntegrationID(),
		Name: "user-created",
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{Name: "my-key", PublicKeyPemEnc: validPublicKeyPEM},
			},
		},
	}

	mockStore.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(*storage.SignatureIntegration) error) error {
			if err := fn(rhIntegrationWithKeys()); err != nil {
				return err
			}
			return fn(userIntegration)
		},
	)

	results, err := ds.GetAllSignatureIntegrations(allAccessCtx())
	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, r := range results {
		if r.GetId() == signatures.DefaultRedHatSignatureIntegration.GetId() {
			// Red Hat integration must be decorated.
			require.Len(t, r.GetCosign().GetPublicKeys(), 2, "Red Hat integration should have 2 keys")
		} else {
			// User integration must be untouched.
			require.Len(t, r.GetCosign().GetPublicKeys(), 1, "user integration should have 1 key")
		}
	}
}

func TestGetSignatureIntegration_EmptyRuntimeDir_ReturnsOnlyEmbeddedKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	ds, mockStore := newDecorationTestDataStore(t)
	rhID := signatures.DefaultRedHatSignatureIntegration.GetId()
	mockStore.EXPECT().Get(gomock.Any(), rhID).Return(rhIntegrationWithKeys(), true, nil)

	result, found, err := ds.GetSignatureIntegration(allAccessCtx(), rhID)
	require.NoError(t, err)
	require.True(t, found)
	// Only the embedded key from DefaultRedHatSignatureIntegration.
	require.Len(t, result.GetCosign().GetPublicKeys(), 1)
}

func TestGetSignatureIntegration_DeduplicatesKeysFromDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	// Write a file whose PEM matches the embedded key — should not be added twice.
	embeddedPEM := signatures.DefaultRedHatSignatureIntegration.GetCosign().GetPublicKeys()[0].GetPublicKeyPemEnc()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "duplicate.pub"), []byte(embeddedPEM), 0o600))

	ds, mockStore := newDecorationTestDataStore(t)
	rhID := signatures.DefaultRedHatSignatureIntegration.GetId()
	mockStore.EXPECT().Get(gomock.Any(), rhID).Return(rhIntegrationWithKeys(), true, nil)

	result, found, err := ds.GetSignatureIntegration(allAccessCtx(), rhID)
	require.NoError(t, err)
	require.True(t, found)
	// Still only one key — the duplicate was filtered.
	require.Len(t, result.GetCosign().GetPublicKeys(), 1)
}
