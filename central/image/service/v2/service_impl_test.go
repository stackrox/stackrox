package service

import (
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

type stubCveExport struct {
	cve             string
	severity        storage.VulnerabilitySeverity
	cvss            float32
	nvdCvss         float32
	summary         string
	link            string
	publishedOn     *time.Time
	epssProbability float32
	epssPercentile  float32
	advisoryName    string
	advisoryLink    string
	cveIDs          []string
}

func (s *stubCveExport) GetCVE() string                             { return s.cve }
func (s *stubCveExport) GetSeverity() storage.VulnerabilitySeverity { return s.severity }
func (s *stubCveExport) GetCVSS() float32                           { return s.cvss }
func (s *stubCveExport) GetNVDCVSS() float32                        { return s.nvdCvss }
func (s *stubCveExport) GetSummary() string                         { return s.summary }
func (s *stubCveExport) GetLink() string                            { return s.link }
func (s *stubCveExport) GetPublishedOn() *time.Time                 { return s.publishedOn }
func (s *stubCveExport) GetEPSSProbability() float32                { return s.epssProbability }
func (s *stubCveExport) GetEPSSPercentile() float32                 { return s.epssPercentile }
func (s *stubCveExport) GetAdvisoryName() string                    { return s.advisoryName }
func (s *stubCveExport) GetAdvisoryLink() string                    { return s.advisoryLink }
func (s *stubCveExport) GetCVEIDs() []string                        { return s.cveIDs }

func TestCveExportToDetail(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	input := &stubCveExport{
		cve:             "CVE-2024-1234",
		severity:        storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		cvss:            9.1,
		summary:         "A critical vulnerability.",
		link:            "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
		publishedOn:     &now,
		epssProbability: 0.85,
		epssPercentile:  0.95,
		advisoryName:    "RHSA-2024:1234",
		advisoryLink:    "https://access.redhat.com/errata/RHSA-2024:1234",
	}

	detail := cveExportToDetail(input)

	assert.Equal(t, "CVE-2024-1234", detail.GetCve())
	assert.Equal(t, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, detail.GetSeverity())
	assert.InDelta(t, 9.1, detail.GetCvss(), 0.01)
	assert.Equal(t, "A critical vulnerability.", detail.GetSummary())
	assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2024-1234", detail.GetLink())
	assert.InDelta(t, 0.85, detail.GetEpssProbability(), 0.01)
	assert.InDelta(t, 0.95, detail.GetEpssPercentile(), 0.01)
	assert.Equal(t, "RHSA-2024:1234", detail.GetAdvisory().GetName())
	assert.NotNil(t, detail.GetPublishedOn())
}

func TestCveExportToDetail_NoAdvisory(t *testing.T) {
	input := &stubCveExport{
		cve:      "CVE-2024-5678",
		severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		cvss:     3.0,
	}

	detail := cveExportToDetail(input)

	assert.Equal(t, "CVE-2024-5678", detail.GetCve())
	assert.Nil(t, detail.GetAdvisory())
	assert.Nil(t, detail.GetPublishedOn())
}

type stubFinding struct {
	deploymentID     string
	imageID          string
	cve              string
	componentName    string
	componentVersion string
	isFixable        bool
	fixedBy          string
	state            storage.VulnerabilityState
	severity         storage.VulnerabilitySeverity
	cvss             float32
	repositoryCPE    string
}

func (s *stubFinding) GetDeploymentID() string                    { return s.deploymentID }
func (s *stubFinding) GetImageID() string                         { return s.imageID }
func (s *stubFinding) GetCVE() string                             { return s.cve }
func (s *stubFinding) GetComponentName() string                   { return s.componentName }
func (s *stubFinding) GetComponentVersion() string                { return s.componentVersion }
func (s *stubFinding) GetIsFixable() bool                         { return s.isFixable }
func (s *stubFinding) GetFixedBy() string                         { return s.fixedBy }
func (s *stubFinding) GetState() storage.VulnerabilityState       { return s.state }
func (s *stubFinding) GetSeverity() storage.VulnerabilitySeverity { return s.severity }
func (s *stubFinding) GetCVSS() float32                           { return s.cvss }
func (s *stubFinding) GetRepositoryCPE() string                   { return s.repositoryCPE }

func TestFindingToProto(t *testing.T) {
	input := &stubFinding{
		deploymentID:     "deploy-123",
		imageID:          "sha256:abc",
		cve:              "CVE-2024-9999",
		componentName:    "openssl",
		componentVersion: "1.1.1k",
		isFixable:        true,
		fixedBy:          "1.1.1l",
		state:            storage.VulnerabilityState_OBSERVED,
		severity:         storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		cvss:             7.5,
		repositoryCPE:    "cpe:2.3:o:redhat:enterprise_linux:8:*:*:*:*:*:*:*",
	}

	result := findingToProto(input)

	assert.Equal(t, "deploy-123", result.GetDeploymentId())
	assert.Equal(t, "sha256:abc", result.GetImageId())
	assert.Equal(t, "CVE-2024-9999", result.GetCve())
	assert.Equal(t, "openssl", result.GetComponentName())
	assert.Equal(t, "1.1.1k", result.GetComponentVersion())
	assert.True(t, result.GetIsFixable())
	assert.Equal(t, "1.1.1l", result.GetFixedBy())
	assert.Equal(t, v2.VulnerabilityState_OBSERVED, result.GetState())
	assert.Equal(t, v2.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, result.GetSeverity())
	assert.InDelta(t, 7.5, result.GetCvss(), 0.01)
	assert.Equal(t, "cpe:2.3:o:redhat:enterprise_linux:8:*:*:*:*:*:*:*", result.GetRepositoryCpe())
}

func TestConvertSeverity(t *testing.T) {
	tests := map[string]struct {
		input    storage.VulnerabilitySeverity
		expected v2.VulnerabilitySeverity
	}{
		"low":       {storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY},
		"moderate":  {storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY},
		"important": {storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY},
		"critical":  {storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		"unknown":   {storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY, v2.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertSeverity(tc.input))
		})
	}
}
