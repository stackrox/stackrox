package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/testcommon"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	ctx := context.Background()
	//clusterId := "22"

	source := pgtest.GetConnectionString(t)
	config, _ := pgxpool.ParseConfig(source)
	pool, _ := pgxpool.ConnectConfig(ctx, config)
	defer pool.Close()

	store := NewClusterStore(pool)
	flowSuite := testcommon.NewFlowStoreTest(store)
	suite.Run(t, flowSuite)
}
