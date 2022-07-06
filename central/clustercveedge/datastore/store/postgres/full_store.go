package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
)

// NewFullStore augments the generated store with backward compatible functions.
func NewFullStore(db *pgxpool.Pool) store.Store {
	return &fullStoreImpl{
		Store: New(db),
	}
}

type fullStoreImpl struct {
	Store
}

func (f *fullStoreImpl) Upsert(_ context.Context, _ ...converter.ClusterCVEParts) error {
	if features.PostgresDatastore.Enabled() {
		return utils.Should(errors.New("Unexpected cluster-cve edge upsert when running on Postgres"))
	}
	return nil
}

func (f *fullStoreImpl) Delete(_ context.Context, _ ...string) error {
	if features.PostgresDatastore.Enabled() {
		return utils.Should(errors.New("Unexpected cluster-cve edge upsert when running on Postgres"))
	}
	return nil
}
