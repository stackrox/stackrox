//go:build sql_integration

package leases

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaseStore_ClaimAndRelease(t *testing.T) {
	ctx := t.Context()
	testDB := pgtest.ForT(t)

	store := New(testDB)
	require.NoError(t, store.Initialize(ctx))

	err := store.Claim(ctx, "cluster-abc", "central-0")
	require.NoError(t, err)

	podID, err := store.GetHolder(ctx, "cluster-abc")
	require.NoError(t, err)
	assert.Equal(t, "central-0", podID)

	err = store.Release(ctx, "cluster-abc", "central-0")
	require.NoError(t, err)

	podID, err = store.GetHolder(ctx, "cluster-abc")
	require.NoError(t, err)
	assert.Empty(t, podID)
}

func TestLeaseStore_Heartbeat(t *testing.T) {
	ctx := t.Context()
	testDB := pgtest.ForT(t)

	store := New(testDB)
	require.NoError(t, store.Initialize(ctx))

	err := store.Claim(ctx, "cluster-abc", "central-0")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	err = store.Heartbeat(ctx, "cluster-abc", "central-0")
	require.NoError(t, err)

	stale, err := store.GetStaleLeases(ctx, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, stale)
}

func TestLeaseStore_StaleDetection(t *testing.T) {
	ctx := t.Context()
	testDB := pgtest.ForT(t)

	store := New(testDB)
	require.NoError(t, store.Initialize(ctx))

	err := store.Claim(ctx, "cluster-abc", "central-0")
	require.NoError(t, err)

	stale, err := store.GetStaleLeases(ctx, 0)
	require.NoError(t, err)
	assert.Len(t, stale, 1)
	assert.Equal(t, "cluster-abc", stale[0].ClusterID)
}

func TestLeaseStore_GetAllLeases(t *testing.T) {
	ctx := t.Context()
	testDB := pgtest.ForT(t)

	store := New(testDB)
	require.NoError(t, store.Initialize(ctx))

	require.NoError(t, store.Claim(ctx, "cluster-a", "central-0"))
	require.NoError(t, store.Claim(ctx, "cluster-b", "central-1"))

	leases, err := store.GetAllLeases(ctx)
	require.NoError(t, err)
	assert.Len(t, leases, 2)
}
