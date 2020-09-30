package renderer

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestReadme(t *testing.T) {
	cases := []struct {
		orchCommand      string
		deploymentFormat v1.DeploymentFormat
		mode             mode
		newExperience    []bool

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
			newExperience:    []bool{false},

			mustContain:                           []string{"helm install --name central ./central", "helm install --name scanner ./scanner", "scanner/scripts/setup.sh"},
			mustNotContain:                        []string{"kubectl create -R -f central", "kubectl create -R -f monitoring", "kubectl create -R -f scanner", "helm install --name monitoring ./monitoring"},
			mustContainInstructionSuffixAndPrefix: true,
		},
		{
			orchCommand:      "kubectl",
			deploymentFormat: v1.DeploymentFormat_HELM,
			mode:             renderAll,
			newExperience:    []bool{true},

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
		experienceVals := c.newExperience
		if len(experienceVals) == 0 {
			experienceVals = []bool{false, true}
		}
		for _, experienceVal := range experienceVals {
			t.Run(fmt.Sprintf("%s/%s/%s/newExperience=%t", c.orchCommand, c.deploymentFormat, c.mode, experienceVal), func(t *testing.T) {
				env := testutils.NewEnvIsolator(t)
				defer env.RestoreAll()

				if buildinfo.ReleaseBuild && experienceVal != features.CentralInstallationExperience.Enabled() {
					t.Skip("Cannot change feature flag settings on release builds")
				}
				env.Setenv(features.CentralInstallationExperience.EnvVar(), strconv.FormatBool(experienceVal))

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
}
