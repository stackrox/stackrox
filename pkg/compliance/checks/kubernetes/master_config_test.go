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
				"test": compliance.CommandLine_builder{
					Process: "kubelet",
					Args:    []*compliance.CommandLine_Args{},
				}.Build(),
			},
			files: map[string]*compliance.File{
				"/etc/cni/net.d": compliance.File_builder{
					Permissions: 0644,
				}.Build(),
				"/opt/cni/bin": compliance.File_builder{
					Permissions: 0644,
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 2,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_10",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "kubelet",
					Args:    []*compliance.CommandLine_Args{},
				}.Build(),
			},
			files: map[string]*compliance.File{
				"/etc/cni/net.d": compliance.File_builder{
					UserName:  "root",
					GroupName: "root",
				}.Build(),
				"/opt/cni/bin": compliance.File_builder{
					UserName:  "root",
					GroupName: "root",
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 2,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_11",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args:    []*compliance.CommandLine_Args{},
				}.Build(),
			},
			files: map[string]*compliance.File{
				"/var/lib/etcddisk": compliance.File_builder{
					Permissions: 0700,
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},

		{
			name: "CIS_Kubernetes_v1_5:1_1_11",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args:    []*compliance.CommandLine_Args{},
				}.Build(),
			},
			files: map[string]*compliance.File{
				"/var/lib/etcddisk": compliance.File_builder{
					UserName:  "etcd",
					GroupName: "etcd",
				}.Build(),
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
				"/etc/kubernetes/pki": compliance.File_builder{
					Children: []*compliance.File{
						compliance.File_builder{
							Path:        "/etc/kubernetes/pki/valid.key",
							Permissions: 0600,
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": compliance.File_builder{
					Children: []*compliance.File{
						compliance.File_builder{
							Path:        "/etc/kubernetes/pki/invalid.key",
							Permissions: 0666,
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": compliance.File_builder{
					Children: []*compliance.File{
						compliance.File_builder{
							Path:        "/etc/kubernetes/pki/valid.key",
							Permissions: 0600,
						}.Build(),
						compliance.File_builder{
							Path:        "/etc/kubernetes/pki/ignored.ignored",
							Permissions: 0666,
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:1_1_21",
			files: map[string]*compliance.File{
				"/etc/kubernetes/pki": compliance.File_builder{
					Children: []*compliance.File{
						compliance.File_builder{
							Path:        "/etc/kubernetes/pki/invalid.key",
							Permissions: 0666,
						}.Build(),
						compliance.File_builder{
							Path:        "/etc/kubernetes/pki/ignored.ignored",
							Permissions: 0666,
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 1,
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
				Files:        c.files,
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, c.numResults)
			for _, checkResult := range checkResults {
				assert.Equal(t, c.status, checkResult.GetState())
			}
		})
	}
}
