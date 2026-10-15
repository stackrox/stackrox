package complianceoperator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		client:                      client,
		stopSignal:                  concurrency.NewSignal(),
		handlerMaxRetries:           2,
		handlerAPICallTimeout:       500 * time.Millisecond,
		handlerRetryTimeout:         2 * time.Second,
		nodeRoleReconcilePassBudget: 2 * time.Second,
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
// turns that error into a fallback, and TestReconcileNodeRoles for the caller that skips on it instead.
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

		assert.Error(t, err, "the caller (defaultNodeRoles or the reconciler) decides what to do about a list failure")
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
		client:                      client,
		complianceOperatorInfo:      fakeStatusInfo{namespace: namespace},
		disabled:                    concurrency.NewSignal(),
		stopSignal:                  concurrency.NewSignal(),
		complianceIsReady:           &ready,
		handlerMaxRetries:           2,
		handlerAPICallTimeout:       500 * time.Millisecond,
		handlerRetryTimeout:         2 * time.Second,
		nodeRoleReconcilePassBudget: 2 * time.Second,
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

	t.Run("does not clobber existing roles when MachineConfigPool listing fails", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, newScanSetting("ss", []string{"infra", masterRole, workerRole}, true))
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("the server could not find the requested resource")
		})
		client.ClearActions()

		m.reconcileNodeRoles()

		// A failed MCP list must never be treated as "the cluster now only has master+worker": that would strip the
		// custom "infra" role from every managed ScanSetting on a transient API hiccup.
		assert.Equal(t, []string{"infra", masterRole, workerRole}, scanSettingRoles(t, client, "ss"))
		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"), "reconciliation must bail out before listing ScanSettings")
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

	t.Run("re-applies the desired roles on a conflicting update instead of losing them", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		objs := append(mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole, workerRole}, true))
		m, client := newReconcileHandler(reconcileNamespace, objs...)

		// The first Update attempt loses a race against some other writer and gets a Conflict; the retry's
		// onConflict callback must Get the (unchanged, in this test) object and reapply desiredRoles on top of it
		// before Update is retried. Returning handled=false on the second call lets the fake client's real
		// ObjectTracker-backed reactor actually perform that second update.
		var updateAttempts int
		client.PrependReactor("update", "scansettings", func(k8stesting.Action) (bool, runtime.Object, error) {
			updateAttempts++
			if updateAttempts == 1 {
				gr := complianceoperator.ScanSetting.GroupVersionResource().GroupResource()
				return true, nil, k8serrors.NewConflict(gr, "ss", errors.New("concurrent modification"))
			}
			return false, nil, nil
		})
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", masterRole, workerRole}, scanSettingRoles(t, client, "ss"),
			"the desired roles must survive the conflict-retry round trip, not just the first, discarded attempt")
		assert.Equal(t, 2, updateAttempts, "expected exactly one retry after the conflict")
	})
}

// TestReconcileNodeRolesBoundsTotalPassTime proves nodeRoleReconcilePassBudget bounds the whole pass, not just one
// ScanSetting's update: each of the 5 ScanSettings here needs a real (slow) update, so without the shared budget the
// pass would take close to 5*updateDelay. The budget is set to fit only two of them, so reconcileNodeRoles must
// return well before that, leaving the rest to the next tick instead of serially waiting out every update in turn.
func TestReconcileNodeRolesBoundsTotalPassTime(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")

	const (
		numScanSettings = 5
		updateDelay     = 150 * time.Millisecond
		passBudget      = 350 * time.Millisecond // fits roughly 2 updates; well under numScanSettings*updateDelay
	)

	objs := mcps("master", "worker", "infra")
	for i := 0; i < numScanSettings; i++ {
		objs = append(objs, newScanSetting(fmt.Sprintf("ss%d", i), []string{masterRole}, true))
	}
	m, client := newReconcileHandler(reconcileNamespace, objs...)
	m.handlerAPICallTimeout = time.Second
	m.handlerRetryTimeout = time.Second
	m.nodeRoleReconcilePassBudget = passBudget
	// Every update genuinely takes updateDelay; returning handled=false lets the real ObjectTracker-backed
	// reactor still perform it afterwards, so this only adds latency, it doesn't change the outcome.
	client.PrependReactor("update", "scansettings", func(k8stesting.Action) (bool, runtime.Object, error) {
		time.Sleep(updateDelay)
		return false, nil, nil
	})
	client.ClearActions()

	start := time.Now()
	m.reconcileNodeRoles()
	elapsed := time.Since(start)

	assert.Less(t, elapsed, numScanSettings*updateDelay, "the pass must not serially wait out every ScanSetting's update")
	updated := countActions(client.Actions(), "update", "scansettings")
	assert.Greater(t, updated, 0, "the budget should still allow at least one update to complete")
	assert.Less(t, updated, numScanSettings, "the budget must cut the pass short before every ScanSetting is reached")
}

// TestNodeRoleReconcilerTicker drives reconciliation through the real Start()/ticker/run() wiring (every other
// reconciler test above calls reconcileNodeRoles directly), to prove that path is actually assembled correctly end
// to end. It sets reconcileInterval directly on the handler, bypassing the env-var clamp enforced by
// NewRequestHandler/clampNodeRoleReconcileInterval, which is the documented way for a test to use a short interval
// without waiting out the production minimum.
func TestNodeRoleReconcilerTicker(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")

	objs := append(mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole, workerRole}, true))
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		machineConfigPoolGVR: "MachineConfigPoolList",
		complianceoperator.ScanSetting.GroupVersionResource(): "ScanSettingList",
	}, objs...)

	ready := concurrency.NewSignal()
	ready.Signal()
	sensorComponent := NewRequestHandler(client, fakeStatusInfo{namespace: reconcileNamespace}, &ready)
	m, ok := sensorComponent.(*handlerImpl)
	require.True(t, ok)
	m.reconcileInterval = 20 * time.Millisecond

	require.NoError(t, sensorComponent.Start())
	defer sensorComponent.Stop()

	require.Eventually(t, func() bool {
		return assert.ObjectsAreEqual([]string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss"))
	}, time.Second, 10*time.Millisecond, "reconciler did not converge the ScanSetting's roles via the real ticker")
}
