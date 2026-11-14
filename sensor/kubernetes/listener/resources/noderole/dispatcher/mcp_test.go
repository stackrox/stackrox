package dispatcher

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fakePublisher struct {
	calls int
}

func (f *fakePublisher) Publish(_ pubsub.Event) error {
	f.calls++
	return nil
}

func mcpObject(name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	return u
}

func TestProcessEvent(t *testing.T) {
	tests := map[string]struct {
		setup              func(s *store.MachineConfigPoolStore)
		obj                interface{}
		action             central.ResourceAction
		expectPublishCalls int
		wantRoles          []string
	}{
		"add new MCP publishes": {
			obj:                mcpObject("worker"),
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          []string{"worker"},
		},
		"add duplicate MCP does not publish": {
			setup:              func(s *store.MachineConfigPoolStore) { s.Add("worker") },
			obj:                mcpObject("worker"),
			action:             central.ResourceAction_UPDATE_RESOURCE,
			expectPublishCalls: 0,
			wantRoles:          []string{"worker"},
		},
		"remove existing MCP publishes": {
			setup:              func(s *store.MachineConfigPoolStore) { s.Add("worker") },
			obj:                mcpObject("worker"),
			action:             central.ResourceAction_REMOVE_RESOURCE,
			expectPublishCalls: 1,
			wantRoles:          nil,
		},
		"remove non-existent MCP does not publish": {
			obj:                mcpObject("worker"),
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
			obj:                mcpObject(""),
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
		d.ProcessEvent(mcpObject("worker"), nil, central.ResourceAction_CREATE_RESOURCE)
	})
	assert.ElementsMatch(t, []string{"worker"}, s.GetRoles())
}
