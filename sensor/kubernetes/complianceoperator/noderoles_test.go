package complianceoperator

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const reconcileNamespace = "openshift-compliance"

type fakeStatusInfo struct {
	namespace string
}

func (f fakeStatusInfo) GetNamespace() string { return f.namespace }

// newScanSetting builds an unstructured ScanSetting with the given roles. The stackrox label is what
// getResourcesInCluster filters on, so ScanSettings without it must never be touched by the reconciler.
func newScanSetting(name string, roles []string, withStackroxLabel bool) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(complianceoperator.ScanSetting.GroupVersionKind())
	obj.SetNamespace(reconcileNamespace)
	obj.SetName(name)
	if withStackroxLabel {
		obj.SetLabels(map[string]string{"app.kubernetes.io/name": "stackrox"})
	}
	if roles != nil {
		_ = unstructured.SetNestedStringSlice(obj.Object, roles, "roles")
	}
	return obj
}

// newReconcileHandler wires a handlerImpl with a fake dynamic client seeded with the given objects. Both MCP and
// ScanSetting list kinds are registered since reconcileNodeRoles discovers roles from MCPs and then lists ScanSettings.
func newReconcileHandler(namespace string, objs ...runtime.Object) (*handlerImpl, *fake.FakeDynamicClient) {
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		machineConfigPoolGVR: "MachineConfigPoolList",
		complianceoperator.ScanSetting.GroupVersionResource(): "ScanSettingList",
	}, objs...)
	ready := concurrency.NewSignal()
	ready.Signal()
	m := &handlerImpl{
		client:                 client,
		complianceOperatorInfo: fakeStatusInfo{namespace: namespace},
		disabled:               concurrency.NewSignal(),
		stopSignal:             concurrency.NewSignal(),
		complianceIsReady:      &ready,
		handlerMaxRetries:      2,
		handlerAPICallTimeout:  500 * time.Millisecond,
		handlerRetryTimeout:    2 * time.Second,
	}
	return m, client
}

func scanSettingRoles(t *testing.T, client *fake.FakeDynamicClient, name string) []string {
	obj, err := client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(reconcileNamespace).Get(context.Background(), name, v1.GetOptions{})
	require.NoError(t, err)
	roles, _, err := unstructured.NestedStringSlice(obj.Object, "roles")
	require.NoError(t, err)
	return roles
}

func countActions(actions []k8stesting.Action, verb, resource string) int {
	n := 0
	for _, a := range actions {
		if a.Matches(verb, resource) {
			n++
		}
	}
	return n
}

func mcps(roles ...string) []runtime.Object {
	objs := make([]runtime.Object, 0, len(roles))
	for _, role := range roles {
		objs = append(objs, newMCP(role, map[string]string{nodeRolePrefix + role: ""}))
	}
	return objs
}

func TestReconcileNodeRoles(t *testing.T) {
	t.Run("updates a ScanSetting whose roles differ from discovered", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		objs := append(mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole, workerRole}, true))
		m, client := newReconcileHandler(reconcileNamespace, objs...)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss"))
		assert.Equal(t, 1, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("no-op when roles already equal regardless of order", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		// Discovered set is {master, worker}; the ScanSetting already has those in a different order.
		objs := append(mcps("master", "worker"), newScanSetting("ss", []string{workerRole, masterRole}, true))
		m, client := newReconcileHandler(reconcileNamespace, objs...)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{workerRole, masterRole}, scanSettingRoles(t, client, "ss"), "roles must be left untouched")
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"), "no update should occur when the role set is unchanged")
	})

	t.Run("env var off is a no-op with no list or update", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "false")
		objs := append(mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		m, client := newReconcileHandler(reconcileNamespace, objs...)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{masterRole}, scanSettingRoles(t, client, "ss"))
		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"))
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("brings multiple ScanSettings into line", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		objs := append(mcps("master", "worker", "infra"),
			newScanSetting("ss1", []string{masterRole, workerRole}, true),
			newScanSetting("ss2", []string{masterRole}, true),
		)
		m, client := newReconcileHandler(reconcileNamespace, objs...)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss1"))
		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss2"))
		assert.Equal(t, 2, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("does not touch a ScanSetting without the stackrox label", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		objs := append(mcps("master", "worker", "infra"),
			newScanSetting("managed", []string{masterRole, workerRole}, true),
			newScanSetting("unmanaged", []string{masterRole, workerRole}, false),
		)
		m, client := newReconcileHandler(reconcileNamespace, objs...)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "managed"))
		assert.Equal(t, []string{masterRole, workerRole}, scanSettingRoles(t, client, "unmanaged"), "unmanaged ScanSetting must be left untouched")
		assert.Equal(t, 1, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("no-op while compliance is disabled", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		objs := append(mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		m, client := newReconcileHandler(reconcileNamespace, objs...)
		m.disabled.Signal()
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"))
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("no-op when the compliance operator namespace is unknown", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		objs := append(mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		m, client := newReconcileHandler("", objs...)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"))
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("logs and returns gracefully when ScanSetting listing fails", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker")...)
		client.PrependReactor("list", "scansettings", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("the server could not find the requested resource")
		})
		client.ClearActions()

		assert.NotPanics(t, m.reconcileNodeRoles)
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})
}
