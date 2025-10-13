//go:build sql_integration

package version

import (
	"testing"

	"github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
)

func TestEnsurer(t *testing.T) {
	suite.Run(t, new(EnsurerTestSuite))
}

type EnsurerTestSuite struct {
	suite.Suite

	pool         postgres.DB
	versionStore store.Store
}

func (suite *EnsurerTestSuite) SetupTest() {
	testDB := pgtest.ForT(suite.T())
	suite.pool = testDB.DB

	suite.versionStore = store.NewPostgres(suite.pool)

}

func (suite *EnsurerTestSuite) TearDownTest() {
	if suite.pool != nil {
		suite.pool.Close()
	}
}

func (suite *EnsurerTestSuite) TestWithEmptyDB() {
	suite.NoError(Ensure(store.NewPostgres(suite.pool)))
	version, err := suite.versionStore.GetVersion()
	suite.NoError(err)
	suite.Equal(migrations.CurrentDBVersionSeqNum(), int(version.GetSeqNum()))
}

func (suite *EnsurerTestSuite) TestWithCurrentVersion() {
	suite.NoError(suite.versionStore.UpdateVersion(&storage.Version{SeqNum: int32(migrations.CurrentDBVersionSeqNum())}))
	suite.NoError(Ensure(store.NewPostgres(suite.pool)))

	version, err := suite.versionStore.GetVersion()
	suite.NoError(err)
	suite.Equal(migrations.CurrentDBVersionSeqNum(), int(version.GetSeqNum()))
}

func (suite *EnsurerTestSuite) TestWithIncorrectVersion() {
	suite.NoError(suite.versionStore.UpdateVersion(&storage.Version{SeqNum: int32(migrations.CurrentDBVersionSeqNum()) - 2}))
	suite.Error(Ensure(store.NewPostgres(suite.pool)))
}
