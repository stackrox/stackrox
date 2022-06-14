package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/stackrox/central/networkgraph/flow/datastore/internal/store/testcommon"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/stackrox/pkg/postgres/schema"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	ctx := context.Background()
	envIsolator := envisolator.NewEnvIsolator(t)

	if !features.PostgresDatastore.Enabled() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	} else {
		envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

		source := pgtest.GetConnectionString(t)
		config, _ := pgxpool.ParseConfig(source)
		pool, _ := pgxpool.ConnectConfig(ctx, config)
		defer pool.Close()

		gormDB := pgtest.OpenGormDB(t, source)
		defer pgtest.CloseGormDB(t, gormDB)
		pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)

		store := NewClusterStore(pool)
		flowSuite := testcommon.NewFlowStoreTest(store)
		suite.Run(t, flowSuite)
	}
}
