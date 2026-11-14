package datastore

import (
	"testing"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/namespace/datastore/internal/store/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMalformedNamespaceIDs(t *testing.T) {
	for name, id := range map[string]string{
		"empty":              "",
		"name":               "my-namespace",
		"invalid UUID":       "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		"SQL metacharacters": "' OR TRUE --",
		"UUID URN":           "urn:uuid:aaaaaaaa-bbbb-4011-0000-111111111111",
		"braced UUID":        "{aaaaaaaa-bbbb-4011-0000-111111111111}",
		"compact UUID":       "aaaaaaaabbbb40110000111111111111",
		"uppercase UUID":     "AAAAAAAA-BBBB-4011-0000-111111111111",
	} {
		t.Run(name, func(t *testing.T) {
			// No store calls should be made for IDs that cannot exist.
			ds := New(storeMocks.NewMockStore(gomock.NewController(t)), nil, nil)
			ctx := sac.WithAllAccess(t.Context())
			namespace, found, err := ds.GetNamespace(ctx, id)
			require.NoError(t, err)
			require.False(t, found)
			require.Nil(t, namespace)
			require.NoError(t, ds.RemoveNamespace(ctx, id))
			require.ErrorIs(t, ds.RemoveNamespace(sac.WithNoAccess(t.Context()), id), sac.ErrResourceAccessDenied)
		})
	}
}

func TestNamespaceIDLookupsPreserveStoreErrors(t *testing.T) {
	store := storeMocks.NewMockStore(gomock.NewController(t))
	ds := New(store, nil, nil)
	ctx := sac.WithAllAccess(t.Context())
	id := uuid.NewV4().String()
	storeError := errors.New("database unavailable")
	store.EXPECT().Get(ctx, id).Return(nil, false, storeError)
	store.EXPECT().Delete(ctx, id).Return(storeError)
	namespace, found, err := ds.GetNamespace(ctx, id)
	require.ErrorIs(t, err, storeError)
	require.False(t, found)
	require.Nil(t, namespace)
	require.ErrorIs(t, ds.RemoveNamespace(ctx, id), storeError)
}
