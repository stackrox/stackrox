package renderer

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestReadme(t *testing.T) {
	cases := []struct {
		orchCommand      string
		monitoringType   MonitoringType
		deploymentFormat v1.DeploymentFormat
		mode             mode
		enableScannerV2  bool

		mustContain                           []string
		mustNotContain                        []string
		mustContainInstructionSuffixAndPrefix bool
		expectedErr                           bool
	}{
		{
			orchCommand:      "kubectl",
			monitoringType:   None,
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             renderAll,

			mustContain:                           []string{"kubectl create -R -f scanner", "kubectl create -R -f central"},
			mustNotContain:                        []string{"kubectl create -R -f monitoring", "helm install"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			orchCommand:      "kubectl",
			monitoringType:   OnPrem,
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             renderAll,

			mustContain:                           []string{"kubectl create -R -f scanner", "kubectl create -R -f central", "kubectl create -R -f monitoring"},
			mustNotContain:                        []string{"helm install"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			orchCommand:      "kubectl",
			monitoringType:   None,
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             scannerOnly,

			mustContain:    []string{"kubectl create -R -f scanner"},
			mustNotContain: []string{"kubectl create -R -f central", "kubectl create -R -f monitoring", "helm install"},
		},
		{
			orchCommand:      "kubectl",
			monitoringType:   OnPrem,
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             scannerOnly,

			mustContain:    []string{"kubectl create -R -f scanner", "scanner/scripts/setup.sh"},
			mustNotContain: []string{"kubectl create -R -f central", "kubectl create -R -f monitoring", "helm install", "kubectl create -R -f scannerv2"},
		},
		{
			orchCommand:      "oc",
			monitoringType:   None,
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             scannerOnly,

			mustContain:    []string{"oc create -R -f scanner"},
			mustNotContain: []string{"oc create -R -f central", "oc create -R -f monitoring", "helm install", "oc create -R -f scannerv2"},
		},
		{
			orchCommand:      "kubectl",
			monitoringType:   OnPrem,
			deploymentFormat: v1.DeploymentFormat_HELM,
			mode:             renderAll,

			mustContain:                           []string{"helm install --name central central", "helm install --name monitoring monitoring", "helm install --name scanner scanner", "scanner/scripts/setup.sh"},
			mustNotContain:                        []string{"kubectl create -R -f central", "kubectl create -R -f monitoring", "kubectl create -R -f scanner"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			orchCommand:      "kubectl",
			monitoringType:   OnPrem,
			deploymentFormat: v1.DeploymentFormat_HELM,
			mode:             scannerOnly,

			expectedErr: true,
		},
		{
			orchCommand:      "kubectl",
			monitoringType:   None,
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             renderAll,
			enableScannerV2:  true,

			mustContain:                           []string{"kubectl create -R -f scannerv2", "scannerv2/scripts/setup.sh", "Scanner V2"},
			mustNotContain:                        []string{"kubectl create -R -f monitoring", "helm install", "scanner/scripts/setup.sh"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			monitoringType:   None,
			deploymentFormat: v1.DeploymentFormat_HELM,
			mode:             renderAll,
			enableScannerV2:  true,

			mustContain:                           []string{"helm install --name scannerv2 scannerv2", "scannerv2/scripts/setup.sh"},
			mustNotContain:                        []string{"kubectl create -R -f scanner", "scanner/scripts/setup.sh"},
			mustContainInstructionSuffixAndPrefix: true,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s/%s/%s/%v", c.orchCommand, c.monitoringType, c.deploymentFormat, c.mode, c.enableScannerV2), func(t *testing.T) {
			a := assert.New(t)
			config := Config{
				K8sConfig: &K8sConfig{
					Command:          c.orchCommand,
					Monitoring:       MonitoringConfig{Type: c.monitoringType},
					DeploymentFormat: c.deploymentFormat,
					ScannerV2Config:  ScannerV2Config{Enable: c.enableScannerV2},
				},
			}
			out, err := generateReadme(&config, c.mode)
			if c.expectedErr {
				a.Error(err)

				// These are assertions on the test data.
				a.Empty(c.mustContain)
				a.Empty(c.mustNotContain)
				return
			}
			a.NoError(err)
			for _, s := range c.mustContain {
				a.Contains(out, s)
			}
			for _, s := range c.mustNotContain {
				a.NotContains(out, s)
			}

			a.Equal(c.mustContainInstructionSuffixAndPrefix, strings.Contains(out, instructionSuffix))
			a.Equal(c.mustContainInstructionSuffixAndPrefix, strings.Contains(out, instructionPrefix()))
		})
	}
}
