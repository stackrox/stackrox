package datastore

import (
	"context"
	"encoding/pem"
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

// allAccessCtx returns an all-access context suitable for unit tests.
func allAccessCtx() context.Context {
	return sac.WithAllAccess(context.Background())
}

// newUpsertTestStore creates a mock store for upsert tests.
func newUpsertTestStore(t *testing.T) *storeMocks.MockSignatureIntegrationStore {
	t.Helper()
	ctrl := gomock.NewController(t)
	return storeMocks.NewMockSignatureIntegrationStore(ctrl)
}

func TestUpsertRedHatSignatureIntegration_EmbeddedKeyOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	mockStore := newUpsertTestStore(t)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, si *storage.SignatureIntegration) error {
			require.Equal(t, signatures.DefaultRedHatSignatureIntegration.GetId(), si.GetId())
			require.Len(t, si.GetCosign().GetPublicKeys(), 1,
				"empty dir: only the embedded key should be present")
			return nil
		},
	)

	upsertRedHatSignatureIntegration(mockStore)
}

func TestUpsertRedHatSignatureIntegration_AddsKeysFromDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pub"), []byte(anotherValidPublicKeyPEM), 0o600))

	mockStore := newUpsertTestStore(t)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, si *storage.SignatureIntegration) error {
			require.Equal(t, signatures.DefaultRedHatSignatureIntegration.GetId(), si.GetId())
			require.Len(t, si.GetCosign().GetPublicKeys(), 2,
				"embedded key + one dir key expected")
			names := map[string]struct{}{}
			for _, k := range si.GetCosign().GetPublicKeys() {
				names[k.GetName()] = struct{}{}
			}
			require.Contains(t, names, "extra.pub")
			return nil
		},
	)

	upsertRedHatSignatureIntegration(mockStore)
}

func TestUpsertRedHatSignatureIntegration_DeduplicatesDirKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	// Write a file that contains the same key as the embedded key.
	embeddedPEM := signatures.DefaultRedHatSignatureIntegration.GetCosign().GetPublicKeys()[0].GetPublicKeyPemEnc()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "duplicate.pub"), []byte(embeddedPEM), 0o600))

	mockStore := newUpsertTestStore(t)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, si *storage.SignatureIntegration) error {
			require.Len(t, si.GetCosign().GetPublicKeys(), 1,
				"duplicate key from dir must be dropped")
			return nil
		},
	)

	upsertRedHatSignatureIntegration(mockStore)
}

func TestUpsertRedHatSignatureIntegration_DeduplicatesWithTrailingWhitespace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	// Write the embedded key with extra trailing newlines — must still be treated as duplicate.
	embeddedPEM := signatures.DefaultRedHatSignatureIntegration.GetCosign().GetPublicKeys()[0].GetPublicKeyPemEnc()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "trailing.pub"), []byte(embeddedPEM+"\n\n"), 0o600))

	mockStore := newUpsertTestStore(t)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, si *storage.SignatureIntegration) error {
			require.Len(t, si.GetCosign().GetPublicKeys(), 1,
				"key with trailing whitespace must be recognised as duplicate of embedded key")
			return nil
		},
	)

	upsertRedHatSignatureIntegration(mockStore)
}

func TestUpsertRedHatSignatureIntegration_PEMStoredInCanonicalForm(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pub"), []byte(anotherValidPublicKeyPEM+"\n\n"), 0o600))

	mockStore := newUpsertTestStore(t)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, si *storage.SignatureIntegration) error {
			require.Len(t, si.GetCosign().GetPublicKeys(), 2)
			for _, k := range si.GetCosign().GetPublicKeys() {
				if k.GetName() == "extra.pub" {
					// The stored PEM must be the canonical form (no trailing whitespace,
					// produced by pem.EncodeToMemory).
					block, _ := pem.Decode([]byte(k.GetPublicKeyPemEnc()))
					require.NotNil(t, block, "stored PEM must be decodable")
					require.Equal(t, string(pem.EncodeToMemory(block)), k.GetPublicKeyPemEnc(),
						"stored PEM must be in canonical form")
				}
			}
			return nil
		},
	)

	upsertRedHatSignatureIntegration(mockStore)
}

func TestUpsertRedHatSignatureIntegration_DoesNotMutateDefaultIntegration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR", dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pub"), []byte(anotherValidPublicKeyPEM), 0o600))

	before := len(signatures.DefaultRedHatSignatureIntegration.GetCosign().GetPublicKeys())

	mockStore := newUpsertTestStore(t)
	mockStore.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	upsertRedHatSignatureIntegration(mockStore)

	after := len(signatures.DefaultRedHatSignatureIntegration.GetCosign().GetPublicKeys())
	require.Equal(t, before, after, "upsert must not modify the global DefaultRedHatSignatureIntegration")
}
