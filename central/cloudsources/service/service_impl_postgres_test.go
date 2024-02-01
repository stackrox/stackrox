//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cloudsources/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestServicePostgres(t *testing.T) {
	suite.Run(t, new(servicePostgresTestSuite))
}

type servicePostgresTestSuite struct {
	suite.Suite

	readCtx   context.Context
	writeCtx  context.Context
	pool      *pgtest.TestPostgres
	datastore datastore.DataStore
	service   Service
}

func (s *servicePostgresTestSuite) SetupTest() {
	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)
	s.datastore = datastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.service = newService(s.datastore)
}

func (s *servicePostgresTestSuite) TearDownTest() {
	s.pool.Teardown(s.T())
	s.pool.Close()
}

func (s *servicePostgresTestSuite) TestCount() {
	s.addCloudSources(50)

	// 1. Count cloud sources without providing a query filter.
	resp, err := s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{})
	s.NoError(err)
	s.Equal(int32(50), resp.GetCount())

	// 2.a. Filter cloud sources based on the name - no match.
	resp, err = s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"this name does not exist"},
		},
	})
	s.NoError(err)
	s.Equal(int32(0), resp.GetCount())

	// 2.b. Filter cloud sources based on the name - one match.
	resp, err = s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"sample name 0"},
		},
	})
	s.NoError(err)
	s.Equal(int32(1), resp.GetCount())

	// 3. Filter cloud sources based on the type.
	resp, err = s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Types: []v1.CloudSource_Type{v1.CloudSource_TYPE_PALADIN_CLOUD},
		},
	})
	s.NoError(err)
	s.Equal(int32(25), resp.GetCount())
}

func (s *servicePostgresTestSuite) TestGetCloudSource() {
	cloudSources := s.addCloudSources(1)

	resp, err := s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{
		Id: cloudSources[0].GetId(),
	})
	s.NoError(err)
	s.Equal(cloudSources[0], resp.GetCloudSource())
}

func (s *servicePostgresTestSuite) TestListCloudSources() {
	cloudSources := s.addCloudSources(50)

	// 1. Count cloud sources without providing a query filter.
	resp, err := s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{})
	s.NoError(err)
	s.Equal(cloudSources, resp.GetCloudSources())

	// 2.a. Filter cloud sources based on the name - no match.
	resp, err = s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"this name does not exist"},
		},
	})
	s.NoError(err)
	s.Empty(resp.GetCloudSources())

	// 2.b. Filter cloud sources based on the name - one match.
	resp, err = s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"sample name 0"},
		},
	})
	s.NoError(err)
	s.Equal([]*v1.CloudSource{cloudSources[0]}, resp.GetCloudSources())

	// 3. Filter cloud sources based on the type.
	resp, err = s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Types: []v1.CloudSource_Type{v1.CloudSource_TYPE_PALADIN_CLOUD},
		},
	})
	s.NoError(err)
	s.Equal(cloudSources[0:25], resp.GetCloudSources())
}

func (s *servicePostgresTestSuite) TestCreateCloudSource() {
	cloudSource := fixtures.GetV1CloudSource()
	cloudSource.Id = ""

	// 1. Create new cloud source.
	postResp, err := s.service.CreateCloudSource(s.writeCtx, &v1.CreateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.NoError(err)
	createdCloudSource := postResp.GetCloudSource()

	// 2. Read back the created cloud source.
	getResp, err := s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: createdCloudSource.GetId()})
	s.NoError(err)
	s.Equal(createdCloudSource, getResp.GetCloudSource())

	// 3. Try to create a cloud source with existing name.
	postResp, err = s.service.CreateCloudSource(s.writeCtx, &v1.CreateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Empty(postResp)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *servicePostgresTestSuite) TestUpdateCloudSource() {
	cloudSource := fixtures.GetV1CloudSource()

	// 1. Create new cloud source.
	putResp, err := s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
		CloudSource:       cloudSource,
		UpdateCredentials: true,
	})
	s.Equal(&v1.Empty{}, putResp)
	s.NoError(err)

	// 2. Read back the created cloud source.
	getResp, err := s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: cloudSource.GetId()})
	s.NoError(err)
	cloudSource.Credentials = nil
	s.Equal(cloudSource, getResp.GetCloudSource())

	// 3. Try to create a cloud source with existing name.
	cloudSource.Id = uuid.NewV4().String()
	putResp, err = s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Empty(putResp)
	s.ErrorIs(err, errox.InvalidArgs)

	// 4. Update existing cloud source name without updating credentials.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Name = "updated-name"
	cloudSource.Credentials = nil
	putResp, err = s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
		CloudSource:       cloudSource,
		UpdateCredentials: false,
	})
	s.Equal(&v1.Empty{}, putResp)
	s.NoError(err)

	// 5. Read back the updated cloud source.
	getResp, err = s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: cloudSource.GetId()})
	s.NoError(err)
	s.Equal(cloudSource, getResp.GetCloudSource())
}

func (s *servicePostgresTestSuite) TestDeleteCloudSource() {
	cloudSources := s.addCloudSources(1)

	deleteResp, err := s.service.DeleteCloudSource(s.writeCtx, &v1.DeleteCloudSourceRequest{
		Id: cloudSources[0].GetId(),
	})
	s.Equal(&v1.Empty{}, deleteResp)
	s.NoError(err)

	_, err = s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: cloudSources[0].GetId()})
	s.ErrorIs(err, errox.NotFound)
}

func (s *servicePostgresTestSuite) addCloudSources(num int) []*v1.CloudSource {
	cloudSources := fixtures.GetManyStorageCloudSources(num)
	result := []*v1.CloudSource{}
	for _, cs := range cloudSources {
		s.Require().NoError(s.datastore.UpsertCloudSource(s.writeCtx, cs))
		result = append(result, toV1Proto(cs))
	}
	return result
}
