package resolvers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/imagecve"
	imageCVEViewMock "github.com/stackrox/rox/central/views/imagecve/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestImageCVECoreResolver(t *testing.T) {
	suite.Run(t, new(ImageCVECoreResolverTestSuite))
}

type ImageCVECoreResolverTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	ctx      context.Context

	imageCVEView *imageCVEViewMock.MockCveView

	resolver *Resolver
}

func (s *ImageCVECoreResolverTestSuite) SetupSuite() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")
	s.T().Setenv(features.VulnMgmtWorkloadCVEs.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() || !features.VulnMgmtWorkloadCVEs.Enabled() {
		s.T().Skipf("Skiping test. %s=false %s=false", env.PostgresDatastoreEnabled.EnvVar(), features.VulnMgmtWorkloadCVEs.EnvVar())
		s.T().SkipNow()
	}

	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithImagePerm(s.T(), s.mockCtrl)
	s.imageCVEView = imageCVEViewMock.NewMockCveView(s.mockCtrl)
	s.resolver, _ = SetupTestResolver(s.T(), s.imageCVEView)
}

func (s *ImageCVECoreResolverTestSuite) TearDownSuite() {}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEsNoImagePerm() {
	q := &PaginatedQuery{}
	response, err := s.resolver.ImageCVEs(context.Background(), *q)
	s.Error(err)
	s.Nil(response)
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEsEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(nil, nil)
	response, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 0)
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEsNonEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	expected := []imagecve.CveCore{
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(expected, nil)
	response, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEsQuery() {
	q := &PaginatedQuery{
		Query: pointers.String("CVE:cve-2022-xyz"),
	}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	expected := []imagecve.CveCore{
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(expected, nil)
	response, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *ImageCVECoreResolverTestSuite) TestCountImageCVEsNoImagePerm() {
	q := &PaginatedQuery{}
	response, err := s.resolver.ImageCVEs(context.Background(), *q)
	s.Error(err)
	s.Nil(response)
}

func (s *ImageCVECoreResolverTestSuite) TestCountImageCVEs() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.imageCVEView.EXPECT().Count(s.ctx, expectedQ).Return(0, nil)
	response, err := s.resolver.ImageCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(0))
}

func (s *ImageCVECoreResolverTestSuite) TestCountImageCVEsWithQuery() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.imageCVEView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)
	response, err := s.resolver.ImageCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(3))
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEMalformed() {
	_, err := s.resolver.ImageCVE(s.ctx, struct{ Cve *string }{})
	s.Error(err)
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVENonEmpty() {
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-xyz").ProtoQuery()
	expected := []imagecve.CveCore{
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(expected, nil)
	response, err := s.resolver.ImageCVE(s.ctx, struct{ Cve *string }{Cve: pointers.String("cve-xyz")})
	s.NoError(err)
	s.NotNil(response.data)
}
