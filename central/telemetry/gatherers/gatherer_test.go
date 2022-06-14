package gatherers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/grpc/metrics"
	installation "github.com/stackrox/stackrox/central/installation/store/bolt"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/telemetry/data"
	"github.com/stackrox/stackrox/pkg/telemetry/gatherers"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
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

	gatherer                     *CentralGatherer
	sensorUpgradeConfigDatastore *mocks.MockDataStore
}

func (s *gathererTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())

	boltDB, err := bolthelper.NewTemp("gatherer_test.db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)
	s.bolt = boltDB

	rocksDB := rocksdbtest.RocksDBForT(s.T())
	s.Require().NoError(err, "Failed to make RocksDB: %s", err)
	s.rocks = rocksDB

	index, err := globalindex.MemOnlyIndex()
	s.Require().NoError(err, "Failed to make in-memory Bleve: %s", err)
	s.index = index

	installationStore := installation.New(s.bolt)
	s.Require().NoError(err, "Failed to make installation store")

	s.sensorUpgradeConfigDatastore = mocks.NewMockDataStore(s.mockCtrl)
	s.sensorUpgradeConfigDatastore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Return(&storage.SensorUpgradeConfig{
		EnableAutoUpgrade: true,
	}, nil)
	s.gatherer = newCentralGatherer(installationStore, newDatabaseGatherer(newRocksDBGatherer(s.rocks), newBoltGatherer(s.bolt), newBleveGatherer(s.index)), newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()), gatherers.NewComponentInfoGatherer(), s.sensorUpgradeConfigDatastore)
}

func (s *gathererTestSuite) TearDownSuite() {
	if s.bolt != nil {
		testutils.TearDownDB(s.bolt)
	}
	if s.rocks != nil {
		rocksdbtest.TearDownRocksDB(s.rocks)
	}
}

func (s *gathererTestSuite) TestJSONSerialization() {
	metrics := s.gatherer.Gather(context.Background())

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
