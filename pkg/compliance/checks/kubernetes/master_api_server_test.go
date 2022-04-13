package kubernetes

import (
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMasterAPIServerChecks(t *testing.T) {
	cases := []struct {
		name         string
		commandLines map[string]*compliance.CommandLine
		status       storage.ComplianceState
	}{
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						{
							Key:    "enable-admission-plugins",
							Values: []string{"PodSecurityPolicy"},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						{
							Key:    "enable-admission-plugins",
							Values: []string{"SecurityContextDeny"},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						{
							Key:    "enable-admission-plugins",
							Values: []string{"Some other value"},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						{
							Key: "enable-admission-plugins",
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
