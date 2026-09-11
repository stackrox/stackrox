//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/views/imagecve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageCVEServicePostgres(t *testing.T) {
	suite.Run(t, new(imageCVEServicePostgresSuite))
}

type imageCVEServicePostgresSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *resolvers.Resolver
	svc      Service
}

func (s *imageCVEServicePostgresSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())

	vulnReqStore, err := resolvers.TestVulnReqDatastore(s.T(), s.testDB)
	s.Require().NoError(err)

	cveView := imagecve.NewCVEView(s.testDB.DB)
	componentStore := resolvers.CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl)
	cveStore := resolvers.CreateTestImageCVEV2Datastore(s.T(), s.testDB)

	if features.FlattenImageData.Enabled() {
		imageStore := resolvers.CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
		s.resolver, _ = resolvers.SetupTestResolver(s.T(),
			cveView,
			imageStore,
			componentStore,
			cveStore,
			vulnReqStore,
		)
		s.Require().NoError(imageStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(s.testImage())))
	} else {
		imageStore := resolvers.CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
		s.resolver, _ = resolvers.SetupTestResolver(s.T(),
			cveView,
			imageStore,
			componentStore,
			cveStore,
			vulnReqStore,
		)
		s.Require().NoError(imageStore.UpsertImage(s.ctx, s.testImage()))
	}

	s.svc = New(cveView, cveStore, componentStore, vulnReqStore)

	req := fixtures.GetImageScopeDeferralRequest("reg1", "img1", "tag1", "CVE-2021-44228")
	s.Require().NoError(vulnReqStore.AddRequest(s.ctx, req))
}

func (s *imageCVEServicePostgresSuite) testImage() *storage.Image {
	return &storage.Image{
		Id: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Name: &storage.ImageName{
			Registry: "reg1",
			Remote:   "img1",
			Tag:      "tag1",
			FullName: "reg1/img1:tag1",
		},
		Scan: &storage.ImageScan{
			OperatingSystem: "debian:11",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "log4j",
					Version: "2.14.1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:          "CVE-2021-44228",
							Cvss:         9.8,
							Summary:      "Log4Shell",
							Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							NvdCvss:      10,
							Epss:         &storage.EPSS{EpssProbability: 0.97},
							Exploit:      &storage.Exploit{KnownRansomwareCampaignUse: "Known"},
							PublishedOn:  protocompat.TimestampNow(),
						},
					},
				},
			},
		},
	}
}

func (s *imageCVEServicePostgresSuite) TestListAndCountParityWithGraphQL() {
	query := "CVE:CVE-2021-44228"
	gqlCtx := resolvers.SetAuthorizerOverride(s.ctx, allow.Anonymous())

	gqlCVEs, err := s.resolver.ImageCVEs(gqlCtx, resolvers.PaginatedQuery{Query: &query})
	s.Require().NoError(err)
	s.Require().NotEmpty(gqlCVEs)

	gqlCount, err := s.resolver.ImageCVECount(gqlCtx, resolvers.RawQuery{Query: &query})
	s.Require().NoError(err)

	pending := "PENDING"
	statuses := []*string{&pending}
	gqlException, err := gqlCVEs[0].ExceptionCount(gqlCtx, struct{ RequestStatus *[]*string }{RequestStatus: &statuses})
	s.Require().NoError(err)

	gqlTuples, err := gqlCVEs[0].DistroTuples(gqlCtx)
	s.Require().NoError(err)
	s.Require().NotEmpty(gqlTuples)

	rest, err := s.svc.ListImageCVEs(s.ctx, &v1.ListImageCVEsRequest{
		Query:           query,
		RequestStatuses: []string{"PENDING"},
	})
	s.Require().NoError(err)
	s.Require().Len(rest.GetImageCves(), len(gqlCVEs))

	restCount, err := s.svc.CountImageCVEs(s.ctx, &v1.RawQuery{Query: query})
	s.Require().NoError(err)
	s.Equal(gqlCount, restCount.GetCount())

	got := rest.GetImageCves()[0]
	gql := gqlCVEs[0]
	s.Equal(gql.CVE(gqlCtx), got.GetCve())
	s.InDelta(gql.TopCVSS(gqlCtx), float64(got.GetTopCvss()), 0.001)
	s.InDelta(gql.TopNVDCVSS(gqlCtx), float64(got.GetTopNvdCvss()), 0.001)
	s.Equal(gql.AffectedImageCount(gqlCtx), got.GetAffectedImageCount())
	s.Equal(gqlException, got.GetExceptionCount())

	gqlSev, err := gql.AffectedImageCountBySeverity(gqlCtx)
	s.Require().NoError(err)
	gqlCritical, err := gqlSev.Critical(gqlCtx)
	s.Require().NoError(err)
	s.Equal(gqlCritical.Total(gqlCtx), got.GetAffectedImageCountBySeverity().GetCritical())

	s.Require().Len(got.GetDistroTuples(), len(gqlTuples))
	gqlSummary, err := gqlTuples[0].Summary(gqlCtx)
	s.Require().NoError(err)
	s.Equal(gqlSummary, got.GetDistroTuples()[0].GetSummary())
	s.Equal(gqlTuples[0].OperatingSystem(gqlCtx), got.GetDistroTuples()[0].GetOperatingSystem())
	s.InDelta(gqlTuples[0].Cvss(gqlCtx), float64(got.GetDistroTuples()[0].GetCvss()), 0.001)
	s.InDelta(gqlTuples[0].Nvdcvss(gqlCtx), float64(got.GetDistroTuples()[0].GetNvdCvss()), 0.001)
	s.Equal(gqlTuples[0].ScoreVersion(gqlCtx), got.GetDistroTuples()[0].GetScoreVersion())
	s.Equal(gqlTuples[0].NvdScoreVersion(gqlCtx), got.GetDistroTuples()[0].GetNvdScoreVersion())

	gqlBase, err := gqlTuples[0].CveBaseInfo(gqlCtx)
	s.Require().NoError(err)
	if gqlBase != nil {
		gqlEPSS, err := gqlBase.Epss(gqlCtx)
		s.Require().NoError(err)
		if gqlEPSS != nil {
			s.InDelta(gqlEPSS.EpssProbability(gqlCtx), float64(got.GetDistroTuples()[0].GetEpssProbability()), 0.001)
		}
	}
}
