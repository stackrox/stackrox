//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestCloudSourcesDatastorePostgres(t *testing.T) {
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
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration, resources.Administration),
		),
	)

	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	s.datastore = GetTestPostgresDataStore(s.T(), s.postgresTest.DB)
}

func (s *datastorePostgresTestSuite) TestCountCloudSources() {
	count, err := s.datastore.CountCloudSources(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Zero(count)

	s.addCloudSources(100)

	count, err = s.datastore.CountCloudSources(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Equal(100, count)
}

func (s *datastorePostgresTestSuite) TestGetCloudSource() {
	nonExistingID := "00000000-0000-0000-0000-000000000000"
	cloudSource, err := s.datastore.GetCloudSource(s.readCtx, nonExistingID)
	s.Assert().ErrorIs(err, errox.NotFound)
	s.Assert().Empty(cloudSource)

	cloudSource = fixtures.GetStorageCloudSource()
	err = s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Require().NoError(err)

	roundtripCloudSource, err := s.datastore.GetCloudSource(s.readCtx, cloudSource.GetId())
	s.Require().NoError(err)
	protoassert.Equal(s.T(), cloudSource, roundtripCloudSource)
}

func (s *datastorePostgresTestSuite) TestProcessCloudSources() {
	count := 0
	counter := func(_ *storage.CloudSource) error {
		count++
		return nil
	}

	err := s.datastore.ProcessCloudSources(s.readCtx, counter)
	s.Require().NoError(err)
	s.Assert().Equal(count, 0)

	s.addCloudSources(100)

	err = s.datastore.ProcessCloudSources(s.readCtx, counter)
	s.Require().NoError(err)
	s.Assert().Equal(count, 100)
}

func (s *datastorePostgresTestSuite) TestListCloudSources() {
	cloudSources, err := s.datastore.ListCloudSources(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Empty(cloudSources)

	s.addCloudSources(100)

	cloudSources, err = s.datastore.ListCloudSources(s.readCtx, &v1.Query{})
	s.Require().NoError(err)
	s.Assert().Len(cloudSources, 100)
}

func (s *datastorePostgresTestSuite) TestUpsertCloudSource_Success() {
	cloudSource := fixtures.GetStorageCloudSource()
	err := s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Require().NoError(err)

	roundtripCloudSource, err := s.datastore.GetCloudSource(s.readCtx, cloudSource.GetId())
	s.Require().NoError(err)
	protoassert.Equal(s.T(), cloudSource, roundtripCloudSource)
}

func (s *datastorePostgresTestSuite) TestUpsertCloudSource_InvalidArgument() {
	err := s.datastore.UpsertCloudSource(s.writeCtx, nil)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	cloudSource := fixtures.GetStorageCloudSource()
	cloudSource.Id = ""
	err = s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	cloudSource = fixtures.GetStorageCloudSource()
	cloudSource.Name = ""
	err = s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	cloudSource = fixtures.GetStorageCloudSource()
	cloudSource.Credentials = nil
	err = s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	cloudSource = fixtures.GetStorageCloudSource()
	cloudSource.Config = nil
	err = s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Assert().ErrorIs(err, errox.InvalidArgs)
}

func (s *datastorePostgresTestSuite) TestDeleteCloudSource() {
	cloudSource := fixtures.GetStorageCloudSource()
	discoveredClusters := fixtures.GetManyDiscoveredClusters(10)
	cloudSource.Id = discoveredClusters[0].GetCloudSourceID()
	err := s.datastore.UpsertCloudSource(s.writeCtx, cloudSource)
	s.Require().NoError(err)

	err = s.datastore.(*datastoreImpl).discoveredClusterDS.UpsertDiscoveredClusters(s.writeCtx, discoveredClusters...)
	s.Require().NoError(err)

	err = s.datastore.DeleteCloudSource(s.writeCtx, cloudSource.GetId())
	s.Require().NoError(err)

	cloudSource, err = s.datastore.GetCloudSource(s.readCtx, cloudSource.GetId())
	s.Assert().ErrorIs(err, errox.NotFound)
	s.Assert().Empty(cloudSource)

	storedDiscoveredClusters, err :=
		s.datastore.(*datastoreImpl).discoveredClusterDS.ListDiscoveredClusters(s.writeCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Len(storedDiscoveredClusters, 5)
}

func (s *datastorePostgresTestSuite) addCloudSources(num int) {
	cloudSources := fixtures.GetManyStorageCloudSources(num)
	for _, cs := range cloudSources {
		s.Require().NoError(s.datastore.UpsertCloudSource(s.writeCtx, cs))
	}
}
