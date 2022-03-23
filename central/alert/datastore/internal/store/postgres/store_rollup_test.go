//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AlertsStoreRollupSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestAlertsStoreRollup(t *testing.T) {
	suite.Run(t, new(AlertsStoreRollupSuite))
}

func (s *AlertsStoreRollupSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}
}

func (s *AlertsStoreRollupSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *AlertsStoreRollupSuite) TestStore() {
	ctx := context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)
	store := New(ctx, pool)

	alert := fixtures.GetAlert()

	foundAlert, exists, err := store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundAlert)

	s.NoError(store.Upsert(ctx, alert))
	foundAlert, exists, err = store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(alert, foundAlert)

	alertCount, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(alertCount, 1)

	alertExists, err := store.Exists(ctx, alert.GetId())
	s.NoError(err)
	s.True(alertExists)
	s.NoError(store.Upsert(ctx, alert))

	foundAlert, exists, err = store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(alert, foundAlert)

	rollupAlert, found, err := store.(*storeImpl).GetWithRollup(ctx, alert.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	spew.Dump(rollupAlert)

	s.NoError(store.Delete(ctx, alert.GetId()))
	foundAlert, exists, err = store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundAlert)
}

func BenchmarkAlertGet(b *testing.B) {
	envIsolator := envisolator.NewEnvIsolator(b)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !features.PostgresDatastore.Enabled() {
		return
	}
	source := pgtest.GetConnectionString(nil)
	config, err := pgxpool.ParseConfig(source)

	pool, err := pgxpool.ConnectConfig(ctx, config)
	require.NoError(b, err)
	defer pool.Close()

	Destroy(ctx, pool)
	store := New(ctx, pool)

	alert := fixtures.GetAlert()

	require.NoError(b, store.Upsert(ctx, alert))

	b.Run("plain get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, found, err := store.Get(ctx, alert.GetId())
			require.NoError(b, err)
			assert.True(b, found)
		}
	})

	b.Run("get with rollup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, found, err := store.(*storeImpl).GetWithRollup(ctx, alert.GetId())
			require.NoError(b, err)
			assert.True(b, found)
		}
	})

}
