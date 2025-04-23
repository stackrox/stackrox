//go:build sql_integration

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/multitest/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContexCancellationInWalk(t *testing.T) {
	t.Parallel()

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	store := postgres.New(testDB.DB)

	testStructs := getTestStructs()

	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	ctxWithCancel, cancel := context.WithCancelCause(ctx)

	count := 0
	err := store.Walk(ctxWithCancel, func(obj *storage.TestStruct) error {
		cancel(fmt.Errorf("cancelling context on the first read"))
		count++
		return nil
	})
	assert.Equal(t, 1, count)
	assert.ErrorIs(t, err, context.Canceled)
}
