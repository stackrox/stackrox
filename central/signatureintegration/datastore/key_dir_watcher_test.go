package datastore

import (
	"testing"

	storeMocks "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestKeyDirHandler_OnStableUpdate_TriggersUpsert verifies that OnStableUpdate
// calls upsertRedHatSignatureIntegration when there is no error.
func TestKeyDirHandler_OnStableUpdate_TriggersUpsert(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	mockStore.EXPECT().
		Upsert(gomock.Any(), gomock.Cond(func(si any) bool {
			type idGetter interface{ GetId() string }
			g, ok := si.(idGetter)
			return ok && g.GetId() == signatures.DefaultRedHatSignatureIntegration.GetId()
		})).
		Return(nil).
		Times(1)

	h := &keyDirHandler{siStore: mockStore}
	h.OnStableUpdate(nil, nil)
}

// TestKeyDirHandler_OnStableUpdate_SkipsOnError verifies that OnStableUpdate
// does NOT call the store when invoked with a non-nil error.
func TestKeyDirHandler_OnStableUpdate_SkipsOnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	// No Upsert call expected — gomock will fail the test if Upsert is called.

	h := &keyDirHandler{siStore: mockStore}
	h.OnStableUpdate(nil, assert.AnError)
}

// TestKeyDirHandler_OnChange_IsSideEffectFree verifies that OnChange is a
// no-op (side-effect-free as required by k8scfgwatch.Handler).
func TestKeyDirHandler_OnChange_IsSideEffectFree(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storeMocks.NewMockSignatureIntegrationStore(ctrl)
	// No store calls expected.

	h := &keyDirHandler{siStore: mockStore}
	val, err := h.OnChange(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, val)
}
