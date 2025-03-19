//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cloudsources/datastore"
	cloudSourcesManagerMocks "github.com/stackrox/rox/central/cloudsources/manager/mocks"
	"github.com/stackrox/rox/central/convert/storagetov1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources"
	cloudSourceClientMocks "github.com/stackrox/rox/pkg/cloudsources/mocks"
	"github.com/stackrox/rox/pkg/cloudsources/opts"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	mockManager := cloudSourcesManagerMocks.NewMockManager(gomock.NewController(s.T()))
	mockManager.EXPECT().ShortCircuit().AnyTimes()
	cloudSourceClientMock := cloudSourceClientMocks.NewMockClient(gomock.NewController(s.T()))
	cloudSourceClientMock.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

	s.service = newService(s.datastore, mockManager)
	s.service.(*serviceImpl).clientFactory = func(_ context.Context, _ *storage.CloudSource, _ ...opts.ClientOpts) (
		cloudsources.Client, error,
	) {
		return cloudSourceClientMock, nil
	}
}

func (s *servicePostgresTestSuite) TestCount() {
	s.addCloudSources(50)

	// 1. Count cloud sources without providing a query filter.
	resp, err := s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{})
	s.Require().NoError(err)
	s.Assert().Equal(int32(50), resp.GetCount())

	// 2.a. Filter cloud sources based on the name - no match.
	resp, err = s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"this name does not exist"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(0), resp.GetCount())

	// 2.b. Filter cloud sources based on the name - one match.
	resp, err = s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"sample name 00"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(1), resp.GetCount())

	// 3. Filter cloud sources based on the type.
	resp, err = s.service.CountCloudSources(s.readCtx, &v1.CountCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Types: []v1.CloudSource_Type{v1.CloudSource_TYPE_PALADIN_CLOUD},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(25), resp.GetCount())
}

func (s *servicePostgresTestSuite) TestGetCloudSource() {
	cloudSources := s.addCloudSources(1)

	resp, err := s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{
		Id: cloudSources[0].GetId(),
	})
	s.Require().NoError(err)
	protoassert.Equal(s.T(), cloudSources[0], resp.GetCloudSource())
	respCredentials := cloudSources[0].GetCredentials()
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetSecret())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientId())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientSecret())
}

func (s *servicePostgresTestSuite) TestListCloudSources() {
	cloudSources := s.addCloudSources(50)

	// 1. Count cloud sources without providing a query filter.
	resp, err := s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), cloudSources, resp.GetCloudSources())

	// 2.a. Filter cloud sources based on the name - no match.
	resp, err = s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"this name does not exist"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Empty(resp.GetCloudSources())

	// 2.b. Filter cloud sources based on the name - one match.
	resp, err = s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Names: []string{"sample name 00"},
		},
	})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), []*v1.CloudSource{cloudSources[0]}, resp.GetCloudSources())
	respCredentials := resp.GetCloudSources()[0].GetCredentials()
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetSecret())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientId())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientSecret())

	// 3. Filter cloud sources based on the type.
	resp, err = s.service.ListCloudSources(s.readCtx, &v1.ListCloudSourcesRequest{
		Filter: &v1.CloudSourcesFilter{
			Types: []v1.CloudSource_Type{v1.CloudSource_TYPE_PALADIN_CLOUD},
		},
	})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), cloudSources[0:25], resp.GetCloudSources())
}

func (s *servicePostgresTestSuite) TestCreateCloudSource() {
	cloudSource := fixtures.GetV1CloudSource()
	cloudSource.Id = ""

	// 1. Create new cloud source.
	createResp, err := s.service.CreateCloudSource(s.writeCtx, &v1.CreateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Require().NoError(err)
	createdCloudSource := createResp.GetCloudSource()

	// 2. Read back the created cloud source.
	getResp, err := s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: createdCloudSource.GetId()})
	s.Require().NoError(err)
	protoassert.Equal(s.T(), createdCloudSource, getResp.GetCloudSource())
	respCredentials := getResp.GetCloudSource().GetCredentials()
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetSecret())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientId())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientSecret())

	// 3. Try to create a cloud source with existing name.
	createResp, err = s.service.CreateCloudSource(s.writeCtx, &v1.CreateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Assert().Empty(createResp)
	s.Assert().ErrorIs(err, errox.InvalidArgs)
}

func (s *servicePostgresTestSuite) TestCreateCloudSourceValidation() {
	testCases := []struct {
		name          string
		cloudSourceFn func() *v1.CloudSource
	}{
		{
			name: "Invalid id",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				return cloudSource
			},
		},
		{
			name: "Invalid name",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				cloudSource.Name = ""
				return cloudSource
			},
		},
		{
			name: "Invalid credentials",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				cloudSource.Credentials = nil
				return cloudSource
			},
		},
		{
			name: "Invalid config",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				cloudSource.Config = nil
				return cloudSource
			},
		},
		{
			name: "Unspecified type",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				cloudSource.Type = v1.CloudSource_TYPE_UNSPECIFIED
				return cloudSource
			},
		},
		{
			name: "Invalid type",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				cloudSource.Type = v1.CloudSource_TYPE_PALADIN_CLOUD
				cloudSource.Config = &v1.CloudSource_Ocm{
					Ocm: &v1.OCMConfig{Endpoint: "https://api.stage.openshift.com"},
				}
				return cloudSource
			},
		},
		{
			name: "Invalid endpoint",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				cloudSource.Type = v1.CloudSource_TYPE_PALADIN_CLOUD
				cloudSource.Config = &v1.CloudSource_PaladinCloud{
					PaladinCloud: &v1.PaladinCloudConfig{Endpoint: "localhost"},
				}
				return cloudSource
			},
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			resp, err := s.service.CreateCloudSource(s.writeCtx, &v1.CreateCloudSourceRequest{
				CloudSource: testCase.cloudSourceFn(),
			})
			s.Assert().Empty(resp)
			s.Assert().ErrorIs(err, errox.InvalidArgs)
		})
	}
}

func (s *servicePostgresTestSuite) TestUpdateCloudSource() {
	cloudSource := fixtures.GetV1CloudSource()

	// 1. Create new cloud source.
	updateResp, err := s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
		CloudSource:       cloudSource,
		UpdateCredentials: true,
	})
	s.Require().NoError(err)
	protoassert.Equal(s.T(), &v1.Empty{}, updateResp)

	// 2. Read back the created cloud source.
	getResp, err := s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: cloudSource.GetId()})
	s.Require().NoError(err)
	cloudSource.Credentials = &v1.CloudSource_Credentials{
		Secret:       secrets.ScrubReplacementStr,
		ClientId:     secrets.ScrubReplacementStr,
		ClientSecret: secrets.ScrubReplacementStr,
	}
	protoassert.Equal(s.T(), cloudSource, getResp.GetCloudSource())
	respCredentials := getResp.GetCloudSource().GetCredentials()
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetSecret())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientId())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientSecret())

	// 3. Try to create a cloud source with existing name.
	cloudSource.Id = uuid.NewV4().String()
	updateResp, err = s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Assert().Empty(updateResp)
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	// 4. Update existing cloud source name without updating credentials.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Name = "updated-name"
	cloudSource.Credentials = nil
	updateResp, err = s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
		CloudSource:       cloudSource,
		UpdateCredentials: false,
	})
	protoassert.Equal(s.T(), &v1.Empty{}, updateResp)
	s.Require().NoError(err)

	// 5. Read back the updated cloud source.
	getResp, err = s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: cloudSource.GetId()})
	s.Require().NoError(err)
	cloudSource.Credentials = &v1.CloudSource_Credentials{
		Secret:       secrets.ScrubReplacementStr,
		ClientId:     secrets.ScrubReplacementStr,
		ClientSecret: secrets.ScrubReplacementStr,
	}
	protoassert.Equal(s.T(), cloudSource, getResp.GetCloudSource())
	respCredentials = getResp.GetCloudSource().GetCredentials()
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetSecret())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientId())
	s.Assert().Equal(secrets.ScrubReplacementStr, respCredentials.GetClientSecret())
}

func (s *servicePostgresTestSuite) TestUpdateCloudSourceValidation() {
	testCases := []struct {
		name          string
		cloudSourceFn func() *v1.CloudSource
	}{
		{
			name: "Invalid id",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Id = ""
				return cloudSource
			},
		},
		{
			name: "Invalid name",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Name = ""
				return cloudSource
			},
		},
		{
			name: "Invalid credentials",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Credentials = nil
				return cloudSource
			},
		},
		{
			name: "Invalid config",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Config = nil
				return cloudSource
			},
		},
		{
			name: "Unspecified type",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Type = v1.CloudSource_TYPE_UNSPECIFIED
				return cloudSource
			},
		},
		{
			name: "Invalid type",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Type = v1.CloudSource_TYPE_PALADIN_CLOUD
				cloudSource.Config = &v1.CloudSource_Ocm{
					Ocm: &v1.OCMConfig{Endpoint: "https://api.stage.openshift.com"},
				}
				return cloudSource
			},
		},
		{
			name: "Invalid endpoint",
			cloudSourceFn: func() *v1.CloudSource {
				cloudSource := fixtures.GetV1CloudSource()
				cloudSource.Type = v1.CloudSource_TYPE_PALADIN_CLOUD
				cloudSource.Config = &v1.CloudSource_PaladinCloud{
					PaladinCloud: &v1.PaladinCloudConfig{Endpoint: "localhost"},
				}
				return cloudSource
			},
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			resp, err := s.service.UpdateCloudSource(s.writeCtx, &v1.UpdateCloudSourceRequest{
				CloudSource:       testCase.cloudSourceFn(),
				UpdateCredentials: true,
			})
			s.Assert().Empty(resp)
			s.Assert().ErrorIs(err, errox.InvalidArgs)
		})
	}
}

func (s *servicePostgresTestSuite) TestDeleteCloudSource() {
	cloudSources := s.addCloudSources(1)

	deleteResp, err := s.service.DeleteCloudSource(s.writeCtx, &v1.DeleteCloudSourceRequest{
		Id: cloudSources[0].GetId(),
	})
	protoassert.Equal(s.T(), &v1.Empty{}, deleteResp)
	s.Require().NoError(err)

	_, err = s.service.GetCloudSource(s.readCtx, &v1.GetCloudSourceRequest{Id: cloudSources[0].GetId()})
	s.Assert().ErrorIs(err, errox.NotFound)
}

func (s *servicePostgresTestSuite) TestCloudSourceTest() {
	cloudSourceClientMock := cloudSourceClientMocks.NewMockClient(gomock.NewController(s.T()))
	s.service.(*serviceImpl).clientFactory = func(_ context.Context, _ *storage.CloudSource, _ ...opts.ClientOpts) (
		cloudsources.Client, error,
	) {
		return cloudSourceClientMock, nil
	}

	cloudSource := fixtures.GetV1CloudSource()

	// 1. Call test without UpdateCredentials on a non-existent cloud source.
	_, err := s.service.TestCloudSource(s.writeCtx, &v1.TestCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Assert().ErrorIs(err, errox.InvalidArgs)

	// 2. Call test with UpdateCredentials without any error during calling client.
	cloudSourceClientMock.EXPECT().Ping(gomock.Any()).Return(nil)
	_, err = s.service.TestCloudSource(s.writeCtx, &v1.TestCloudSourceRequest{
		CloudSource:       cloudSource,
		UpdateCredentials: true,
	})
	s.Assert().NoError(err)

	// 3. Call test with UpdateCredentials with an error during calling client.
	clientErr := errors.New("failed calling client")
	cloudSourceClientMock.EXPECT().Ping(gomock.Any()).
		Return(clientErr)
	_, err = s.service.TestCloudSource(s.writeCtx, &v1.TestCloudSourceRequest{
		CloudSource:       cloudSource,
		UpdateCredentials: true,
	})
	s.Assert().ErrorContains(err, clientErr.Error())

	// 4. Call test without UpdateCredentials with an existent cloud source.
	cloudSource.Id = ""
	resp, err := s.service.CreateCloudSource(s.writeCtx, &v1.CreateCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Require().NoError(err)
	cloudSource = resp.GetCloudSource()
	cloudSourceClientMock.EXPECT().Ping(gomock.Any()).Return(nil)

	_, err = s.service.TestCloudSource(s.writeCtx, &v1.TestCloudSourceRequest{
		CloudSource: cloudSource,
	})
	s.Assert().NoError(err)
}

func (s *servicePostgresTestSuite) addCloudSources(num int) []*v1.CloudSource {
	cloudSources := fixtures.GetManyStorageCloudSources(num)
	result := []*v1.CloudSource{}
	for _, cs := range cloudSources {
		s.Require().NoError(s.datastore.UpsertCloudSource(s.writeCtx, cs))
		result = append(result, storagetov1.CloudSource(cs))
	}
	return result
}
