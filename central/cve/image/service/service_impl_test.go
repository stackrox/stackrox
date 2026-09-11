package service

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	componentMocks "github.com/stackrox/rox/central/imagecomponent/v2/datastore/mocks"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/central/views/imagecve"
	imageCVEViewMocks "github.com/stackrox/rox/central/views/imagecve/mocks"
	vulnReqMocks "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestImageCVEService(t *testing.T) {
	suite.Run(t, new(imageCVEServiceTestSuite))
}

type fakeCveCore struct {
	cve            string
	cveIDs         []string
	severity       common.ResourceCountByCVESeverity
	topCVSS        float32
	topNVDCVSS     float32
	affectedImages int
	firstSeen      *time.Time
	published      *time.Time
}

func (f *fakeCveCore) GetCVE() string      { return f.cve }
func (f *fakeCveCore) GetCVEIDs() []string { return f.cveIDs }
func (f *fakeCveCore) GetImagesBySeverity() common.ResourceCountByCVESeverity {
	return f.severity
}
func (f *fakeCveCore) GetTopCVSS() float32                    { return f.topCVSS }
func (f *fakeCveCore) GetTopNVDCVSS() float32                 { return f.topNVDCVSS }
func (f *fakeCveCore) GetAffectedImageCount() int             { return f.affectedImages }
func (f *fakeCveCore) GetFirstDiscoveredInSystem() *time.Time { return f.firstSeen }
func (f *fakeCveCore) GetPublishDate() *time.Time             { return f.published }

type imageCVEServiceTestSuite struct {
	suite.Suite

	mockCtrl   *gomock.Controller
	cveView    *imageCVEViewMocks.MockCveView
	imageCVEs  *mocks.MockDataStore
	components *componentMocks.MockDataStore
	vulnReqs   *vulnReqMocks.MockDataStore
	svc        Service
	gql        *resolvers.Resolver
	ctx        context.Context
	firstSeen  time.Time
	published  time.Time
	core       *fakeCveCore
}

func (s *imageCVEServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.cveView = imageCVEViewMocks.NewMockCveView(s.mockCtrl)
	s.imageCVEs = mocks.NewMockDataStore(s.mockCtrl)
	s.components = componentMocks.NewMockDataStore(s.mockCtrl)
	s.vulnReqs = vulnReqMocks.NewMockDataStore(s.mockCtrl)
	s.svc = New(s.cveView, s.imageCVEs, s.components, s.vulnReqs)
	s.gql, _ = resolvers.SetupTestResolver(s.T(), s.cveView, s.vulnReqs)
	s.ctx = context.Background()

	s.firstSeen = time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC)
	s.published = time.Date(2021, 11, 26, 0, 0, 0, 0, time.UTC)
	s.core = &fakeCveCore{
		cve:    "CVE-2021-44228",
		cveIDs: []string{"cve-id-1"},
		severity: &common.ResourceCountByImageCVESeverity{
			CriticalSeverityCount:  4,
			ImportantSeverityCount: 1,
			ModerateSeverityCount:  0,
			LowSeverityCount:       0,
			UnknownSeverityCount:   0,
		},
		topCVSS:        9.8,
		topNVDCVSS:     10,
		affectedImages: 4,
		firstSeen:      &s.firstSeen,
		published:      &s.published,
	}
}

func (s *imageCVEServiceTestSuite) expectedListQuery(raw string) *v1.Query {
	q, err := search.ParseQuery(raw, search.MatchAllIfEmpty())
	s.Require().NoError(err)
	paginated.FillPagination(q, nil, maxImageCVEsReturned)
	return q
}

func (s *imageCVEServiceTestSuite) TestCountImageCVEsWithQuery() {
	expectedQ := search.NewQueryBuilder().AddStrings(search.CVE, "CVE-2021-44228").ProtoQuery()
	s.cveView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)

	resp, err := s.svc.CountImageCVEs(s.ctx, &v1.RawQuery{Query: "CVE:CVE-2021-44228"})
	s.Require().NoError(err)
	s.Equal(int32(3), resp.GetCount())
}

func (s *imageCVEServiceTestSuite) TestListImageCVEsEmpty() {
	s.cveView.EXPECT().Get(s.ctx, gomock.Any(), views.ReadOptions{}).Return(nil, nil)

	resp, err := s.svc.ListImageCVEs(s.ctx, &v1.ListImageCVEsRequest{})
	s.Require().NoError(err)
	s.Empty(resp.GetImageCves())
}

func (s *imageCVEServiceTestSuite) TestListImageCVEsQueryAndFieldParity() {
	rawQuery := "CVE:CVE-2021-44228"
	expectedQ := s.expectedListQuery(rawQuery)
	s.cveView.EXPECT().Get(s.ctx, expectedQ, views.ReadOptions{}).Return([]imagecve.CveCore{s.core}, nil)

	vuln := &storage.ImageCVEV2{
		Id:              "cve-id-1",
		ComponentId:     "comp-1",
		Cvss:            9.8,
		Nvdcvss:         10,
		Severity:        storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		NvdScoreVersion: storage.CvssScoreVersion_V3,
		CveBaseInfo: &storage.CVEInfo{
			Cve:          "CVE-2021-44228",
			Summary:      "Log4Shell",
			ScoreVersion: storage.CVEInfo_V3,
			Epss:         &storage.EPSS{EpssProbability: 0.97},
			Exploit:      &storage.Exploit{KnownRansomwareCampaignUse: "Known"},
		},
	}
	s.imageCVEs.EXPECT().SearchRawImageCVEs(s.ctx, gomock.Any()).Return([]*storage.ImageCVEV2{vuln}, nil)
	s.components.EXPECT().GetBatch(s.ctx, gomock.Any()).Return([]*storage.ImageComponentV2{{
		Id:              "comp-1",
		OperatingSystem: "debian:11",
	}}, nil)
	s.vulnReqs.EXPECT().Count(s.ctx, gomock.Any()).Return(2, nil)

	resp, err := s.svc.ListImageCVEs(s.ctx, &v1.ListImageCVEsRequest{
		Query:           rawQuery,
		RequestStatuses: []string{"PENDING"},
	})
	s.Require().NoError(err)
	s.Require().Len(resp.GetImageCves(), 1)

	got := resp.GetImageCves()[0]
	s.Equal("CVE-2021-44228", got.GetCve())
	s.InDelta(9.8, float64(got.GetTopCvss()), 0.001)
	s.InDelta(10, float64(got.GetTopNvdCvss()), 0.001)
	s.Equal(int32(4), got.GetAffectedImageCount())
	s.Equal(int32(4), got.GetAffectedImageCountBySeverity().GetCritical())
	s.Equal(int32(1), got.GetAffectedImageCountBySeverity().GetImportant())
	s.Equal(int32(2), got.GetExceptionCount())
	s.True(protocompat.ConvertTimeToTimestampOrNil(&s.firstSeen).AsTime().Equal(got.GetFirstDiscoveredInSystem().AsTime()))
	s.True(protocompat.ConvertTimeToTimestampOrNil(&s.published).AsTime().Equal(got.GetPublishedOn().AsTime()))

	s.Require().Len(got.GetDistroTuples(), 1)
	tuple := got.GetDistroTuples()[0]
	s.Equal("Log4Shell", tuple.GetSummary())
	s.Equal("debian:11", tuple.GetOperatingSystem())
	s.InDelta(9.8, float64(tuple.GetCvss()), 0.001)
	s.Equal(storage.CVEInfo_V3.String(), tuple.GetScoreVersion())
	s.InDelta(10, float64(tuple.GetNvdCvss()), 0.001)
	s.Equal(storage.CvssScoreVersion_V3.String(), tuple.GetNvdScoreVersion())
	s.InDelta(0.97, float64(tuple.GetEpssProbability()), 0.001)
	s.Equal("Known", tuple.GetKnownRansomwareCampaignUse())
	s.Equal(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, tuple.GetSeverity())
}

func (s *imageCVEServiceTestSuite) TestListAndCountFieldParityVsGraphQL() {
	rawQuery := "CVE:CVE-2021-44228"
	cores := []imagecve.CveCore{s.core}
	s.cveView.EXPECT().Get(gomock.Any(), gomock.Any(), views.ReadOptions{}).Return(cores, nil).AnyTimes()
	s.cveView.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
	s.imageCVEs.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	s.vulnReqs.EXPECT().Count(gomock.Any(), gomock.Any()).Return(2, nil).AnyTimes()

	gqlCtx := resolvers.SetAuthorizerOverride(s.ctx, allow.Anonymous())
	gqlCVEs, err := s.gql.ImageCVEs(gqlCtx, resolvers.PaginatedQuery{Query: &rawQuery})
	s.Require().NoError(err)
	s.Require().Len(gqlCVEs, 1)

	gqlCount, err := s.gql.ImageCVECount(gqlCtx, resolvers.RawQuery{Query: &rawQuery})
	s.Require().NoError(err)

	rest, err := s.svc.ListImageCVEs(s.ctx, &v1.ListImageCVEsRequest{Query: rawQuery})
	s.Require().NoError(err)
	s.Require().Len(rest.GetImageCves(), 1)

	restCount, err := s.svc.CountImageCVEs(s.ctx, &v1.RawQuery{Query: rawQuery})
	s.Require().NoError(err)

	gql := gqlCVEs[0]
	got := rest.GetImageCves()[0]
	s.Equal(gql.CVE(gqlCtx), got.GetCve())
	s.InDelta(gql.TopCVSS(gqlCtx), float64(got.GetTopCvss()), 0.001)
	s.InDelta(gql.TopNVDCVSS(gqlCtx), float64(got.GetTopNvdCvss()), 0.001)
	s.Equal(gql.AffectedImageCount(gqlCtx), got.GetAffectedImageCount())

	gqlSev, err := gql.AffectedImageCountBySeverity(gqlCtx)
	s.Require().NoError(err)
	gqlCritical, err := gqlSev.Critical(gqlCtx)
	s.Require().NoError(err)
	s.Equal(gqlCritical.Total(gqlCtx), got.GetAffectedImageCountBySeverity().GetCritical())

	gqlFirst := gql.FirstDiscoveredInSystem(gqlCtx)
	s.Require().NotNil(gqlFirst)
	s.True(gqlFirst.Time.Equal(got.GetFirstDiscoveredInSystem().AsTime()))
	gqlPublished := gql.PublishedOn(gqlCtx)
	s.Require().NotNil(gqlPublished)
	s.True(gqlPublished.Time.Equal(got.GetPublishedOn().AsTime()))

	gqlException, err := gql.ExceptionCount(gqlCtx, struct{ RequestStatus *[]*string }{})
	s.Require().NoError(err)
	s.Equal(gqlException, got.GetExceptionCount())
	s.Equal(gqlCount, restCount.GetCount())
}
