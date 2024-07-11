package resolvers

import (
	"context"
	"math"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/imagecve"
	imageCVEViewMock "github.com/stackrox/rox/central/views/imagecve/mocks"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	s.T().Setenv(features.VulnMgmtWorkloadCVEs.EnvVar(), "true")

	if !features.VulnMgmtWorkloadCVEs.Enabled() {
		s.T().Skipf("Skiping test. %s=false", features.VulnMgmtWorkloadCVEs.EnvVar())
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

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEsWithQuery() {
	q := &PaginatedQuery{
		Query: pointers.String("CVE:cve-2022-xyz"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.CVE, "cve-2022-xyz").
		WithPagination(search.NewPagination().Limit(math.MaxInt32)).ProtoQuery()

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

func (s *ImageCVECoreResolverTestSuite) TestImageCVEsWithPaginatedQuery() {
	q := &PaginatedQuery{
		Pagination: &inputtypes.Pagination{
			SortOption: &inputtypes.SortOption{
				Field: pointers.String(search.CVSS.String()),
				AggregateBy: &inputtypes.AggregateBy{
					AggregateFunc: pointers.String(aggregatefunc.Max.Name()),
				},
			},
		},
	}
	expectedQ := search.NewQueryBuilder().WithPagination(
		search.NewPagination().AddSortOption(
			search.NewSortOption(search.CVSS).AggregateBy(aggregatefunc.Max, false),
		).Limit(math.MaxInt32),
	).ProtoQuery()

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(nil, nil)
	_, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
}

func (s *ImageCVECoreResolverTestSuite) TestImageCVEsNoImagePerm() {
	response, err := s.resolver.ImageCVEs(context.Background(), PaginatedQuery{})
	s.Error(err)
	s.Nil(response)
}

func (s *ImageCVECoreResolverTestSuite) TestImageCVECountNoImagePerm() {
	response, err := s.resolver.ImageCVECount(context.Background(), RawQuery{})
	s.Error(err)
	s.Zero(response)
}

func (s *ImageCVECoreResolverTestSuite) TestImageCVECount() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.imageCVEView.EXPECT().Count(s.ctx, expectedQ).Return(0, nil)
	response, err := s.resolver.ImageCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(0))
}

func (s *ImageCVECoreResolverTestSuite) TestImageCVECountWithQuery() {
	q := &RawQuery{
		Query: pointers.String("Image:image"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.ImageName, "image").ProtoQuery()

	s.imageCVEView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)
	response, err := s.resolver.ImageCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(3))
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVEMalformed() {
	_, err := s.resolver.ImageCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{})
	s.Error(err)
}

func (s *ImageCVECoreResolverTestSuite) TestGetImageCVENonEmpty() {
	// without filter
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-xyz").ProtoQuery()
	expected := []imagecve.CveCore{
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(expected, nil)
	response, err := s.resolver.ImageCVE(
		s.ctx, struct {
			Cve                *string
			SubfieldScopeQuery *string
		}{
			Cve: pointers.String("cve-xyz"),
		},
	)
	s.NoError(err)
	s.NotNil(response.data)

	// with filter
	expectedQ = search.NewQueryBuilder().
		AddExactMatches(search.CVE, "cve-xyz").
		AddStrings(search.Fixable, "true").
		ProtoQuery()
	expected = []imagecve.CveCore{
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(expected, nil)
	response, err = s.resolver.ImageCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{
		Cve:                pointers.String("cve-xyz"),
		SubfieldScopeQuery: pointers.String("Fixable:true"),
	},
	)
	s.NoError(err)
	s.NotNil(response.data)

	// with filter
	expectedQ = search.NewQueryBuilder().
		AddExactMatches(search.CVE, "cve-xyz").
		AddStrings(search.Namespace, "n1").
		ProtoQuery()
	expected = []imagecve.CveCore{
		imageCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(expected, nil)
	response, err = s.resolver.ImageCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{
		Cve:                pointers.String("cve-xyz"),
		SubfieldScopeQuery: pointers.String("Namespace:n1"),
	},
	)
	s.NoError(err)
	s.NotNil(response.data)
}
