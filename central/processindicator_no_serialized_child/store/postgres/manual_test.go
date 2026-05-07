//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
)

func TestManualTableCreation(t *testing.T) {
	testDB := pgtest.ForT(t)
	ctx := sac.WithAllAccess(context.Background())

	// Try CreateTableAndNewStore which should create the table
	store := CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	require.NotNil(t, store)

	// Now try a simple operation
	_, exists, err := store.Get(ctx, "test-id")
	require.NoError(t, err)
	require.False(t, exists)
}
