package scheduler

import (
	"bytes"
	"testing"
	"time"

	"text/template"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/defaults"
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
	assert.Nil(t, err)
	var tpl0 bytes.Buffer
	err = tmpl.Execute(&tpl0, data)
	assert.Nil(t, err)

	expectedVulnReportEmailTemplateRhacsBranding := tpl0.String()

	tmpl, err = template.New("VulnsStackrox").Parse(expectedVulnReportEmailTemplateStackroxBrandingWithPlaceholders)
	assert.Nil(t, err)
	var tpl1 bytes.Buffer
	err = tmpl.Execute(&tpl1, data)
	assert.Nil(t, err)

	expectedVulnReportEmailTemplateStackroxBranding := tpl1.String()

	return expectedVulnReportEmailTemplateRhacsBranding, expectedVulnReportEmailTemplateStackroxBranding
}

func TestVulnMessageBranding(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	rc := fixtures.GetValidReportConfiguration()

	expectedVulnReportEmailTemplateRhacsBranding, expectedVulnReportEmailTemplateStackroxBranding := generateExpectedVulnReportEmailTemplates(t)

	// Setting: RHACS release, expected: RHACS branding
	envIsolator.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameRHACSRelease)

	receivedBrandedVulnFound, err := formatMessage(rc)
	assert.Nil(t, err)
	receivedBrandedNoVulnFound, err := formatNoVulnsFoundMessage()
	assert.Nil(t, err)

	assert.Equal(t, expectedVulnReportEmailTemplateRhacsBranding, receivedBrandedVulnFound)
	assert.Equal(t, expectedNoVulnsFoundEmailTemplateRhacsBranding, receivedBrandedNoVulnFound)

	// Setting: Stackrox release, expected: Stackrox branding
	envIsolator.RestoreAll()
	envIsolator.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameStackRoxIORelease)

	receivedBrandedVulnFound, err = formatMessage(rc)
	assert.Nil(t, err)
	receivedBrandedNoVulnFound, err = formatNoVulnsFoundMessage()
	assert.Nil(t, err)

	assert.Equal(t, expectedVulnReportEmailTemplateStackroxBranding, receivedBrandedVulnFound)
	assert.Equal(t, expectedNoVulnsFoundEmailTemplateStackroxBranding, receivedBrandedNoVulnFound)

	// Setting: Development build, expected: RHACS branding
	envIsolator.RestoreAll()
	envIsolator.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameDevelopmentBuild)

	receivedBrandedVulnFound, err = formatMessage(rc)
	assert.Nil(t, err)
	receivedBrandedNoVulnFound, err = formatNoVulnsFoundMessage()
	assert.Nil(t, err)

	assert.Equal(t, expectedVulnReportEmailTemplateRhacsBranding, receivedBrandedVulnFound)
	assert.Equal(t, expectedNoVulnsFoundEmailTemplateRhacsBranding, receivedBrandedNoVulnFound)
}
