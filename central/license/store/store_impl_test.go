package store

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Parallel()

	db, err := bolthelper.NewTemp(t.Name() + ".db")
	require.NoError(t, err)

	store, err := New(db)
	require.NoError(t, err)

	allKeys, err := store.ListLicenseKeys()
	require.NoError(t, err)
	assert.Empty(t, allKeys)

	key1 := &storage.StoredLicenseKey{
		LicenseId:  uuid.NewV4().String(),
		LicenseKey: "ABCD.EFGH",
		Selected:   true,
	}

	err = store.UpsertLicenseKeys([]*storage.StoredLicenseKey{key1})
	require.NoError(t, err)

	allKeys, err = store.ListLicenseKeys()
	require.NoError(t, err)

	assert.ElementsMatch(t, allKeys, []*storage.StoredLicenseKey{key1})

	key2 := &storage.StoredLicenseKey{
		LicenseId:  uuid.NewV4().String(),
		LicenseKey: "IJKL.MNOP",
		Selected:   false,
	}

	err = store.UpsertLicenseKeys([]*storage.StoredLicenseKey{key2})
	require.NoError(t, err)

	allKeys, err = store.ListLicenseKeys()
	require.NoError(t, err)

	assert.ElementsMatch(t, allKeys, []*storage.StoredLicenseKey{key1, key2})

	err = store.DeleteLicenseKey(key1.GetLicenseId())
	require.NoError(t, err)

	allKeys, err = store.ListLicenseKeys()
	require.NoError(t, err)

	assert.ElementsMatch(t, allKeys, []*storage.StoredLicenseKey{key2})
}
