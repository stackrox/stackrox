package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	dsMocks "github.com/stackrox/rox/central/cloudsources/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake = errors.New("fake error")
	fakeID  = "0925514f-3a33-5931-b431-756406e1a008"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestCloudSourcesService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(cloudSourcesTestSuite))
}

type cloudSourcesTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx           context.Context
	datastoreMock *dsMocks.MockDataStore
	service       Service
}

func (s *cloudSourcesTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.datastoreMock = dsMocks.NewMockDataStore(s.mockCtrl)
	s.service = newService(s.datastoreMock)
}

// Test CountCloudSource

func (s *cloudSourcesTestSuite) TestCountCloudSources_Success() {
	s.datastoreMock.EXPECT().CountCloudSources(s.ctx, gomock.Any()).Return(1, nil)
	resp, err := s.service.CountCloudSources(s.ctx, &v1.CountCloudSourcesRequest{})

	s.Require().NoError(err)
	s.Equal(&v1.CountCloudSourcesResponse{Count: 1}, resp)
}

func (s *cloudSourcesTestSuite) TestCountCloudSources_Error() {
	s.datastoreMock.EXPECT().CountCloudSources(s.ctx, gomock.Any()).Return(0, errFake)
	resp, err := s.service.CountCloudSources(s.ctx, &v1.CountCloudSourcesRequest{})

	s.ErrorIs(err, errFake)
	s.Nil(resp)
}

// Test GetCloudSource

func (s *cloudSourcesTestSuite) TestGetCloudSource_Success() {
	s.datastoreMock.EXPECT().GetCloudSource(s.ctx, fakeID).Return(fixtures.GetStorageCloudSource(), nil)
	resp, err := s.service.GetCloudSource(s.ctx, &v1.GetCloudSourceRequest{Id: fakeID})

	s.Require().NoError(err)
	s.Equal(&v1.GetCloudSourceResponse{CloudSource: toV1Proto(fixtures.GetStorageCloudSource())}, resp)
}

func (s *cloudSourcesTestSuite) TestGetCloudSource_Error() {
	s.datastoreMock.EXPECT().GetCloudSource(s.ctx, fakeID).Return(nil, errFake)
	resp, err := s.service.GetCloudSource(s.ctx, &v1.GetCloudSourceRequest{Id: fakeID})

	s.ErrorIs(err, errFake)
	s.Nil(resp)
}

func (s *cloudSourcesTestSuite) TestGetCloudSource_NotFound() {
	s.datastoreMock.EXPECT().GetCloudSource(s.ctx, fakeID).Return(nil, errox.NotFound)
	resp, err := s.service.GetCloudSource(s.ctx, &v1.GetCloudSourceRequest{Id: fakeID})

	s.ErrorIs(err, errox.NotFound)
	s.Nil(resp)
}

// Test ListCloudSources

func (s *cloudSourcesTestSuite) TestListCloudSources_Success() {
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).
		Return([]*storage.CloudSource{fixtures.GetStorageCloudSource()}, nil)
	resp, err := s.service.ListCloudSources(s.ctx, &v1.ListCloudSourcesRequest{})

	s.Require().NoError(err)
	expected := []*v1.CloudSource{toV1Proto(fixtures.GetStorageCloudSource())}
	s.Equal(expected, resp.GetCloudSources())
}

func (s *cloudSourcesTestSuite) TestListCloudSources_Error() {
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, errFake)
	resp, err := s.service.ListCloudSources(s.ctx, &v1.ListCloudSourcesRequest{})

	s.ErrorIs(err, errFake)
	s.Nil(resp)
}

// Test CreateCloudSource

func (s *cloudSourcesTestSuite) TestCreateCloudSource_Success() {
	cloudSource := fixtures.GetV1CloudSource()
	cloudSource.Id = ""
	s.datastoreMock.EXPECT().UpsertCloudSource(s.ctx, gomock.Any()).Return(nil)
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, nil)
	resp, err := s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)

	s.Require().NoError(err)
	s.Equal(cloudSource.GetName(), resp.GetCloudSource().GetName())
	s.Equal(cloudSource.GetConfig(), resp.GetCloudSource().GetConfig())
}

func (s *cloudSourcesTestSuite) TestCreateCloudSource_Validate() {
	// Invalid ID.
	cloudSource := fixtures.GetV1CloudSource()
	cloudSource.Id = "invalid-id"
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, nil).Times(2)
	resp, err := s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)
	s.Error(err)
	s.Nil(resp)

	// Invalid Name.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Id = ""
	cloudSource.Name = ""
	resp, err = s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	// Invalid Type.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Type = v1.CloudSource_TYPE_UNSPECIFIED
	resp, err = s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Config = &v1.CloudSource_Ocm{}
	resp, err = s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	// Invalid Credentials.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Id = ""
	cloudSource.Credentials = nil
	resp, err = s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	// Invalid Endpoint.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Id = ""
	cloudSource.Config = &v1.CloudSource_PaladinCloud{
		PaladinCloud: &v1.PaladinCloudConfig{Endpoint: "localhost"},
	}
	resp, err = s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)
}

func (s *cloudSourcesTestSuite) TestCreateCloudSource_Error() {
	cloudSource := fixtures.GetV1CloudSource()
	cloudSource.Id = ""
	s.datastoreMock.EXPECT().UpsertCloudSource(s.ctx, gomock.Any()).Return(errFake)
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, nil)
	s.datastoreMock.EXPECT().DeleteCloudSource(s.ctx, gomock.Any())
	resp, err := s.service.CreateCloudSource(s.ctx,
		&v1.CreateCloudSourceRequest{CloudSource: cloudSource},
	)

	s.ErrorIs(err, errFake)
	s.Nil(resp)
}

// Test UpdateCloudSource

func (s *cloudSourcesTestSuite) TestUpdateCloudSource_Success() {
	cloudSource := fixtures.GetV1CloudSource()
	s.datastoreMock.EXPECT().UpsertCloudSource(s.ctx, gomock.Any()).Return(nil)
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, nil)
	resp, err := s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)

	s.Equal(&v1.Empty{}, resp)
	s.Require().NoError(err)
}

func (s *cloudSourcesTestSuite) TestUpdateCloudSource_Validate() {
	// Invalid Name.
	cloudSource := fixtures.GetV1CloudSource()
	cloudSource.Name = ""
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, nil).Times(4)
	resp, err := s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	// Invalid Type.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Type = v1.CloudSource_TYPE_UNSPECIFIED
	resp, err = s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Config = &v1.CloudSource_Ocm{}
	resp, err = s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	// Invalid Credentials.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Credentials = nil
	resp, err = s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)

	// Invalid Endpoint.
	cloudSource = fixtures.GetV1CloudSource()
	cloudSource.Config = &v1.CloudSource_PaladinCloud{
		PaladinCloud: &v1.PaladinCloudConfig{Endpoint: "localhost"},
	}
	resp, err = s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)
}

func (s *cloudSourcesTestSuite) TestUpdateCloudSources_Error() {
	cloudSource := fixtures.GetV1CloudSource()
	s.datastoreMock.EXPECT().UpsertCloudSource(s.ctx, gomock.Any()).Return(errFake)
	s.datastoreMock.EXPECT().ListCloudSources(s.ctx, gomock.Any()).Return(nil, nil)
	resp, err := s.service.UpdateCloudSource(s.ctx,
		&v1.UpdateCloudSourceRequest{CloudSource: cloudSource, UpdateCredentials: true},
	)

	s.ErrorIs(err, errFake)
	s.Nil(resp)
}

// Test DeleteCloudSource

func (s *cloudSourcesTestSuite) TestDeleteCloudSource_Success() {
	s.datastoreMock.EXPECT().DeleteCloudSource(s.ctx, gomock.Any()).Return(nil)
	resp, err := s.service.DeleteCloudSource(s.ctx,
		&v1.DeleteCloudSourceRequest{Id: fakeID},
	)

	s.Equal(&v1.Empty{}, resp)
	s.Require().NoError(err)
}

func (s *cloudSourcesTestSuite) TestDeleteCloudSources_Error() {
	s.datastoreMock.EXPECT().DeleteCloudSource(s.ctx, gomock.Any()).Return(errFake)
	resp, err := s.service.DeleteCloudSource(s.ctx,
		&v1.DeleteCloudSourceRequest{Id: fakeID},
	)

	s.ErrorIs(err, errFake)
	s.Nil(resp)
}

// Test query builder

func TestCloudSourcesQueryBuilder(t *testing.T) {
	t.Parallel()
	filter := &v1.CloudSourcesFilter{
		Names: []string{"my-integration"},
		Types: []v1.CloudSource_Type{v1.CloudSource_TYPE_PALADIN_CLOUD, v1.CloudSource_TYPE_OCM},
	}
	queryBuilder := getQueryBuilderFromFilter(filter)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Contains(t, rawQuery, `Integration Name:"my-integration"`)
	assert.Contains(t, rawQuery, `Integration Type:"TYPE_OCM","TYPE_PALADIN_CLOUD"`)
}

func TestCloudSourcesQueryBuilderNilFilter(t *testing.T) {
	t.Parallel()
	queryBuilder := getQueryBuilderFromFilter(nil)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Empty(t, rawQuery)
}
