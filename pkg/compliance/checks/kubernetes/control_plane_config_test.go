package kubernetes

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlPlaneConfigChecks(t *testing.T) {
	cases := []struct {
		name         string
		commandLines map[string]*compliance.CommandLine
		status       storage.ComplianceState
	}{
		{
			name: "CIS_Kubernetes_v1_5:3_2_1",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "kube-apiserver",
					Args: []*compliance.CommandLine_Args{
						{
							Key: "--audit-policy-file",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			t.Parallel()

			standard := standards.NodeChecks[standards.CISKubernetes]
			require.NotNil(t, standard)
			check := standard[c.name]
			require.NotNil(t, check)

			mockNodeData := &standards.ComplianceData{
				CommandLines: c.commandLines,
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}
