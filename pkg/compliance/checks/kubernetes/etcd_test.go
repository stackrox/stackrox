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

func TestETCDChecks(t *testing.T) {
	cases := []struct {
		name         string
		commandLines map[string]*compliance.CommandLine
		status       storage.ComplianceState
		numResults   int
	}{
		{
			name: "CIS_Kubernetes_v1_5:2_1",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "cert-file",
							Values: []string{"test"},
						}.Build(),
						compliance.CommandLine_Args_builder{
							Key:    "key-file",
							Values: []string{"test"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 2,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_1",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key: "cert-file",
						}.Build(),
						compliance.CommandLine_Args_builder{
							Key: "key-file",
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 2,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_2",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "client-cert-auth",
							Values: []string{"true"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_2",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "client-cert-auth",
							Values: []string{"false"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_3",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "auto-tls",
							Values: []string{"false"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_3",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "auto-tls",
							Values: []string{"true"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_4",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "peer-cert-file",
							Values: []string{"test"},
						}.Build(),
						compliance.CommandLine_Args_builder{
							Key:    "peer-key-file",
							Values: []string{"test"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 2,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_4",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key: "peer-cert-file",
						}.Build(),
						compliance.CommandLine_Args_builder{
							Key: "peer-key-file",
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 2,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_5",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "peer-client-cert-auth",
							Values: []string{"true"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_5",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "peer-client-cert-auth",
							Values: []string{"false"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_6",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "peer-auto-tls",
							Values: []string{"false"},
						}.Build(),
					},
				}.Build(),
			},
			status:     storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			numResults: 1,
		},
		{
			name: "CIS_Kubernetes_v1_5:2_6",
			commandLines: map[string]*compliance.CommandLine{
				"test": compliance.CommandLine_builder{
					Process: "etcd",
					Args: []*compliance.CommandLine_Args{
						compliance.CommandLine_Args_builder{
							Key:    "peer-auto-tls",
							Values: []string{"true"},
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
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, c.numResults)
			for _, checkResult := range checkResults {
				assert.Equal(t, c.status, checkResult.GetState())
			}
		})
	}
}
