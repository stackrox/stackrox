package gatherers

import (
	"encoding/json"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/grpc/metrics"
	installation "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestGatherers(t *testing.T) {
	suite.Run(t, new(gathererTestSuite))
}

type gathererTestSuite struct {
	suite.Suite

	bolt     *bbolt.DB
	badger   *badger.DB
	index    bleve.Index
	dir      string
	gatherer *CentralGatherer
}

func (s *gathererTestSuite) SetupSuite() {
	boltDB, err := bolthelper.NewTemp("gatherer_test.db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)
	s.bolt = boltDB

	badgerDB, dir, err := badgerhelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BadgerDB: %s", err)
	s.badger = badgerDB
	s.dir = dir

	index, err := globalindex.MemOnlyIndex()
	s.Require().NoError(err, "Failed to make in-memory Bleve: %s", err)
	s.index = index

	installationStore := installation.New(s.bolt)
	s.Require().NoError(err, "Failed to make installation store")

	s.gatherer = newCentralGatherer(nil, installationStore, newDatabaseGatherer(newBadgerGatherer(s.badger), newBoltGatherer(s.bolt), newBleveGatherer(s.index)), newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()), gatherers.NewComponentInfoGatherer())
}

func (s *gathererTestSuite) TearDownSuite() {
	if s.bolt != nil {
		testutils.TearDownDB(s.bolt)
	}
	if s.badger != nil {
		testutils.TearDownBadger(s.badger, s.dir)
	}
}

func (s *gathererTestSuite) TestJSONSerialization() {
	metrics := s.gatherer.Gather()

	bytes, err := json.Marshal(metrics)
	s.NoError(err)

	marshalledMetrics := &data.CentralInfo{}
	err = json.Unmarshal(bytes, &marshalledMetrics)
	s.NoError(err)

	s.Equal(metrics.Orchestrator, marshalledMetrics.Orchestrator)
	s.Equal(metrics.Errors, marshalledMetrics.Errors)
	s.Equal(metrics.Storage, marshalledMetrics.Storage)
	s.Equal(metrics.License, marshalledMetrics.License)
	s.Equal(metrics.Process, marshalledMetrics.Process)
	s.Equal(metrics.Restarts, marshalledMetrics.Restarts)
	s.Equal(metrics.Version, marshalledMetrics.Version)
	s.Equal(metrics.RoxComponentInfo, marshalledMetrics.RoxComponentInfo)
	// API stats will be empty so the marshalled metrics will contain nil instead of empty
	s.Nil(marshalledMetrics.APIStats.HTTP)
	s.Nil(marshalledMetrics.APIStats.GRPC)
}
