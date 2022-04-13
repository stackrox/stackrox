package scheduler

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

var expectedVulnReportEmailTemplateRhacsBranding = `
	Red Hat Advanced Cluster Security for Kubernetes has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

var expectedVulnReportEmailTemplateStackroxBranding = `
	StackRox has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

var expectedNoVulnsFoundEmailTemplateRhacsBranding = `
	Red Hat Advanced Cluster Security for Kubernetes has found zero vulnerabilities associated with the running container images owned by your organization.`

var expectedNoVulnsFoundEmailTemplateStackroxBranding = `
	StackRox has found zero vulnerabilities associated with the running container images owned by your organization.`

func TestVulnMessageBranding(t *testing.T) {

	envIsolator := envisolator.NewEnvIsolator(t)
	rc := storage.ReportConfiguration{}

	// Setting: RHACS release, expected: RHACS branding
	envIsolator.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameRHACSRelease)

	receivedBrandedVulnFound, err := formatMessage(&rc)
	assert.Nil(t, err)
	receivedBrandedNoVulnFound, err := formatNoVulnsFoundMessage()
	assert.Nil(t, err)

	assert.Equal(t, expectedVulnReportEmailTemplateRhacsBranding, receivedBrandedVulnFound)
	assert.Equal(t, expectedNoVulnsFoundEmailTemplateRhacsBranding, receivedBrandedNoVulnFound)

	// Setting: Stackrox release, expected: Stackrox branding
	envIsolator.RestoreAll()
	envIsolator.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameStackRoxIORelease)

	receivedBrandedVulnFound, err = formatMessage(&rc)
	assert.Nil(t, err)
	receivedBrandedNoVulnFound, err = formatNoVulnsFoundMessage()
	assert.Nil(t, err)

	assert.Equal(t, expectedVulnReportEmailTemplateStackroxBranding, receivedBrandedVulnFound)
	assert.Equal(t, expectedNoVulnsFoundEmailTemplateStackroxBranding, receivedBrandedNoVulnFound)

	// Setting: Development build, expected: RHACS branding
	envIsolator.RestoreAll()
	envIsolator.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameDevelopmentBuild)

	receivedBrandedVulnFound, err = formatMessage(&rc)
	assert.Nil(t, err)
	receivedBrandedNoVulnFound, err = formatNoVulnsFoundMessage()
	assert.Nil(t, err)

	assert.Equal(t, expectedVulnReportEmailTemplateRhacsBranding, receivedBrandedVulnFound)
	assert.Equal(t, expectedNoVulnsFoundEmailTemplateRhacsBranding, receivedBrandedNoVulnFound)
}
