package resolvers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/views/imagecve"
	imageCVEViewMock "github.com/stackrox/rox/central/views/imagecve/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/pointers"
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

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ).Return(nil, nil)
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

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
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

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}
