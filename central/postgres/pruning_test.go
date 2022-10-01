package postgres

import (
	"context"
	"testing"

	activeComponent "github.com/stackrox/rox/central/activecomponent/datastore"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type PostgresPruningSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	ctx         context.Context
	testDB      *pgtest.TestPostgres
}

func TestPruning(t *testing.T) {
	suite.Run(t, new(PostgresPruningSuite))
}

func (s *PostgresPruningSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *PostgresPruningSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
	s.envIsolator.RestoreAll()
}

func (s *PostgresPruningSuite) TestPruneActiveComponents() {
	depStore, _ := deploymentStore.GetTestPostgresDataStore(s.T(), s.testDB.Postgres)
	acDS, err := activeComponent.NewForTestOnly(s.T(), s.testDB.Postgres)
	s.NoError(err)

	// Create and save a deployment
	deployment := &storage.Deployment{
		Id:   "TEST123",
		Name: "TestDeployment",
	}
	err = depStore.UpsertDeployment(s.ctx, deployment)
	s.Nil(err)

	activeComponents := []*storage.ActiveComponent{
		{
			Id:           "test1",
			DeploymentId: "TEST123",
		},
		{
			Id:           "test2",
			DeploymentId: "NO DEPLOYMENT",
		},
		{
			Id:           "test3",
			DeploymentId: "NO DEPLOYMENT",
		},
	}
	err = acDS.UpsertBatch(s.ctx, activeComponents)
	s.Nil(err)

	exists, err := acDS.Exists(s.ctx, "test1")
	s.Nil(err)
	s.True(exists)
	exists, err = acDS.Exists(s.ctx, "test2")
	s.Nil(err)
	s.True(exists)

	PruneActiveComponents(s.ctx, s.testDB.Postgres)

	exists, err = acDS.Exists(s.ctx, "test1")
	s.Nil(err)
	s.True(exists)
	exists, err = acDS.Exists(s.ctx, "test2")
	s.Nil(err)
	s.False(exists)
}

func (s *PostgresPruningSuite) TestPruneClusterHealthStatuses() {
	clusterDS, err := clusterStore.GetTestPostgresDataStore(s.T(), s.testDB.Postgres)
	s.Nil(err)

	clusterID, err := clusterDS.AddCluster(s.ctx, &storage.Cluster{Name: "testCluster", MainImage: "docker.io/stackrox/rox:latest"})
	s.Nil(err)

	clusterHealthStore := clusterHealthPostgresStore.New(s.testDB.Postgres)
	healthStatuses := []*storage.ClusterHealthStatus{
		{
			Id:                 clusterID,
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			Id:                    "fakeCluster",
			SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
			CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			Id:                 "randomCluster",
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
	}

	err = clusterHealthStore.UpsertMany(s.ctx, healthStatuses)
	s.Nil(err)

	count, err := clusterHealthStore.Count(s.ctx)
	s.Nil(err)
	s.Equal(count, 3)
	exists, err := clusterHealthStore.Exists(s.ctx, "randomCluster")
	s.Nil(err)
	s.True(exists)

	PruneClusterHealthStatuses(s.ctx, s.testDB.Postgres)

	count, err = clusterHealthStore.Count(s.ctx)
	s.Nil(err)
	s.Equal(count, 1)
	exists, err = clusterHealthStore.Exists(s.ctx, "randomCluster")
	s.Nil(err)
	s.False(exists)
}
