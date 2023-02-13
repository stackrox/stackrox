//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/testcommon"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	ctx := context.Background()
	t.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	}

	source := pgtest.GetConnectionString(t)
	config, _ := postgres.ParseConfig(source)
	pool, _ := postgres.New(ctx, config)
	defer pool.Close()

	gormDB := pgtest.OpenGormDB(t, source)
	defer pgtest.CloseGormDB(t, gormDB)
	Destroy(ctx, pool)
	pkgSchema.ApplySchemaForTable(ctx, gormDB, networkFlowsTable)

	store := NewClusterStore(pool)
	flowSuite := testcommon.NewFlowStoreTest(store)
	suite.Run(t, flowSuite)
}
