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

func TestMasterConfigChecks(t *testing.T) {
	cases := []struct {
		name         string
		commandLines map[string]*compliance.CommandLine
		files        map[string]*compliance.File
		status       storage.ComplianceState
		numResults   int
	}{
		{
			name: "CIS_Kubernetes_v1_5:1_1_9",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "kubelet",
					Args:    []*compliance.CommandLine_Args{},
				},
			},
			files: map[string]*compliance.File{
				"/etc/cni/net.d": {
					Permissions: 0644,
				},
				"/opt/cni/bin": {
					Permissions: 0644,
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 2,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_10",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "kubelet",
					Args:    []*compliance.CommandLine_Args{},
				},
			},
			files: map[string]*compliance.File{
				"/etc/cni/net.d": {
					UserName:  "root",
					GroupName: "root",
				},
				"/opt/cni/bin": {
					UserName:  "root",
					GroupName: "root",
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 2,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_11",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "etcd",
					Args:    []*compliance.CommandLine_Args{},
				},
			},
			files: map[string]*compliance.File{
				"/var/lib/etcddisk": {
					Permissions: 0700,
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_11",
			commandLines: map[string]*compliance.CommandLine{
				"test": {
					Process: "etcd",
					Args:    []*compliance.CommandLine_Args{},
				},
			},
			files: map[string]*compliance.File{
				"/var/lib/etcddisk": {
					UserName:  "etcd",
					GroupName: "etcd",
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": {},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": {
					Children: []*compliance.File{
						{
							Path:        "/etc/kubernetes/pki/valid.key",
							Permissions: 0600,
						},
					},
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": {
					Children: []*compliance.File{
						{
							Path:        "/etc/kubernetes/pki/invalid.key",
							Permissions: 0666,
						},
					},
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": {
					Children: []*compliance.File{
						{
							Path:        "/etc/kubernetes/pki/valid.key",
							Permissions: 0600,
						},
						{
							Path:        "/etc/kubernetes/pki/ignored.ignored",
							Permissions: 0666,
						},
					},
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": {
					Children: []*compliance.File{
						{
							Path:        "/etc/kubernetes/pki/invalid.key",
							Permissions: 0666,
						},
						{
							Path:        "/etc/kubernetes/pki/ignored.ignored",
							Permissions: 0666,
						},
					},
				},
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
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
