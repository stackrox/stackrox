package scheduler

import (
	"testing"
	"text/template"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	expectedVulnReportEmailTemplateRhacsBrandingWithPlaceholders = `
	Red Hat Advanced Cluster Security for Kubernetes has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

	expectedVulnReportEmailTemplateStackroxBrandingWithPlaceholders = `
	StackRox has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

	expectedNoVulnsFoundEmailTemplateRhacsBranding = `
	Red Hat Advanced Cluster Security for Kubernetes has found zero vulnerabilities associated with the running container images owned by your organization.`

	expectedNoVulnsFoundEmailTemplateStackroxBranding = `
	StackRox has found zero vulnerabilities associated with the running container images owned by your organization.`

	timeStr = `today`
)

type vulnsAndDate struct {
	WhichVulns string
	DateStr    string
}

type ScheduleTestSuite struct {
	suite.Suite

	rc                                              *storage.ReportConfiguration
	envIsolator                                     *envisolator.EnvIsolator
	expectedVulnReportEmailTemplateRhacsBranding    string
	expectedVulnReportEmailTemplateStackroxBranding string
}

func (s *ScheduleTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.rc = fixtures.GetValidReportConfiguration()

	data := &vulnsAndDate{
		WhichVulns: "for all vulnerabilities",
		DateStr:    timeStr,
	}

	tmpl, err := template.New("VulnsRHACS").Parse(expectedVulnReportEmailTemplateRhacsBrandingWithPlaceholders)
	s.NoError(err)
	s.expectedVulnReportEmailTemplateRhacsBranding, err = templates.ExecuteToString(tmpl, data)
	s.NoError(err)

	tmpl, err = template.New("VulnsStackrox").Parse(expectedVulnReportEmailTemplateStackroxBrandingWithPlaceholders)
	s.NoError(err)
	s.expectedVulnReportEmailTemplateStackroxBranding, err = templates.ExecuteToString(tmpl, data)
	s.NoError(err)
}

func (s *ScheduleTestSuite) TeardownSuite() {
	s.envIsolator.RestoreAll()
}

func (s *ScheduleTestSuite) TestFormatVulnMessage() {
	tests := map[string]struct {
		productBranding string
		vulnReport      string
		noVulnReport    string
	}{
		"RHACS branding": {
			productBranding: "RHACS_BRANDING",
			vulnReport:      s.expectedVulnReportEmailTemplateRhacsBranding,
			noVulnReport:    expectedNoVulnsFoundEmailTemplateRhacsBranding,
		},
		"StackRox branding": {
			productBranding: "STACKROX_BRANDING",
			vulnReport:      s.expectedVulnReportEmailTemplateStackroxBranding,
			noVulnReport:    expectedNoVulnsFoundEmailTemplateStackroxBranding,
		},
	}
	for name, tt := range tests {
		s.T().Run(name, func(t *testing.T) {
			s.envIsolator.Setenv(branding.ProductBrandingEnvName, tt.productBranding)

			receivedBrandedVulnFound, err := formatMessage(rc, vulnReportEmailTemplate, timeStr)
			s.NoError(err)
			receivedBrandedNoVulnFound, err := formatMessage(rc, noVulnsFoundEmailTemplate, timeStr)
			s.NoError(err)

			s.Equal(tt.vulnReport, receivedBrandedVulnFound)
			s.Equal(tt.noVulnReport, receivedBrandedNoVulnFound)
		})
	}
}
