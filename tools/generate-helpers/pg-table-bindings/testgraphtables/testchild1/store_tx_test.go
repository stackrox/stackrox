//go:build sql_integration

package postgres

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
)

func (s *TestChild1StoreSuite) TestStoreTxRollback() {
	ctx := sac.WithAllAccess(context.Background())

	tx, err := s.testDB.Begin(ctx)
	s.NoError(err)

	ctx = postgres.ContextWithTx(ctx, tx)
	ctx, _ = contextutil.ContextWithTimeoutIfNotExists(ctx, time.Minute*10)
	tx, ok := postgres.TxFromContext(ctx)
	s.True(ok)
	s.testWithCtx(ctx)
	s.NoError(tx.Rollback(ctx))

	// The transaction is cancelled so no rows should exist
	testChild1Count, err := s.store.Count(sac.WithAllAccess(context.Background()))
	s.NoError(err)
	s.Equal(0, testChild1Count)
}

func (s *TestChild1StoreSuite) TestStoreTxCommit() {
	ctx := sac.WithAllAccess(context.Background())

	tx, err := s.testDB.Begin(ctx)
	s.NoError(err)

	ctx = postgres.ContextWithTx(ctx, tx)
	ctx, _ = contextutil.ContextWithTimeoutIfNotExists(ctx, time.Minute*10)
	tx, ok := postgres.TxFromContext(ctx)
	s.True(ok)
	s.testWithCtx(ctx)
	s.NoError(tx.Commit(ctx))

	// Transaction is committed and expect 200 rows.
	testChild1Count, err := s.store.Count(sac.WithAllAccess(context.Background()))
	s.NoError(err)
	s.Equal(200, testChild1Count)
}

func (s *TestChild1StoreSuite) TestStoreTxInnerRollback() {
	ctx := sac.WithAllAccess(context.Background())

	tx, err := s.testDB.Begin(ctx)
	s.NoError(err)

	ctx = postgres.ContextWithTx(ctx, tx)
	ctx, _ = contextutil.ContextWithTimeoutIfNotExists(ctx, time.Minute*10)
	tx, ok := postgres.TxFromContext(ctx)
	s.True(ok)
	s.testWithCtx(ctx)
	{
		// This is to emulate an error occurred in the store. In reality,
		// we should not extract and use inner tx here.
		innerCtx, ok := postgres.TxFromContext(ctx)
		s.True(ok)
		s.NoError(innerCtx.Rollback(ctx))

		// The transaction is cancelled so no rows should exist
		testChild1Count, err := s.store.Count(sac.WithAllAccess(context.Background()))
		s.NoError(err)
		s.Equal(0, testChild1Count)
	}
	s.NoError(tx.Rollback(ctx))

	// The transaction is cancelled so no rows should exist
	testChild1Count, err := s.store.Count(sac.WithAllAccess(context.Background()))
	s.NoError(err)
	s.Equal(0, testChild1Count)
}

func (s *TestChild1StoreSuite) testWithCtx(ctx context.Context) {
	store := s.store

	testChild1 := &storage.TestChild1{}
	s.NoError(testutils.FullInit(testChild1, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	foundTestChild1, exists, err := store.Get(ctx, testChild1.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundTestChild1)

	s.NoError(store.Upsert(ctx, testChild1))
	foundTestChild1, exists, err = store.Get(ctx, testChild1.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(testChild1, foundTestChild1)

	testChild1Count, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(1, testChild1Count)

	testChild1Exists, err := store.Exists(ctx, testChild1.GetId())
	s.NoError(err)
	s.True(testChild1Exists)
	s.NoError(store.Upsert(ctx, testChild1))

	s.NoError(store.Delete(ctx, testChild1.GetId()))
	foundTestChild1, exists, err = store.Get(ctx, testChild1.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundTestChild1)

	var testChild1s []*storage.TestChild1
	for i := 0; i < 200; i++ {
		testChild1 := &storage.TestChild1{}
		s.NoError(testutils.FullInit(testChild1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		testChild1s = append(testChild1s, testChild1)
	}

	s.NoError(store.UpsertMany(ctx, testChild1s))

	testChild1Count, err = store.Count(ctx)
	s.NoError(err)
	s.Equal(200, testChild1Count)
}
