//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/testcommon"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	ctx := context.Background()
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !features.PostgresDatastore.Enabled() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	}

	source := pgtest.GetConnectionString(t)
	config, _ := pgxpool.ParseConfig(source)
	pool, _ := pgxpool.ConnectConfig(ctx, config)
	defer pool.Close()

	gormDB := pgtest.OpenGormDB(t, source)
	defer pgtest.CloseGormDB(t, gormDB)
	Destroy(ctx, pool)
	pkgSchema.ApplySchemaForTable(ctx, gormDB, networkFlowsTable)

	store := NewClusterStore(pool)
	flowSuite := testcommon.NewFlowStoreTest(store)
	suite.Run(t, flowSuite)
}
