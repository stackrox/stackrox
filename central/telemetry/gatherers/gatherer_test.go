//go:build sql_integration

package gatherers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/central/grpc/metrics"
	installationPostgres "github.com/stackrox/rox/central/installation/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGatherers(t *testing.T) {
	suite.Run(t, new(gathererTestSuite))
}

type gathererTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	tp *pgtest.TestPostgres

	gatherer *CentralGatherer
}

func (s *gathererTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())

	s.tp = pgtest.ForTCustomDB(s.T(), "postgres")
	source := pgtest.GetConnectionString(s.T())
	adminConfig, err := postgres.ParseConfig(source)
	s.NoError(err)

	installationStore := installationPostgres.New(s.tp.DB)

	s.gatherer = newCentralGatherer(installationStore, newDatabaseGatherer(newPostgresGatherer(s.tp.DB, adminConfig)), newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()), gatherers.NewComponentInfoGatherer())
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
