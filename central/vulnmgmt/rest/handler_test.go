//go:build sql_integration

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/central/views/imagecveflat"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stretchr/testify/suite"
)

func TestVulnMgmtRESTHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

type HandlerTestSuite struct {
	suite.Suite

	testDB          *pgtest.TestPostgres
	handler         http.Handler
	ctx             context.Context
	imageStore      imageV2DS.DataStore
	deploymentStore deploymentDS.DataStore
	watchedStore    watchedImageDS.DataStore
}

func (s *HandlerTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	s.imageStore = imageV2DS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	s.deploymentStore = deploymentStore
	s.watchedStore = watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	s.handler = NewHandler(Deps{
		CVEView:           imagecveflat.NewCVEFlatView(s.testDB.DB),
		CVECountView:      imagecve.NewCVEView(s.testDB.DB),
		CVEDataStore:      imageCVEDS.GetTestPostgresDataStore(s.T(), s.testDB.DB),
		ImageStore:        s.imageStore,
		DeploymentStore:   deploymentStore,
		WatchedImageStore: s.watchedStore,
		DB:                s.testDB.DB,
	})

	s.seedTestData()
}

func imageVuln(cve string, severity storage.VulnerabilitySeverity, cvss float32, summary string) *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:                cve,
		Cvss:               cvss,
		Severity:           severity,
		VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
		Summary:            summary,
		Link:               "https://nvd.nist.gov/vuln/detail/" + cve,
	}
}

func testImageV2(id, registry, remote, tag string, vulns []*storage.EmbeddedVulnerability) *storage.ImageV2 {
	return &storage.ImageV2{
		Id:     id,
		Digest: "sha256:" + id,
		Name: &storage.ImageName{
			Registry: registry,
			Remote:   remote,
			Tag:      tag,
			FullName: registry + "/" + remote + ":" + tag,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{Created: protocompat.TimestampNow()},
		},
		Scan: &storage.ImageScan{
			ScanTime:        protocompat.TimestampNow(),
			OperatingSystem: "rhel:9",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "openssl",
					Version: "1.1.1",
					Vulns:   vulns,
				},
			},
		},
	}
}

func (s *HandlerTestSuite) seedTestData() {
	image1 := testImageV2("image-1", "registry.example.com", "test/image1", "latest", []*storage.EmbeddedVulnerability{
		imageVuln("CVE-2024-0001", storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, 9.8, "Critical vulnerability"),
		imageVuln("CVE-2024-0002", storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, 5.0, "Moderate vulnerability"),
	})
	// CVE-2024-0001 is a known exploit and a known ransomware campaign target.
	image1.GetScan().GetComponents()[0].GetVulns()[0].Exploit = &storage.Exploit{
		KnownRansomwareCampaignUse: "Known",
	}
	image1.SignatureVerificationData = &storage.ImageSignatureVerificationData{
		Results: []*storage.ImageSignatureVerificationResult{
			{
				Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
				VerifierId:              "verifier-id",
				VerifierName:            "verifier-name",
				VerifiedImageReferences: []string{image1.GetName().GetFullName()},
			},
		},
	}

	image2 := testImageV2("image-2", "registry.example.com", "test/image2", "latest", []*storage.EmbeddedVulnerability{
		imageVuln("CVE-2024-0001", storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, 9.8, "Critical vulnerability"),
		imageVuln("CVE-2024-0003", storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, 2.0, "Low vulnerability"),
	})
	image3 := testImageV2("image-3", "registry.example.com", "test/image3", "latest", nil)

	for _, image := range []*storage.ImageV2{image1, image2, image3} {
		s.Require().NoError(s.imageStore.UpsertImage(s.ctx, image))
	}

	deployments := []*storage.Deployment{
		fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, testconsts.NamespaceA, image1),
		fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, testconsts.NamespaceA, image2),
		fixtures.GetDeploymentWithImageV2(testconsts.Cluster2, testconsts.NamespaceB, image3),
	}
	for i, deployment := range deployments {
		deployment.Name = []string{"deployment-1", "deployment-2", "deployment-3"}[i]
		s.Require().NoError(s.deploymentStore.UpsertDeployment(s.ctx, deployment))
	}

	s.Require().NoError(s.watchedStore.UpsertWatchedImage(s.ctx, image1.GetName().GetFullName()))
}

func (s *HandlerTestSuite) get(path string, params url.Values) *httptest.ResponseRecorder {
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

func decodeResponse[T any](s *HandlerTestSuite, rec *httptest.ResponseRecorder) T {
	if rec.Code != http.StatusOK {
		s.T().Logf("unexpected response %d: %s", rec.Code, rec.Body.String())
	}
	s.Require().Equal(http.StatusOK, rec.Code)
	var resp T
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&resp))
	return resp
}

func (s *HandlerTestSuite) TestListCVEs() {
	resp := decodeResponse[CVEListResponse](s, s.get("/api/v2/vulnmgmt/cves", nil))
	s.Len(resp.Items, 3)
	for _, item := range resp.Items {
		s.NotEmpty(item.CVE)
		if item.CVE == "CVE-2024-0001" {
			s.InDelta(9.8, item.TopCVSS, 0.1)
			s.Equal(2, item.AffectedImageCount)
			s.True(item.KnownExploit)
			s.True(item.KnownRansomwareCampaignUse)
		}
		if item.CVE == "CVE-2024-0002" {
			s.False(item.KnownExploit)
			s.False(item.KnownRansomwareCampaignUse)
		}
	}
}

func (s *HandlerTestSuite) TestListCVEsWithSeverityFilter() {
	params := url.Values{}
	params.Set("query", "Severity:CRITICAL_VULNERABILITY_SEVERITY")
	resp := decodeResponse[CVEListResponse](s, s.get("/api/v2/vulnmgmt/cves", params))
	s.Require().Len(resp.Items, 1)
	s.Equal("CVE-2024-0001", resp.Items[0].CVE)
}

func (s *HandlerTestSuite) TestGetCVEDetail() {
	resp := decodeResponse[CVEDetailResponse](s, s.get("/api/v2/vulnmgmt/cves/CVE-2024-0001/detail", nil))
	s.Require().NotEmpty(resp.DistroDetails)
	s.Equal("CVE-2024-0001", resp.DistroDetails[0].CVE)
	s.Equal("Critical vulnerability", resp.DistroDetails[0].Summary)
}

func (s *HandlerTestSuite) TestListImages() {
	resp := decodeResponse[ImageListResponse](s, s.get("/api/v2/vulnmgmt/images", nil))
	s.Len(resp.Items, 3)

	byID := make(map[string]ImageListItem)
	for _, item := range resp.Items {
		byID[item.ID] = item
	}
	s.Equal(2, byID["image-1"].MatchedCVECount)
	s.Equal(1, byID["image-1"].DeploymentCount)
	s.Equal(watchStatusWatched, byID["image-1"].WatchStatus)
	s.Require().NotNil(byID["image-1"].SignatureVerificationData)
	s.Require().Len(byID["image-1"].SignatureVerificationData.Results, 1)
	s.Equal("VERIFIED", byID["image-1"].SignatureVerificationData.Results[0].Status)
	s.Equal("verifier-name", byID["image-1"].SignatureVerificationData.Results[0].VerifierName)
	s.Equal(2, byID["image-2"].MatchedCVECount)
	s.Equal(watchStatusNotWatched, byID["image-2"].WatchStatus)
	s.Equal(0, byID["image-3"].MatchedCVECount)
	s.Equal(1, byID["image-3"].DeploymentCount)
}

func (s *HandlerTestSuite) TestListImagesWithSeverityFilter() {
	params := url.Values{}
	params.Set("query", "Severity:CRITICAL_VULNERABILITY_SEVERITY")
	resp := decodeResponse[ImageListResponse](s, s.get("/api/v2/vulnmgmt/images", params))
	s.Len(resp.Items, 2)
	for _, item := range resp.Items {
		s.Equal(1, item.MatchedCVECount)
	}
}

func (s *HandlerTestSuite) TestListImagesWithoutCVEs() {
	params := url.Values{}
	params.Set("query", "Image CVE Count:0")
	resp := decodeResponse[ImageListResponse](s, s.get("/api/v2/vulnmgmt/images", params))
	s.Require().Len(resp.Items, 1)
	s.Equal("image-3", resp.Items[0].ID)
	s.Equal(0, resp.Items[0].MatchedCVECount)
	s.Equal(1, resp.Items[0].DeploymentCount)
}

func (s *HandlerTestSuite) TestListDeployments() {
	resp := decodeResponse[DeploymentListResponse](s, s.get("/api/v2/vulnmgmt/deployments", nil))
	s.Len(resp.Items, 3)

	byName := make(map[string]DeploymentListItem)
	for _, item := range resp.Items {
		byName[item.Name] = item
	}
	s.Equal(2, byName["deployment-1"].MatchedCVECount)
	s.Equal(1, byName["deployment-1"].ImageCount)
	s.Equal(0, byName["deployment-3"].MatchedCVECount)
	s.Equal(1, byName["deployment-3"].ImageCount)
}

func (s *HandlerTestSuite) TestListDeploymentsWithSeverityFilter() {
	params := url.Values{}
	params.Set("query", "Severity:CRITICAL_VULNERABILITY_SEVERITY")
	resp := decodeResponse[DeploymentListResponse](s, s.get("/api/v2/vulnmgmt/deployments", params))
	s.Len(resp.Items, 2)
	for _, item := range resp.Items {
		s.Equal(1, item.MatchedCVECount)
	}
}

func (s *HandlerTestSuite) TestCounts() {
	resp := decodeResponse[CountsResponse](s, s.get("/api/v2/vulnmgmt/counts", nil))
	s.Equal(3, resp.CVE)
	s.Equal(3, resp.Image)
	s.Equal(3, resp.Deployment)
	s.Equal(SeverityCounts{Critical: 1, Moderate: 1, Low: 1}, resp.Severity)
}

func (s *HandlerTestSuite) TestCountsWithSeverityFilter() {
	params := url.Values{}
	params.Set("query", "Severity:CRITICAL_VULNERABILITY_SEVERITY")
	resp := decodeResponse[CountsResponse](s, s.get("/api/v2/vulnmgmt/counts", params))
	s.Equal(1, resp.CVE)
	s.Equal(2, resp.Image)
	s.Equal(2, resp.Deployment)
}

func (s *HandlerTestSuite) TestMethodNotAllowed() {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/vulnmgmt/cves", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	s.Equal(http.StatusMethodNotAllowed, rec.Code)
}

func (s *HandlerTestSuite) TestInvalidRoute() {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/unknown", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	s.Equal(http.StatusNotFound, rec.Code)
}
