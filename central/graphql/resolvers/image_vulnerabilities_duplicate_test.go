//go:build sql_integration

package resolvers

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imagesView "github.com/stackrox/rox/central/views/images"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageVulnerabilitiesDuplicates(t *testing.T) {
	suite.Run(t, new(ImageVulnerabilityDuplicateTestSuite))
}

type ImageVulnerabilityDuplicateTestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver
}

func (s *ImageVulnerabilityDuplicateTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())
	vulnReqDatastore, err := TestVulnReqDatastore(s.T(), s.testDB)
	s.Require().NoError(err)

	if features.FlattenImageData.Enabled() {
		s.resolver, _ = SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			vulnReqDatastore,
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	} else {
		s.resolver, _ = SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			CreateTestImageDatastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			vulnReqDatastore,
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	}

	img := &storage.Image{
		Id: "sha-dup-test",
		Name: &storage.ImageName{
			Registry: "reg1",
			Remote:   "img-dup",
			Tag:      "latest",
			FullName: "reg1/img-dup:latest",
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "google.golang.org/grpc",
					Version: "v1.24.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "CVE-2099-DUPTEST",
							Cvss:     9.1,
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "CVE-2099-DUPTEST",
							Cvss:     0,
							Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
						},
					},
				},
				{
					Name:    "stdlib",
					Version: "1.20.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "CVE-2099-NODUP",
							Cvss:     5.0,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
		},
	}

	if features.FlattenImageData.Enabled() {
		s.Require().NoError(s.resolver.ImageV2DataStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(img)))
	} else {
		s.Require().NoError(s.resolver.ImageDataStore.UpsertImage(s.ctx, img))
	}
}

func (s *ImageVulnerabilityDuplicateTestSuite) TestDuplicateCVEsReturnedSeparately() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	query := "CVE:CVE-2099-DUPTEST"
	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.Require().NoError(err)
	s.Require().Equal(2, len(vulns), "Expected 2 entries for duplicate CVE with different severities")

	severities := make(map[string]bool)
	for _, v := range vulns {
		severities[v.Severity(ctx)] = true
	}
	s.Contains(severities, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String())
	s.Contains(severities, storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY.String())
}

func (s *ImageVulnerabilityDuplicateTestSuite) TestNonDuplicateCVEReturnedOnce() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	query := "CVE:CVE-2099-NODUP"
	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.Require().NoError(err)
	s.Equal(1, len(vulns), "Non-duplicate CVE should return exactly one entry")
	s.Equal("CVE-2099-NODUP", vulns[0].CVE(ctx))
}

func (s *ImageVulnerabilityDuplicateTestSuite) TestDuplicateCVEsViaImageScope() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	// Scope to the specific image, simulating the image detail page query path
	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha-dup-test"},
		Level: v1.SearchCategory_IMAGES,
	})

	query := "CVE:CVE-2099-DUPTEST"
	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.Require().NoError(err)
	s.Require().Equal(2, len(vulns), "Image-scoped query should return both duplicate CVE entries")

	cvssSet := make(map[float64]bool)
	for _, v := range vulns {
		cvssSet[v.Cvss(ctx)] = true
	}
	s.True(cvssSet[0], "Expected a CVSS of 0 for the UNKNOWN entry")
	s.Equal(2, len(cvssSet), "Expected 2 distinct CVSS values")
}

func (s *ImageVulnerabilityDuplicateTestSuite) TestDatasourceFieldAvailable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	query := "CVE:CVE-2099-DUPTEST"
	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(vulns), 1)

	// Datasource resolver should not panic — value may be empty for test fixtures
	for _, v := range vulns {
		_ = v.Datasource(ctx)
	}
}
