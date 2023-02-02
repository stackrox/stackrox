package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/utils"
)

// NewFullStore augments the generated store with backward compatible functions.
func NewFullStore(db *pgxpool.Pool) store.Store {
	return &fullStoreImpl{
		Store: New(db),
	}
}

// NewFullTestStore is used for testing.
func NewFullTestStore(_ testing.TB, store Store) store.Store {
	return &fullStoreImpl{
		Store: store,
	}
}

type fullStoreImpl struct {
	Store
}

func (f *fullStoreImpl) Upsert(_ context.Context, _ ...converter.ClusterCVEParts) error {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return utils.ShouldErr(errors.New("Unexpected cluster-cve edge upsert when running on Postgres"))
	}
	return nil
}

func (f *fullStoreImpl) Delete(_ context.Context, _ ...string) error {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return utils.ShouldErr(errors.New("Unexpected cluster-cve edge upsert when running on Postgres"))
	}
	return nil
}
