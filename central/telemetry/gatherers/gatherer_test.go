package gatherers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/grpc/metrics"
	installation "github.com/stackrox/rox/central/installation/store"
	installationBolt "github.com/stackrox/rox/central/installation/store/bolt"
	installationPostgres "github.com/stackrox/rox/central/installation/store/postgres"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"go.etcd.io/bbolt"
)

func TestGatherers(t *testing.T) {
	suite.Run(t, new(gathererTestSuite))
}

type gathererTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	bolt  *bbolt.DB
	rocks *rocksdb.RocksDB
	index bleve.Index

	tp *pgtest.TestPostgres

	gatherer                     *CentralGatherer
	sensorUpgradeConfigDatastore *mocks.MockDataStore
}

func (s *gathererTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())

	s.sensorUpgradeConfigDatastore = mocks.NewMockDataStore(s.mockCtrl)
	s.sensorUpgradeConfigDatastore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Return(&storage.SensorUpgradeConfig{
		EnableAutoUpgrade: true,
	}, nil)

	var installationStore installation.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.tp = pgtest.ForTCustomDB(s.T(), "postgres")
		source := pgtest.GetConnectionString(s.T())
		adminConfig, err := pgxpool.ParseConfig(source)
		s.NoError(err)

		installationStore = installationPostgres.New(s.tp.Pool)

		s.gatherer = newCentralGatherer(installationStore, newDatabaseGatherer(nil, nil, nil, newPostgresGatherer(s.tp.Pool, adminConfig)), newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()), gatherers.NewComponentInfoGatherer(), s.sensorUpgradeConfigDatastore)
	} else {

		boltDB, err := bolthelper.NewTemp("gatherer_test.db")
		s.Require().NoError(err, "Failed to make BoltDB: %s", err)
		s.bolt = boltDB

		rocksDB := rocksdbtest.RocksDBForT(s.T())
		s.Require().NoError(err, "Failed to make RocksDB: %s", err)
		s.rocks = rocksDB

		index, err := globalindex.MemOnlyIndex()
		s.Require().NoError(err, "Failed to make in-memory Bleve: %s", err)
		s.index = index

		installationStore = installationBolt.New(s.bolt)
		s.Require().NoError(err, "Failed to make installation store")

		s.gatherer = newCentralGatherer(installationStore, newDatabaseGatherer(newRocksDBGatherer(s.rocks), newBoltGatherer(s.bolt), newBleveGatherer(s.index), nil), newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()), gatherers.NewComponentInfoGatherer(), s.sensorUpgradeConfigDatastore)
	}
}

func (s *gathererTestSuite) TearDownSuite() {
	if s.bolt != nil {
		testutils.TearDownDB(s.bolt)
	}
	if s.rocks != nil {
		rocksdbtest.TearDownRocksDB(s.rocks)
	}
	if s.tp != nil {
		s.tp.Teardown(s.T())
	}
}

func (s *gathererTestSuite) TestJSONSerialization() {
	metrics := s.gatherer.Gather(sac.WithAllAccess(context.Background()))

	bytes, err := json.Marshal(metrics)
	s.NoError(err)

	marshalledMetrics := &data.CentralInfo{}
	err = json.Unmarshal(bytes, &marshalledMetrics)
	s.NoError(err)

	s.Equal(metrics.Orchestrator, marshalledMetrics.Orchestrator)
	s.Equal(metrics.Errors, marshalledMetrics.Errors)
	s.Equal(metrics.Storage, marshalledMetrics.Storage)
	s.Equal(metrics.Process, marshalledMetrics.Process)
	s.Equal(metrics.Restarts, marshalledMetrics.Restarts)
	s.Equal(metrics.Version, marshalledMetrics.Version)
	s.Equal(metrics.RoxComponentInfo, marshalledMetrics.RoxComponentInfo)
	// API stats will be empty so the marshalled metrics will contain nil instead of empty
	s.Nil(marshalledMetrics.APIStats.HTTP)
	s.Nil(marshalledMetrics.APIStats.GRPC)
}
