package scheduler

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

const (
	expectedVulnReportEmailTemplateRhacsBranding = `
	Red Hat Advanced Cluster Security for Kubernetes has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report for all vulnerabilities for December 31, 1999.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

	expectedVulnReportEmailTemplateStackroxBranding = `
	StackRox has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report for all vulnerabilities for December 31, 1999.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

	expectedNoVulnsFoundEmailTemplateRhacsBranding = `
	Red Hat Advanced Cluster Security for Kubernetes has found zero vulnerabilities associated with the running container images owned by your organization.`

	expectedNoVulnsFoundEmailTemplateStackroxBranding = `
	StackRox has found zero vulnerabilities associated with the running container images owned by your organization.`
)

var _ suite.SetupAllSuite = (*ScheduleTestSuite)(nil)

func TestSchedule(t *testing.T) {
	suite.Run(t, new(ScheduleTestSuite))
}

type ScheduleTestSuite struct {
	suite.Suite

	time time.Time
	rc   *storage.ReportConfiguration
}

func (s *ScheduleTestSuite) SetupSuite() {
	s.rc = fixtures.GetValidReportConfiguration()
	s.time = time.Date(1999, 12, 31, 23, 59, 59, 999, time.Local)
}

func (s *ScheduleTestSuite) TestFormatVulnMessage() {
	tests := map[string]struct {
		vulnReport   string
		noVulnReport string
	}{
		"RHACS_BRANDING": {
			vulnReport:   expectedVulnReportEmailTemplateRhacsBranding,
			noVulnReport: expectedNoVulnsFoundEmailTemplateRhacsBranding,
		},
		"STACKROX_BRANDING": {
			vulnReport:   expectedVulnReportEmailTemplateStackroxBranding,
			noVulnReport: expectedNoVulnsFoundEmailTemplateStackroxBranding,
		},
	}
	for productBranding, tt := range tests {
		s.Run(productBranding, func() {
			s.T().Setenv(branding.ProductBrandingEnvName, productBranding)

			receivedBrandedVulnFound, err := formatMessage(s.rc, vulnReportEmailTemplate, s.time)
			s.NoError(err)
			receivedBrandedNoVulnFound, err := formatMessage(s.rc, noVulnsFoundEmailTemplate, s.time)
			s.NoError(err)

			s.Equal(tt.vulnReport, receivedBrandedVulnFound)
			s.Equal(tt.noVulnReport, receivedBrandedNoVulnFound)
		})
	}
}
