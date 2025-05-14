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

func TestImageCVEV2CoreResolver(t *testing.T) {
	suite.Run(t, new(ImageCVEV2CoreResolverTestSuite))
}

type ImageCVEV2CoreResolverTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	ctx      context.Context

	imageCVEView *imageCVEViewMock.MockCveView

	resolver *Resolver
}

func (s *ImageCVEV2CoreResolverTestSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv(features.FlattenCVEData.EnvVar(), "true")
	}

	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithImagePerm(s.T(), s.mockCtrl)
	s.imageCVEView = imageCVEViewMock.NewMockCveView(s.mockCtrl)
	s.resolver, _ = SetupTestResolver(s.T(), s.imageCVEView)
}

func (s *ImageCVEV2CoreResolverTestSuite) TearDownSuite() {}

func (s *ImageCVEV2CoreResolverTestSuite) TestGetImageCVEsNoImagePerm() {
	q := &PaginatedQuery{}
	response, err := s.resolver.ImageCVEs(context.Background(), *q)
	s.Error(err)
	s.Nil(response)
}

func (s *ImageCVEV2CoreResolverTestSuite) TestGetImageCVEsEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(nil, nil)
	response, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 0)
}

func (s *ImageCVEV2CoreResolverTestSuite) TestGetImageCVEsNonEmpty() {
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

func (s *ImageCVEV2CoreResolverTestSuite) TestGetImageCVEsWithQuery() {
	q := &PaginatedQuery{
		Query: pointers.String("CVE:cve-2022-xyz"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.CVE, "cve-2022-xyz").
		WithPagination(search.NewPagination().Limit(paginated.Unlimited)).ProtoQuery()

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

func (s *ImageCVEV2CoreResolverTestSuite) TestImageCVEsWithPaginatedQuery() {
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
		).Limit(paginated.Unlimited),
	).ProtoQuery()

	s.imageCVEView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return(nil, nil)
	_, err := s.resolver.ImageCVEs(s.ctx, *q)
	s.NoError(err)
}

func (s *ImageCVEV2CoreResolverTestSuite) TestImageCVEsNoImagePerm() {
	response, err := s.resolver.ImageCVEs(context.Background(), PaginatedQuery{})
	s.Error(err)
	s.Nil(response)
}

func (s *ImageCVEV2CoreResolverTestSuite) TestImageCVECountNoImagePerm() {
	response, err := s.resolver.ImageCVECount(context.Background(), RawQuery{})
	s.Error(err)
	s.Zero(response)
}

func (s *ImageCVEV2CoreResolverTestSuite) TestImageCVECount() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.imageCVEView.EXPECT().Count(s.ctx, expectedQ).Return(0, nil)
	response, err := s.resolver.ImageCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(0))
}

func (s *ImageCVEV2CoreResolverTestSuite) TestImageCVECountWithQuery() {
	q := &RawQuery{
		Query: pointers.String("Image:image"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.ImageName, "image").ProtoQuery()

	s.imageCVEView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)
	response, err := s.resolver.ImageCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(3))
}

func (s *ImageCVEV2CoreResolverTestSuite) TestGetImageCVEMalformed() {
	_, err := s.resolver.ImageCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{})
	s.Error(err)
}

func (s *ImageCVEV2CoreResolverTestSuite) TestGetImageCVENonEmpty() {
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
