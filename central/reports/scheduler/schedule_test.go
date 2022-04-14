package scheduler

import (
	"testing"
	"text/template"
	"time"

	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

var expectedVulnReportEmailTemplateRhacsBrandingWithPlaceholders = `
	Red Hat Advanced Cluster Security for Kubernetes has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

var expectedVulnReportEmailTemplateStackroxBrandingWithPlaceholders = `
	StackRox has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

var expectedNoVulnsFoundEmailTemplateRhacsBranding = `
	Red Hat Advanced Cluster Security for Kubernetes has found zero vulnerabilities associated with the running container images owned by your organization.`

var expectedNoVulnsFoundEmailTemplateStackroxBranding = `
	StackRox has found zero vulnerabilities associated with the running container images owned by your organization.`

type vulnsAndDate struct {
	WhichVulns string
	DateStr    string
}

func generateExpectedVulnReportEmailTemplates(t *testing.T) (string, string) {
	data := &vulnsAndDate{
		WhichVulns: "for all vulnerabilities",
		DateStr:    time.Now().Format("January 02, 2006"),
	}

	tmpl, err := template.New("VulnsRHACS").Parse(expectedVulnReportEmailTemplateRhacsBrandingWithPlaceholders)
	assert.NoError(t, err)
	expectedVulnReportEmailTemplateRhacsBranding, err := templates.ExecuteToString(tmpl, data)
	assert.NoError(t, err)

	tmpl, err = template.New("VulnsStackrox").Parse(expectedVulnReportEmailTemplateStackroxBrandingWithPlaceholders)
	expectedVulnReportEmailTemplateStackroxBranding, err := templates.ExecuteToString(tmpl, data)
	assert.NoError(t, err)

	return expectedVulnReportEmailTemplateRhacsBranding, expectedVulnReportEmailTemplateStackroxBranding
}

func TestFormatVulnMessageBranding(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	rc := fixtures.GetValidReportConfiguration()

	expectedVulnReportEmailTemplateRhacsBranding, expectedVulnReportEmailTemplateStackroxBranding := generateExpectedVulnReportEmailTemplates(t)

	tests := map[string]struct {
		productBranding string
		vulnReport      string
		noVulnReport    string
	}{
		"RHACS branding": {
			productBranding: "RHACS_BRANDING",
			vulnReport:      expectedVulnReportEmailTemplateRhacsBranding,
			noVulnReport:    expectedNoVulnsFoundEmailTemplateRhacsBranding,
		},
		"StackRox branding": {
			productBranding: "STACKROX_BRANDING",
			vulnReport:      expectedVulnReportEmailTemplateStackroxBranding,
			noVulnReport:    expectedNoVulnsFoundEmailTemplateStackroxBranding,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			envIsolator.Setenv(branding.ProductBrandingEnvName, tt.productBranding)

			receivedBrandedVulnFound, err := formatMessage(rc, vulnReportEmailTemplate)
			assert.NoError(t, err)
			receivedBrandedNoVulnFound, err := formatMessage(rc, noVulnsFoundEmailTemplate)
			assert.NoError(t, err)

			assert.Equal(t, tt.vulnReport, receivedBrandedVulnFound)
			assert.Equal(t, tt.noVulnReport, receivedBrandedNoVulnFound)
		})
	}
}
