package version

import (
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/version/postgres"
	"github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestEnsurer(t *testing.T) {
	suite.Run(t, new(EnsurerTestSuite))
}

type EnsurerTestSuite struct {
	suite.Suite

	boltDB       *bolt.DB
	rocksDB      *rocksdb.RocksDB
	pgStore      postgres.Store
	pool         *pgxpool.Pool
	versionStore store.Store
}

func (suite *EnsurerTestSuite) SetupTest() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testDB := pgtest.ForT(suite.T())
		suite.pool = testDB.Pool

		suite.versionStore = store.NewPostgres(suite.pool)
	} else {
		boltDB, err := bolthelper.NewTemp(testutils.DBFileName(suite))
		suite.Require().NoError(err, "Failed to make BoltDB")

		rocksDB := rocksdbtest.RocksDBForT(suite.T())
		suite.Require().NoError(err, "Failed to create RocksDB")

		suite.boltDB = boltDB
		suite.rocksDB = rocksDB

		suite.versionStore = store.New(boltDB, rocksDB)
	}
}

func (suite *EnsurerTestSuite) TearDownTest() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		if suite.pool != nil {
			suite.pool.Close()
		}
	} else {
		suite.NoError(suite.boltDB.Close())
	}
}

func (suite *EnsurerTestSuite) TestWithEmptyDB() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		suite.NoError(Ensure(store.NewPostgres(suite.pool)))
	} else {
		suite.NoError(Ensure(store.New(suite.boltDB, suite.rocksDB)))
	}
	version, err := suite.versionStore.GetVersion()
	suite.NoError(err)
	suite.Equal(migrations.CurrentDBVersionSeqNum(), int(version.GetSeqNum()))
}

func (suite *EnsurerTestSuite) TestWithCurrentVersion() {
	suite.NoError(suite.versionStore.UpdateVersion(&storage.Version{SeqNum: int32(migrations.CurrentDBVersionSeqNum())}))
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		suite.NoError(Ensure(store.NewPostgres(suite.pool)))
	} else {
		suite.NoError(Ensure(store.New(suite.boltDB, suite.rocksDB)))
	}

	version, err := suite.versionStore.GetVersion()
	suite.NoError(err)
	suite.Equal(migrations.CurrentDBVersionSeqNum(), int(version.GetSeqNum()))
}

func (suite *EnsurerTestSuite) TestWithIncorrectVersion() {
	suite.NoError(suite.versionStore.UpdateVersion(&storage.Version{SeqNum: int32(migrations.CurrentDBVersionSeqNum()) - 2}))
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		suite.Error(Ensure(store.NewPostgres(suite.pool)))
	} else {
		suite.Error(Ensure(store.New(suite.boltDB, suite.rocksDB)))
	}
}
