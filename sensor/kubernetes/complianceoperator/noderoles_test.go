package complianceoperator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newMCP(name string, matchLabels map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   machineConfigPoolGVR.Group,
		Version: machineConfigPoolGVR.Version,
		Kind:    "MachineConfigPool",
	})
	obj.SetName(name)
	if matchLabels != nil {
		labels := make(map[string]interface{}, len(matchLabels))
		for k, v := range matchLabels {
			labels[k] = v
		}
		_ = unstructured.SetNestedMap(obj.Object, labels, "spec", "nodeSelector", "matchLabels")
	}
	// Zero-node pool: machineCount stays 0. Discovery derives roles from the nodeSelector, not the node count.
	_ = unstructured.SetNestedField(obj.Object, int64(0), "status", "machineCount")
	return obj
}

func newHandlerWithMCPs(objs ...runtime.Object) *handlerImpl {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		machineConfigPoolGVR: "MachineConfigPoolList",
	}, objs...)
	return &handlerImpl{client: client}
}

func TestDiscoverNodeRoles(t *testing.T) {
	roleLabel := func(role string) map[string]string {
		return map[string]string{nodeRolePrefix + role: ""}
	}

	t.Run("env off returns master+worker without listing", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "false")
		// Seed an infra MCP that would change the result if it were listed.
		m := newHandlerWithMCPs(newMCP("infra", roleLabel("infra")))
		assert.Equal(t, []string{masterRole, workerRole}, m.discoverNodeRoles(context.Background()))
	})

	t.Run("discovers and sorts roles from MCPs", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs(
			newMCP("master", roleLabel("master")),
			newMCP("worker", roleLabel("worker")),
			newMCP("infra", roleLabel("infra")),
		)
		assert.Equal(t, []string{"infra", "master", "worker"}, m.discoverNodeRoles(context.Background()))
	})

	t.Run("falls back when MCP API is absent", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		// Simulate a non-OpenShift cluster where listing MCPs fails (API not served).
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		})
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("the server could not find the requested resource")
		})
		m := &handlerImpl{client: client}
		assert.Equal(t, []string{masterRole, workerRole}, m.discoverNodeRoles(context.Background()))
	})

	t.Run("falls back when no MCPs exist", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs()
		assert.Equal(t, []string{masterRole, workerRole}, m.discoverNodeRoles(context.Background()))
	})

	t.Run("skips pools without a node-role label", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs(
			newMCP("master", roleLabel("master")),
			newMCP("custom", map[string]string{"custom-label": "value"}),
		)
		assert.Equal(t, []string{"master"}, m.discoverNodeRoles(context.Background()))
	})

	t.Run("zero-node pool still contributes its role", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs(newMCP("infra", roleLabel("infra")))
		assert.Equal(t, []string{"infra"}, m.discoverNodeRoles(context.Background()))
	})
}

func TestGetFirstNodeRole(t *testing.T) {
	cases := map[string]struct {
		matchLabels map[string]string
		expected    string
	}{
		"master role": {
			matchLabels: map[string]string{nodeRolePrefix + "master": ""},
			expected:    "master",
		},
		"infra role": {
			matchLabels: map[string]string{nodeRolePrefix + "infra": ""},
			expected:    "infra",
		},
		"no node-role label": {
			matchLabels: map[string]string{"custom-label": "value"},
			expected:    "",
		},
		"nil labels": {
			matchLabels: nil,
			expected:    "",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getFirstNodeRole(tc.matchLabels))
		})
	}
}
