package complianceoperator

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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

// newTestHandler builds a handlerImpl with a reset stopSignal and short retry settings.
func newTestHandler(client dynamic.Interface) *handlerImpl {
	return &handlerImpl{
		client:                client,
		stopSignal:            concurrency.NewSignal(),
		handlerMaxRetries:     2,
		handlerAPICallTimeout: 500 * time.Millisecond,
		handlerRetryTimeout:   2 * time.Second,
	}
}

func newHandlerWithMCPs(objs ...runtime.Object) *handlerImpl {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		machineConfigPoolGVR: "MachineConfigPoolList",
	}, objs...)
	return newTestHandler(client)
}

// TestDiscoverNodeRoles exercises the raw lister: it applies no fallback and no env-var gating (that policy lives in
// adaptiveDefaultNodeRoles, see TestAdaptiveDefaultNodeRoles below), so a failed or empty list is reported as-is.
func TestDiscoverNodeRoles(t *testing.T) {
	roleLabel := func(role string) map[string]string {
		return map[string]string{nodeRolePrefix + role: ""}
	}

	t.Run("discovers and sorts roles from MCPs", func(t *testing.T) {
		m := newHandlerWithMCPs(
			newMCP("master", roleLabel("master")),
			newMCP("worker", roleLabel("worker")),
			newMCP("infra", roleLabel("infra")),
		)
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Equal(t, []string{"infra", "master", "worker"}, roles)
	})

	t.Run("returns an error when MCP API is absent", func(t *testing.T) {
		// Simulate a non-OpenShift cluster where listing MCPs fails (API not served).
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		})
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("the server could not find the requested resource")
		})
		m := newTestHandler(client)
		roles, err := m.discoverNodeRoles()
		assert.Error(t, err)
		assert.Empty(t, roles)
	})

	t.Run("returns no roles when no MCPs exist", func(t *testing.T) {
		m := newHandlerWithMCPs()
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Empty(t, roles)
	})

	t.Run("skips pools without a node-role label", func(t *testing.T) {
		m := newHandlerWithMCPs(
			newMCP("master", roleLabel("master")),
			newMCP("custom", map[string]string{"custom-label": "value"}),
		)
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Equal(t, []string{"master"}, roles)
	})

	t.Run("zero-node pool still contributes its role", func(t *testing.T) {
		m := newHandlerWithMCPs(newMCP("infra", roleLabel("infra")))
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Equal(t, []string{"infra"}, roles)
	})
}

// TestAdaptiveDefaultNodeRoles covers the policy built on top of the raw lister (TestDiscoverNodeRoles): the
// ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES gate, and the master+worker fallback on an empty result. A list failure is
// reported as an error rather than silently folded into a fallback — see TestDefaultNodeRoles for the caller that
// turns that error into a fallback.
func TestAdaptiveDefaultNodeRoles(t *testing.T) {
	t.Run("env off returns master+worker without calling discovery", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "false")
		// Seed an infra MCP that would change the result (and show up in Actions) if discovery ran.
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		}, newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""}))
		m := newTestHandler(client)

		roles, err := m.adaptiveDefaultNodeRoles()

		require.NoError(t, err)
		assert.Equal(t, []string{masterRole, workerRole}, roles)
		assert.Empty(t, client.Actions(), "discovery must not run at all while the env var is off")
	})

	t.Run("env on discovers from MCPs", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs(newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""}))

		roles, err := m.adaptiveDefaultNodeRoles()

		require.NoError(t, err)
		assert.Equal(t, []string{"infra"}, roles)
	})

	t.Run("returns an error when the MCP list fails, without applying a fallback itself", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		})
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("the server could not find the requested resource")
		})
		m := newTestHandler(client)

		roles, err := m.adaptiveDefaultNodeRoles()

		assert.Error(t, err, "the caller (defaultNodeRoles) decides what to do about a list failure")
		assert.Empty(t, roles)
	})

	t.Run("falls back to master+worker when no roles are discovered", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs()

		roles, err := m.adaptiveDefaultNodeRoles()

		require.NoError(t, err, "an empty but successful list is not a failure")
		assert.Equal(t, []string{masterRole, workerRole}, roles)
	})
}

// TestDefaultNodeRoles covers the create/update path's policy on top of adaptiveDefaultNodeRoles: it never fails,
// falling back to master+worker when discovery errors. It also pins the exact text of the warning it logs: a
// downstream e2e test matches this same text against live Sensor logs to detect a silent fallback, so a wording
// change here must be deliberate, not an accidental drift.
func TestDefaultNodeRoles(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		machineConfigPoolGVR: "MachineConfigPoolList",
	})
	client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the server could not find the requested resource")
	})
	m := newTestHandler(client)

	core, logs := observer.New(zap.WarnLevel)
	origLog := log
	log = &logging.LoggerImpl{InnerLogger: zap.New(core).Sugar()}
	t.Cleanup(func() { log = origLog })

	assert.Equal(t, []string{masterRole, workerRole}, m.defaultNodeRoles())
	assert.Equal(t, 1, logs.FilterMessageSnippet("Could not auto-discover node roles, falling back to").Len(),
		"a downstream e2e test matches this exact text against live Sensor logs; keep the two in sync")
}

// TestResolveNodeRolesForUpdate covers the update/sync path's policy: on a discovery failure, an existing
// ScanSetting keeps its current roles instead of losing them to the static fallback, but a ScanSetting that doesn't
// exist yet (or has no roles) still gets the fallback.
func TestResolveNodeRolesForUpdate(t *testing.T) {
	newScanSetting := func(roles []string) *unstructured.Unstructured {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		if roles != nil {
			require.NoError(t, unstructured.SetNestedStringSlice(obj.Object, roles, "roles"))
		}
		return obj
	}
	discoverErr := errors.New("the server could not find the requested resource")

	cases := map[string]struct {
		discoveredRoles []string
		discoverErr     error
		existing        *unstructured.Unstructured
		want            []string
	}{
		"discovery succeeded: use the discovered roles regardless of existing state": {
			discoveredRoles: []string{"infra"},
			existing:        newScanSetting([]string{masterRole, workerRole}),
			want:            []string{"infra"},
		},
		"discovery failed, no existing ScanSetting: falls back": {
			discoverErr: discoverErr,
			existing:    nil,
			want:        []string{masterRole, workerRole},
		},
		"discovery failed, existing ScanSetting has no roles: falls back": {
			discoverErr: discoverErr,
			existing:    newScanSetting(nil),
			want:        []string{masterRole, workerRole},
		},
		"discovery failed, existing ScanSetting has roles: preserves them": {
			discoverErr: discoverErr,
			existing:    newScanSetting([]string{"infra"}),
			want:        []string{"infra"},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := resolveNodeRolesForUpdate(c.discoveredRoles, c.discoverErr, c.existing)
			assert.Equal(t, c.want, got)
		})
	}
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
		"multiple node-role labels picks the lexicographically first, deterministically": {
			matchLabels: map[string]string{
				nodeRolePrefix + "worker": "",
				nodeRolePrefix + "infra":  "",
				nodeRolePrefix + "master": "",
			},
			expected: "infra",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getFirstNodeRole(tc.matchLabels))
		})
	}
}
