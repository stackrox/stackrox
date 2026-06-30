//go:build sql_integration

package propagation

import (
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionStore_IncrementAndGet(t *testing.T) {
	ctx := t.Context()
	testDB := pgtest.ForT(t)

	store := NewVersionStore(testDB)
	require.NoError(t, store.Initialize(ctx))

	// Initial version should be 0
	version, err := store.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), version)

	// First increment: 0 → 1
	require.NoError(t, store.IncrementVersion(ctx))
	version, err = store.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)

	// Second increment: 1 → 2
	require.NoError(t, store.IncrementVersion(ctx))
	version, err = store.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), version)
}
