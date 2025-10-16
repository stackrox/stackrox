package kubernetes

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
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
				"test": compliance.CommandLine_builder{
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "enable-admission-plugins",
							Values: []string{"PodSecurityPolicy"},
						}.Build(),
					},
				}.Build(),
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "enable-admission-plugins",
							Values: []string{"SecurityContextDeny"},
						}.Build(),
					},
				}.Build(),
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "enable-admission-plugins",
							Values: []string{"Some other value"},
						}.Build(),
					},
				}.Build(),
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_2_13",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: common.KubeAPIProcessName,
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key: "enable-admission-plugins",
						}.Build(),
					},
				}.Build(),
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
	}

	for _, c := range cases {
		t.Run(strings.ReplaceAll(c.name, ":", "-"), func(t *testing.T) {

			standard := standards.NodeChecks[standards.CISKubernetes]
			require.NotNil(t, standard)
			check := standard[c.name]
			require.NotNil(t, check)

			mockNodeData := &standards.ComplianceData{
				CommandLines: c.commandLines,
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].GetState())
		})
	}
}
