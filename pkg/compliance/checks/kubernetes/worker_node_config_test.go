package kubernetes

import (
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerNodeConfigChecks(t *testing.T) {
	cases := []struct {
		name         string
		commandLines map[string]*compliance.CommandLine
		files        map[string]*compliance.File
		status       storage.ComplianceState
		numResults   int
	}{
		{
			name: "CIS_Kubernetes_v1_5:4_1_1",
			files: map[string]*compliance.File{
				"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf": {
					Permissions: 0644,
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},

		{
			name: "CIS_Kubernetes_v1_5:4_1_2",
			files: map[string]*compliance.File{
				"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf": {
					UserName:  "root",
					GroupName: "root",
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},

		{
			name: "CIS_Kubernetes_v1_5:4_1_3",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "kubelet",
					Args: []*compliance.CommandLine_Args{
						{
							Key: "kubeconfig",
							File: &compliance.File{
								Permissions: 0644,
							},
						},
					},
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},

		{
			name: "CIS_Kubernetes_v1_5:4_1_4",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "kubelet",
					Args: []*compliance.CommandLine_Args{
						{
							Key: "kubeconfig",
							File: &compliance.File{
								UserName:  "root",
								GroupName: "root",
							},
						},
					},
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
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
				Files:        c.files,
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, c.numResults)
			for _, checkResult := range checkResults {
				assert.Equal(t, c.status, checkResult.State)
			}
		})
	}
}
