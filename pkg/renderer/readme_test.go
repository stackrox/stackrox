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
		deploymentFormat v1.DeploymentFormat
		mode             mode

		mustContain                           []string
		mustNotContain                        []string
		mustContainInstructionSuffixAndPrefix bool
		expectedErr                           bool
	}{
		{
			orchCommand:      "kubectl",
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             renderAll,

			mustContain:                           []string{"kubectl create -R -f scanner", "kubectl create -R -f central"},
			mustNotContain:                        []string{"kubectl create -R -f monitoring", "helm install"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			orchCommand:      "kubectl",
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             scannerOnly,

			mustContain:    []string{"kubectl create -R -f scanner"},
			mustNotContain: []string{"kubectl create -R -f central", "kubectl create -R -f monitoring", "helm install"},
		},
		{
			orchCommand:      "oc",
			deploymentFormat: v1.DeploymentFormat_KUBECTL,
			mode:             scannerOnly,

			mustContain:    []string{"oc create -R -f scanner"},
			mustNotContain: []string{"oc create -R -f central", "oc create -R -f monitoring", "helm install", "oc create -R -f scannerv2"},
		},
		{
			orchCommand:      "kubectl",
			deploymentFormat: v1.DeploymentFormat_HELM,
			mode:             renderAll,

			mustContain:                           []string{"helm install"},
			mustNotContain:                        []string{"kubectl", "helm install --name", "central/scripts/setup.sh", "scanner/scripts/setup.sh"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			orchCommand:      "kubectl",
			deploymentFormat: v1.DeploymentFormat_HELM,
			mode:             scannerOnly,

			expectedErr: true,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s/%s", c.orchCommand, c.deploymentFormat, c.mode), func(t *testing.T) {
			a := assert.New(t)
			config := Config{
				K8sConfig: &K8sConfig{
					Command:          c.orchCommand,
					DeploymentFormat: c.deploymentFormat,
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
			a.Equal(c.mustContainInstructionSuffixAndPrefix, strings.Contains(out, instructionPrefix(c.deploymentFormat)))
		})
	}
}
