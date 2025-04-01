//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/convert/typetostorage"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestDiscoveredClustersDatastorePostgres(t *testing.T) {
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	readCtx      context.Context
	writeCtx     context.Context
	postgresTest *pgtest.TestPostgres
	datastore    DataStore
}

func (s *datastorePostgresTestSuite) SetupTest() {
	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	s.datastore = GetTestPostgresDataStore(s.T(), s.postgresTest.DB)
}

func (s *datastorePostgresTestSuite) TestCountDiscoveredClusters() {
	count, err := s.datastore.CountDiscoveredClusters(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Zero(count)

	s.addDiscoveredClusters(100)

	count, err = s.datastore.CountDiscoveredClusters(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Equal(100, count)
}

func (s *datastorePostgresTestSuite) TestGetNonExistingDiscoveredCluster() {
	nonExistingID := "00000000-0000-0000-0000-000000000000"
	discoveredCluster, err := s.datastore.GetDiscoveredCluster(s.readCtx, nonExistingID)
	s.Assert().ErrorIs(err, errox.NotFound)
	s.Assert().Empty(discoveredCluster)
}

func (s *datastorePostgresTestSuite) TestUpsertAndGetDiscoveredCluster() {
	fakeCluster := fixtures.GetDiscoveredCluster()
	err := s.datastore.UpsertDiscoveredClusters(s.writeCtx, fakeCluster)
	s.Require().NoError(err)

	expectedCluster := typetostorage.DiscoveredCluster(fakeCluster)
	roundtripCluster, err := s.datastore.GetDiscoveredCluster(s.readCtx, expectedCluster.GetId())
	s.Require().NoError(err)
	expectedCluster.LastUpdatedAt = roundtripCluster.LastUpdatedAt
	protoassert.Equal(s.T(), expectedCluster, roundtripCluster)
}

func (s *datastorePostgresTestSuite) TestListDiscoveredClusters() {
	discoveredClusters, err := s.datastore.ListDiscoveredClusters(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Empty(discoveredClusters)

	s.addDiscoveredClusters(100)

	discoveredClusters, err = s.datastore.ListDiscoveredClusters(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Len(discoveredClusters, 100)
}

func (s *datastorePostgresTestSuite) TestUpsertDiscoveredClusters_InvalidArgument() {
	err := s.datastore.UpsertDiscoveredClusters(s.writeCtx, nil)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	fakeCluster := fixtures.GetDiscoveredCluster()
	fakeCluster.ID = ""
	err = s.datastore.UpsertDiscoveredClusters(s.writeCtx, fakeCluster)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	fakeCluster = fixtures.GetDiscoveredCluster()
	fakeCluster.Name = ""
	err = s.datastore.UpsertDiscoveredClusters(s.writeCtx, fakeCluster)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	fakeCluster = fixtures.GetDiscoveredCluster()
	fakeCluster.CloudSourceID = ""
	err = s.datastore.UpsertDiscoveredClusters(s.writeCtx, fakeCluster)
	s.Assert().ErrorIs(err, errox.InvalidArgs)
}

func (s *datastorePostgresTestSuite) TestDeleteDiscoveredCluster() {
	s.addDiscoveredClusters(100)

	result, err := s.datastore.DeleteDiscoveredClusters(s.writeCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Len(result, 100)

	count, err := s.datastore.CountDiscoveredClusters(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Equal(0, count)
}

func (s *datastorePostgresTestSuite) addDiscoveredClusters(num int) {
	fakeClusters := fixtures.GetManyDiscoveredClusters(num)
	s.Require().NoError(s.datastore.UpsertDiscoveredClusters(s.writeCtx, fakeClusters...))
}
