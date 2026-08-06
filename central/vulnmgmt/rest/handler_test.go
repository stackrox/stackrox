//go:build sql_integration

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageCVEStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	imageV2Store "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	"github.com/stackrox/rox/central/views/imagecveflat"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestVulnMgmtRESTHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

type HandlerTestSuite struct {
	suite.Suite

	testDB  *pgtest.TestPostgres
	handler http.Handler
	ctx     context.Context
}

func (s *HandlerTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnMgmtRESTAPI.EnvVar(), "true")
	s.T().Setenv(features.FlattenImageData.EnvVar(), "true")

	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	cveView := imagecveflat.NewCVEFlatView(s.testDB.DB)
	cveDS := imageCVEDS.New(imageCVEStore.New(s.testDB.DB))
	s.handler = NewHandler(cveView, cveDS, nil)

	s.seedTestData()
}

func (s *HandlerTestSuite) seedTestData() {
	store := imageV2Store.New(s.testDB.DB, false, concurrency.NewKeyFence())
	image := &storage.ImageV2{
		Id:     uuid.NewV4().String(),
		Digest: "sha256:test123",
		Name: &storage.ImageName{
			Registry: "registry.example.com",
			Remote:   "test/image",
			Tag:      "latest",
			FullName: "registry.example.com/test/image:latest",
		},
		Scan: &storage.ImageScan{
			ScanTime:        protocompat.TimestampNow(),
			OperatingSystem: "rhel:9",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "openssl",
					Version: "1.1.1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                "CVE-2024-0001",
							Cvss:               9.8,
							Severity:           storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							Summary:            "Critical vulnerability in OpenSSL",
							Link:               "https://nvd.nist.gov/vuln/detail/CVE-2024-0001",
						},
						{
							Cve:                "CVE-2024-0002",
							Cvss:               5.0,
							Severity:           storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							Summary:            "Moderate vulnerability in OpenSSL",
							Link:               "https://nvd.nist.gov/vuln/detail/CVE-2024-0002",
						},
					},
				},
			},
		},
	}
	s.Require().NoError(store.Upsert(s.ctx, image))
}

func (s *HandlerTestSuite) TestListCVEs() {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/cves", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp CVEListResponse
	s.NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.Equal(2, resp.TotalCount)
	s.Len(resp.Items, 2)

	cveNames := make(map[string]bool)
	for _, item := range resp.Items {
		cveNames[item.CVE] = true
		s.NotEmpty(item.TopSeverity)
	}
	s.True(cveNames["CVE-2024-0001"])
	s.True(cveNames["CVE-2024-0002"])
}

func (s *HandlerTestSuite) TestListCVEsFieldValues() {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/cves", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	var resp CVEListResponse
	s.NoError(json.NewDecoder(rec.Body).Decode(&resp))

	for _, item := range resp.Items {
		if item.CVE == "CVE-2024-0001" {
			s.Equal("CRITICAL_VULNERABILITY_SEVERITY", item.TopSeverity)
			s.InDelta(9.8, item.TopCVSS, 0.1)
			s.Equal(1, item.AffectedImageCount)
		}
		if item.CVE == "CVE-2024-0002" {
			s.Equal("MODERATE_VULNERABILITY_SEVERITY", item.TopSeverity)
			s.InDelta(5.0, item.TopCVSS, 0.1)
		}
	}
}

func (s *HandlerTestSuite) TestGetCVEDetail() {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/cves/CVE-2024-0001/detail", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp CVEDetailResponse
	s.NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.NotEmpty(resp.DistroDetails)
	s.Equal("CVE-2024-0001", resp.DistroDetails[0].CVE)
	s.Contains(resp.DistroDetails[0].Summary, "OpenSSL")
}

func (s *HandlerTestSuite) TestGetCVEDetailNotFound() {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/cves/CVE-9999-9999/detail", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp CVEDetailResponse
	s.NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.Empty(resp.DistroDetails)
}

func (s *HandlerTestSuite) TestFlagDisabled() {
	s.T().Setenv(features.VulnMgmtRESTAPI.EnvVar(), "false")
	if features.VulnMgmtRESTAPI.Enabled() {
		s.T().Skip("Cannot disable VulnMgmtRESTAPI")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/cves", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *HandlerTestSuite) TestListCVEsWithSortBySeverity() {
	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/vulnmgmt/cves?pagination.limit=5&pagination.sortOption.field=Severity&pagination.sortOption.reversed=true",
		nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp CVEListResponse
	s.NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.NotEmpty(resp.Items)
	s.Equal(2, resp.TotalCount)
	s.Equal("CRITICAL_VULNERABILITY_SEVERITY", resp.Items[0].TopSeverity)
}

func (s *HandlerTestSuite) TestListCVEsWithSortByCVSS() {
	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/vulnmgmt/cves?pagination.limit=5&pagination.sortOption.field=CVSS&pagination.sortOption.reversed=true",
		nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp CVEListResponse
	s.NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.NotEmpty(resp.Items)
	s.InDelta(9.8, resp.Items[0].TopCVSS, 0.1)
}

func (s *HandlerTestSuite) TestMethodNotAllowed() {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/vulnmgmt/cves", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusMethodNotAllowed, rec.Code)
}

func (s *HandlerTestSuite) TestInvalidRoute() {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/vulnmgmt/cves/invalid/path/here", nil).WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}
