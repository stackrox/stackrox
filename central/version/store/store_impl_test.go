//go:build sql_integration

package store

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/version/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestVersionStore(t *testing.T) {
	suite.Run(t, new(VersionStoreTestSuite))
}

type VersionStoreTestSuite struct {
	suite.Suite

	pool  postgres.DB
	ctx   context.Context
	store Store
}

func (suite *VersionStoreTestSuite) SetupTest() {
	suite.ctx = sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(suite.T())
	suite.pool = testDB.DB

	suite.store = NewPostgres(suite.pool)
}

func (suite *VersionStoreTestSuite) TearDownTest() {
	if suite.pool != nil {
		pgStore.Destroy(suite.ctx, suite.pool)
		suite.pool.Close()
	}
}

func (suite *VersionStoreTestSuite) TestVersionStore() {
	v, err := suite.store.GetVersion()
	suite.NoError(err)
	suite.Nil(v)

	for _, version := range []int32{2, 5, 19} {
		protoVersion := &storage.Version{SeqNum: version, Version: fmt.Sprintf("Version %d", version)}
		suite.NoError(suite.store.UpdateVersion(protoVersion))
		got, err := suite.store.GetVersion()
		suite.NoError(err)
		suite.Equal(protoVersion, got)
	}
}
