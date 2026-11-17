package noderole

import (
	"testing"

	mcpv1 "github.com/openshift/api/machineconfiguration/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mcpWithSpec(selector *mcpv1.MachineConfigPoolSpec) *mcpv1.MachineConfigPool {
	return &mcpv1.MachineConfigPool{
		Spec: *selector,
	}
}

func TestRolesFromMachineConfigPool(t *testing.T) {
	tests := map[string]struct {
		mcp   *mcpv1.MachineConfigPool
		want  []string
	}{
		"nil MachineConfigPool": {
			mcp:  nil,
			want: nil,
		},
		"no selectors": {
			mcp:  mcpWithSpec(&mcpv1.MachineConfigPoolSpec{}),
			want: nil,
		},
		"node selector matchLabels": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/infra": "",
					},
				},
			}),
			want: []string{"infra"},
		},
		"node selector ignores non-role labels": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-type":                     "storage",
						"node-role.kubernetes.io/infra": "",
					},
				},
			}),
			want: []string{"infra"},
		},
		"node selector matchLabels with non-empty value": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
			}),
			want: []string{"master"},
		},
		"node selector matchExpressions with Exists operator": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "node-role.kubernetes.io/worker",
							Operator: metav1.LabelSelectorOpExists,
						},
					},
				},
			}),
			want: []string{"worker"},
		},
		"node selector matchExpressions with In operator": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "node-role.kubernetes.io/worker",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{""},
						},
					},
				},
			}),
			want: []string{"worker"},
		},
		"node selector matchExpressions ignores negative operators": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "node-role.kubernetes.io/worker",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{""},
						},
					},
				},
			}),
			want: nil,
		},
		"machine config selector matchLabels": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				MachineConfigSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"machineconfiguration.openshift.io/role": "worker",
					},
				},
			}),
			want: []string{"worker"},
		},
		"machine config selector ignores other labels": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				MachineConfigSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"machineconfiguration.openshift.io/role": "worker",
						"other-label":                            "value",
					},
				},
			}),
			want: []string{"worker"},
		},
		"machine config selector matchExpressions with In operator": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				MachineConfigSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "machineconfiguration.openshift.io/role",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"worker", "infra"},
						},
					},
				},
			}),
			want: []string{"worker", "infra"},
		},
		"machine config selector matchExpressions ignores other keys and operators": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				MachineConfigSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "machineconfiguration.openshift.io/role",
							Operator: metav1.LabelSelectorOpExists,
						},
						{
							Key:      "other-label",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"worker"},
						},
					},
				},
			}),
			want: nil,
		},
		"roles from both selectors are combined": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/infra": "",
					},
				},
				MachineConfigSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"machineconfiguration.openshift.io/role": "worker",
					},
				},
			}),
			want: []string{"infra", "worker"},
		},
		"default master pool": {
			mcp: mcpWithSpec(&mcpv1.MachineConfigPoolSpec{
				NodeSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
				},
				MachineConfigSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"machineconfiguration.openshift.io/role": "master",
					},
				},
			}),
			want: []string{"master"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			roles := RolesFromMachineConfigPool(tt.mcp)
			assert.ElementsMatch(t, tt.want, roles.AsSlice())
		})
	}
}
