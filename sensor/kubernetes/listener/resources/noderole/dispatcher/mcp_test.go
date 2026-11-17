package dispatcher

import (
	"testing"

	mcpv1 "github.com/openshift/api/machineconfiguration/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type fakePublisher struct {
	calls int
}

func (f *fakePublisher) Publish(_ pubsub.Event) error {
	f.calls++
	return nil
}

func mcpObject(t *testing.T, name string, spec *mcpv1.MachineConfigPoolSpec) *unstructured.Unstructured {
	t.Helper()
	mcp := &mcpv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	if spec != nil {
		mcp.Spec = *spec
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(mcp)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: u}
}

func nodeSelectorSpec(matchLabels map[string]string) *mcpv1.MachineConfigPoolSpec {
	return &mcpv1.MachineConfigPoolSpec{
		NodeSelector: &metav1.LabelSelector{MatchLabels: matchLabels},
	}
}

func TestProcessEvent(t *testing.T) {
	tests := map[string]struct {
		setup              func(s *store.MachineConfigPoolStore)
		obj                interface{}
		action             central.ResourceAction
		expectPublishCalls int
		wantRoles          []string
	}{
		"add new MCP with node selector publishes": {
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/infra": ""})),
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          []string{"infra"},
		},
		"add MCP named after one role but selecting another": {
			obj:                mcpObject(t, "custom-pool", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/custom-role": ""})),
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          []string{"custom-role"},
		},
		"add MCP with machine config selector matchExpressions": {
			obj: mcpObject(t, "worker", &mcpv1.MachineConfigPoolSpec{
				MachineConfigSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "machineconfiguration.openshift.io/role",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"worker"},
						},
					},
				},
			}),
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          []string{"worker"},
		},
		"add duplicate MCP does not publish": {
			setup:              func(s *store.MachineConfigPoolStore) { s.Upsert("infra", set.NewStringSet("infra")) },
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/infra": ""})),
			action:             central.ResourceAction_UPDATE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          []string{"infra"},
		},
		"update MCP with changed roles publishes": {
			setup:              func(s *store.MachineConfigPoolStore) { s.Upsert("infra", set.NewStringSet("infra")) },
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/gpu": ""})),
			action:             central.ResourceAction_UPDATE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          []string{"gpu"},
		},
		"update MCP with role already tracked by another pool does not publish": {
			setup: func(s *store.MachineConfigPoolStore) {
				s.Upsert("worker", set.NewStringSet("worker", "infra"))
			},
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/infra": ""})),
			action:             central.ResourceAction_UPDATE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          []string{"worker", "infra"},
		},
		"add MCP without role selectors does not publish": {
			obj:                mcpObject(t, "worker", &mcpv1.MachineConfigPoolSpec{}),
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          nil,
		},
		"remove existing MCP publishes": {
			setup:              func(s *store.MachineConfigPoolStore) { s.Upsert("infra", set.NewStringSet("infra")) },
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/infra": ""})),
			action:             central.ResourceAction_REMOVE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          nil,
		},
		"remove MCP sharing roles with another pool does not change roles": {
			setup: func(s *store.MachineConfigPoolStore) {
				s.Upsert("worker", set.NewStringSet("worker"))
				s.Upsert("infra", set.NewStringSet("worker"))
			},
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/worker": ""})),
			action:             central.ResourceAction_REMOVE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          []string{"worker"},
		},
		"remove non-existent MCP does not publish": {
			obj:                mcpObject(t, "infra", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/infra": ""})),
			action:             central.ResourceAction_REMOVE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          nil,
		},
		"invalid object type returns nil": {
			obj:                "not-an-unstructured",
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          nil,
		},
		"object with empty name returns nil": {
			obj:                mcpObject(t, "", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/worker": ""})),
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := store.NewMachineConfigPoolStore()
			if tt.setup != nil {
				tt.setup(s)
			}
			pub := &fakePublisher{}
			d := NewMachineConfigPoolDispatcher(s, pub)

			result := d.ProcessEvent(tt.obj, nil, tt.action)

			assert.Nil(t, result)
			assert.Equal(t, tt.expectPublishCalls, pub.calls, "expected publish")
			assert.ElementsMatch(t, tt.wantRoles, s.GetRoles())
		})
	}
}

func TestProcessEventNilPublisher(t *testing.T) {
	s := store.NewMachineConfigPoolStore()
	d := NewMachineConfigPoolDispatcher(s, nil)

	assert.NotPanics(t, func() {
		d.ProcessEvent(mcpObject(t, "worker", nodeSelectorSpec(map[string]string{"node-role.kubernetes.io/worker": ""})), nil, central.ResourceAction_CREATE_RESOURCE)
	})
	assert.ElementsMatch(t, []string{"worker"}, s.GetRoles())
}
