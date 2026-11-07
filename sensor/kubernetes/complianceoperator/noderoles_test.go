package complianceoperator

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
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

// newTestHandler builds a handlerImpl with a reset stopSignal and short retry settings. Its MachineConfigPool
// informer fields (mcpLister, mcpInformerSynced) are left nil, matching an un-Start()-ed handler or one on a
// non-OpenShift cluster.
func newTestHandler(client dynamic.Interface) *handlerImpl {
	return &handlerImpl{
		client:                      client,
		stopSignal:                  concurrency.NewSignal(),
		handlerMaxRetries:           2,
		handlerAPICallTimeout:       500 * time.Millisecond,
		handlerRetryTimeout:         2 * time.Second,
		nodeRoleReconcilePassBudget: 2 * time.Second,
		// Large so the periodic safety-net tick never fires during a short test unless a test overrides it.
		nodeRoleReconcileInterval: time.Hour,
		roleReconcileTrigger:      make(chan struct{}, 1),
	}
}

// newMCPLister builds a synced GenericLister directly from an Indexer, bypassing the real informer/watch machinery
// (startMCPRoleInformer and the event wiring around it are exercised separately), so discoverNodeRoles's
// cache-reading path can be tested deterministically and without waiting on a real sync.
func newMCPLister(mcps ...*unstructured.Unstructured) cache.GenericLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, mcp := range mcps {
		_ = indexer.Add(mcp)
	}
	return cache.NewGenericLister(indexer, machineConfigPoolGVR.GroupResource())
}

// newHandlerWithMCPs builds a handlerImpl whose MachineConfigPool cache is pre-populated and already synced.
func newHandlerWithMCPs(mcps ...*unstructured.Unstructured) *handlerImpl {
	m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
	m.mcpLister = newMCPLister(mcps...)
	m.mcpInformerSynced = func() bool { return true }
	return m
}

// explodingLister panics on any use; it proves a code path never consults the informer's cache at all (as opposed
// to consulting it and merely getting an empty/fallback result).
type explodingLister struct{}

func (explodingLister) List(_ labels.Selector) ([]runtime.Object, error) {
	panic("the MachineConfigPool informer must not be consulted here")
}
func (explodingLister) Get(_ string) (runtime.Object, error) {
	panic("the MachineConfigPool informer must not be consulted here")
}
func (explodingLister) ByNamespace(_ string) cache.GenericNamespaceLister {
	panic("the MachineConfigPool informer must not be consulted here")
}

// TestDiscoverNodeRoles exercises the raw cache read: it applies no fallback and no env-var gating (that policy
// lives in discoverDefaultNodeRoles, see TestDiscoverDefaultNodeRoles below), so an unavailable/unsynced informer
// or an empty cache is reported as-is.
func TestDiscoverNodeRoles(t *testing.T) {
	roleLabel := func(role string) map[string]string {
		return map[string]string{nodeRolePrefix + role: ""}
	}

	t.Run("discovers and sorts roles from the cache", func(t *testing.T) {
		m := newHandlerWithMCPs(
			newMCP("master", roleLabel("master")),
			newMCP("worker", roleLabel("worker")),
			newMCP("infra", roleLabel("infra")),
		)
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Equal(t, []string{"infra", "master", "worker"}, roles)
	})

	t.Run("returns an error when the informer was never started", func(t *testing.T) {
		// Simulates a non-OpenShift cluster, or discovery disabled: startMCPRoleInformer never ran, so mcpLister
		// is still nil.
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		roles, err := m.discoverNodeRoles()
		assert.Error(t, err)
		assert.Empty(t, roles)
	})

	t.Run("returns an error when the cache has not synced yet", func(t *testing.T) {
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		m.mcpLister = newMCPLister(newMCP("infra", roleLabel("infra")))
		m.mcpInformerSynced = func() bool { return false }
		// Make the bounded wait return immediately instead of riding out mcpCacheSyncCallTimeout: the stop signal
		// short-circuits waitForMCPCacheSync's derived context regardless of the timeout.
		m.stopSignal.Signal()

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

// TestDiscoverDefaultNodeRoles covers the classification built on top of the raw cache read (TestDiscoverNodeRoles
// above): the ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES gate, and the four outcomes (disabled, discovered, empty,
// unavailable). The interpreters rolesForCreate / rolesForUpdate and the reconciler then apply their own policy to
// these outcomes, see the tests below.
func TestDiscoverDefaultNodeRoles(t *testing.T) {
	t.Run("env off is reported as disabled without consulting the informer", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "false")
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		// A lister that would panic if touched, and that would answer "infra" (changing the result) if it were.
		m.mcpLister = explodingLister{}
		m.mcpInformerSynced = func() bool { return true }

		disc := m.discoverDefaultNodeRoles()

		assert.Equal(t, nodeRolesDisabled, disc.outcome)
		assert.Empty(t, disc.roles)
	})

	t.Run("env on with roles in the cache is reported as discovered", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs(newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""}))

		disc := m.discoverDefaultNodeRoles()

		assert.Equal(t, nodeRolesDiscovered, disc.outcome)
		assert.Equal(t, []string{"infra"}, disc.roles)
	})

	t.Run("env on with an unavailable informer is reported as unavailable, with the error", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))

		disc := m.discoverDefaultNodeRoles()

		assert.Equal(t, nodeRolesUnavailable, disc.outcome)
		assert.Error(t, disc.err)
		assert.Empty(t, disc.roles)
	})

	t.Run("env on with a synced but empty cache is reported as empty, not discovered", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m := newHandlerWithMCPs()

		disc := m.discoverDefaultNodeRoles()

		assert.Equal(t, nodeRolesEmpty, disc.outcome, "a synced-but-empty cache must be distinct from discovered, so callers can preserve existing roles")
		assert.Empty(t, disc.roles)
	})
}

// TestRolesForCreate covers the create path's policy: discovered roles win, every non-authoritative outcome falls back
// to master+worker, logging a warning on the fallback.
func TestRolesForCreate(t *testing.T) {
	cases := map[string]struct {
		disc       roleDiscoveryResult
		want       []string
		wantWarnFB bool // expect the "Could not auto-discover node roles, falling back to" warning
	}{
		"discovered: use the discovered roles": {
			disc: roleDiscoveryResult{roles: []string{"infra"}, outcome: nodeRolesDiscovered},
			want: []string{"infra"},
		},
		"disabled: fallback, no warning (flag off is intentional)": {
			disc: roleDiscoveryResult{outcome: nodeRolesDisabled},
			want: []string{masterRole, workerRole},
		},
		"empty: fallback with warning": {
			disc:       roleDiscoveryResult{outcome: nodeRolesEmpty},
			want:       []string{masterRole, workerRole},
			wantWarnFB: true,
		},
		"unavailable: fallback with warning": {
			disc:       roleDiscoveryResult{outcome: nodeRolesUnavailable, err: errors.New("informer not available")},
			want:       []string{masterRole, workerRole},
			wantWarnFB: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			core, logs := observer.New(zap.WarnLevel)
			origLog := log
			log = &logging.LoggerImpl{InnerLogger: zap.New(core).Sugar()}
			t.Cleanup(func() { log = origLog })

			assert.Equal(t, c.want, c.disc.rolesForCreate())

			warns := logs.FilterMessageSnippet("Could not auto-discover node roles, falling back to").Len()
			if c.wantWarnFB {
				assert.Equal(t, 1, warns, "a non-authoritative outcome must log exactly one fallback warning")
			} else {
				assert.Zero(t, warns)
			}
		})
	}
}

// TestRolesForUpdate covers the update/sync path's policy. Discovered roles win; the flag being off overwrites with the
// static fallback (legacy behavior). When discovery is not authoritative (empty OR unavailable) an existing ScanSetting
// keeps its current roles, but one that doesn't exist yet (or has no roles) still gets the fallback. The empty-but-has
// -existing case is the regression guard: a synced cache that momentarily reports no roles must not strip custom roles.
func TestRolesForUpdate(t *testing.T) {
	newScanSettingObj := func(roles []string) *unstructured.Unstructured {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		if roles != nil {
			require.NoError(t, unstructured.SetNestedStringSlice(obj.Object, roles, "roles"))
		}
		return obj
	}
	unavailable := roleDiscoveryResult{outcome: nodeRolesUnavailable, err: errors.New("MachineConfigPool informer is not available")}
	empty := roleDiscoveryResult{outcome: nodeRolesEmpty}

	cases := map[string]struct {
		disc     roleDiscoveryResult
		existing *unstructured.Unstructured
		want     []string
	}{
		"discovered: use the discovered roles regardless of existing state": {
			disc:     roleDiscoveryResult{roles: []string{"infra"}, outcome: nodeRolesDiscovered},
			existing: newScanSettingObj([]string{masterRole, workerRole}),
			want:     []string{"infra"},
		},
		"disabled: overwrite existing roles with the fallback (legacy behavior)": {
			disc:     roleDiscoveryResult{outcome: nodeRolesDisabled},
			existing: newScanSettingObj([]string{"infra"}),
			want:     []string{masterRole, workerRole},
		},
		"unavailable, no existing ScanSetting: falls back": {
			disc:     unavailable,
			existing: nil,
			want:     []string{masterRole, workerRole},
		},
		"unavailable, existing ScanSetting has no roles: falls back": {
			disc:     unavailable,
			existing: newScanSettingObj(nil),
			want:     []string{masterRole, workerRole},
		},
		"unavailable, existing ScanSetting has roles: preserves them": {
			disc:     unavailable,
			existing: newScanSettingObj([]string{"infra"}),
			want:     []string{"infra"},
		},
		"empty, existing ScanSetting has roles: preserves them (regression guard)": {
			disc:     empty,
			existing: newScanSettingObj([]string{"infra"}),
			want:     []string{"infra"},
		},
		"empty, no existing ScanSetting: falls back": {
			disc:     empty,
			existing: nil,
			want:     []string{masterRole, workerRole},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := c.disc.rolesForUpdate(c.existing)
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

// newReconcileHandler wires a handlerImpl with a synced MachineConfigPool cache (mcps) and a fake dynamic client
// seeded with ScanSettings: reconcileNodeRoles derives desired roles from the former and reconciles the latter.
func newReconcileHandler(namespace string, mcps []*unstructured.Unstructured, scanSettings ...runtime.Object) (*handlerImpl, *fake.FakeDynamicClient) {
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		complianceoperator.ScanSetting.GroupVersionResource(): "ScanSettingList",
	}, scanSettings...)
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
		// Large so the periodic safety-net tick never fires during a short test unless a test overrides it.
		nodeRoleReconcileInterval: time.Hour,
		roleReconcileTrigger:      make(chan struct{}, 1),
	}
	m.mcpLister = newMCPLister(mcps...)
	m.mcpInformerSynced = func() bool { return true }
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

func mcps(roles ...string) []*unstructured.Unstructured {
	objs := make([]*unstructured.Unstructured, 0, len(roles))
	for _, role := range roles {
		objs = append(objs, newMCP(role, map[string]string{nodeRolePrefix + role: ""}))
	}
	return objs
}

func TestReconcileNodeRoles(t *testing.T) {
	t.Run("updates a ScanSetting whose roles differ from discovered", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole, workerRole}, true))
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss"))
		assert.Equal(t, 1, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("no-op when roles already equal regardless of order", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		// Discovered set is {master, worker}; the ScanSetting already has those in a different order.
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker"), newScanSetting("ss", []string{workerRole, masterRole}, true))
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{workerRole, masterRole}, scanSettingRoles(t, client, "ss"), "roles must be left untouched")
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"), "no update should occur when the role set is unchanged")
	})

	t.Run("env var off is a no-op with no list or update", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "false")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{masterRole}, scanSettingRoles(t, client, "ss"))
		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"))
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("brings multiple ScanSettings into line", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"),
			newScanSetting("ss1", []string{masterRole, workerRole}, true),
			newScanSetting("ss2", []string{masterRole}, true),
		)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss1"))
		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "ss2"))
		assert.Equal(t, 2, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("does not touch a ScanSetting without the stackrox label", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"),
			newScanSetting("managed", []string{masterRole, workerRole}, true),
			newScanSetting("unmanaged", []string{masterRole, workerRole}, false),
		)
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", "master", "worker"}, scanSettingRoles(t, client, "managed"))
		assert.Equal(t, []string{masterRole, workerRole}, scanSettingRoles(t, client, "unmanaged"), "unmanaged ScanSetting must be left untouched")
		assert.Equal(t, 1, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("no-op while compliance is disabled", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		m.disabled.Signal()
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"))
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("no-op when the compliance operator namespace is unknown", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler("", mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"))
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("does not clobber existing roles when the MachineConfigPool informer is unavailable", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, nil, newScanSetting("ss", []string{"infra", masterRole, workerRole}, true))
		// Simulate the informer never having started (e.g. non-OpenShift, or it failed to start).
		m.mcpLister = nil
		m.mcpInformerSynced = nil
		client.ClearActions()

		m.reconcileNodeRoles()

		// An unavailable informer must never be treated as "the cluster now only has master+worker": that would
		// strip the custom "infra" role from every managed ScanSetting on a transient failure.
		assert.Equal(t, []string{"infra", masterRole, workerRole}, scanSettingRoles(t, client, "ss"))
		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"), "reconciliation must bail out before listing ScanSettings")
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("does not clobber existing roles when discovery finds zero roles on a synced cache", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		// Synced cache (mcpInformerSynced true via newReconcileHandler) but no MachineConfigPools, so discovery is
		// nodeRolesEmpty, not nodeRolesDiscovered. This is the regression guard: an empty-but-synced result must not be
		// mistaken for "the cluster now has only master+worker" and strip the custom "infra" role.
		m, client := newReconcileHandler(reconcileNamespace, nil, newScanSetting("ss", []string{"infra", masterRole, workerRole}, true))
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Equal(t, []string{"infra", masterRole, workerRole}, scanSettingRoles(t, client, "ss"))
		assert.Zero(t, countActions(client.Actions(), "list", "scansettings"), "reconciliation must bail out before listing ScanSettings")
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("logs and returns gracefully when ScanSetting listing fails", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker"))
		client.PrependReactor("list", "scansettings", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("the server could not find the requested resource")
		})
		client.ClearActions()

		assert.NotPanics(t, m.reconcileNodeRoles)
		assert.Zero(t, countActions(client.Actions(), "update", "scansettings"))
	})

	t.Run("re-applies the desired roles on a conflicting update instead of losing them", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole, workerRole}, true))

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
// return well before that, leaving the rest to the next trigger instead of serially waiting out every update in turn.
func TestReconcileNodeRolesBoundsTotalPassTime(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")

	const (
		numScanSettings = 5
		updateDelay     = 150 * time.Millisecond
		passBudget      = 350 * time.Millisecond // fits roughly 2 updates; well under numScanSettings*updateDelay
	)

	var scanSettings []runtime.Object
	for i := range numScanSettings {
		scanSettings = append(scanSettings, newScanSetting(fmt.Sprintf("ss%d", i), []string{masterRole}, true))
	}
	m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), scanSettings...)
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

// TestReconcileNodeRolesBoundsSlowList proves nodeRoleReconcilePassBudget also bounds the initial ScanSetting list,
// not just the per-ScanSetting updates (see TestReconcileNodeRolesBoundsTotalPassTime above): the list is made to
// fail repeatedly with a much longer handlerRetryTimeout than the pass budget, so if the list were still bounded
// only by handlerRetryTimeout, it would run far longer and use far more of the retry budget than asserted here.
func TestReconcileNodeRolesBoundsSlowList(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")

	const (
		attemptDelay = 100 * time.Millisecond
		passBudget   = 150 * time.Millisecond // enough for only one List attempt plus a bit of backoff
	)

	m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker"), newScanSetting("ss", []string{masterRole}, true))
	m.handlerAPICallTimeout = time.Second
	m.handlerRetryTimeout = 5 * time.Second // much longer than passBudget: proves the budget, not this, bounds the list
	m.handlerMaxRetries = 10
	m.nodeRoleReconcilePassBudget = passBudget

	var listAttempts int
	client.PrependReactor("list", "scansettings", func(k8stesting.Action) (bool, runtime.Object, error) {
		listAttempts++
		time.Sleep(attemptDelay)
		return true, nil, errors.New("server is busy")
	})
	client.ClearActions()

	start := time.Now()
	assert.NotPanics(t, m.reconcileNodeRoles)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, m.handlerRetryTimeout, "the pass budget, not handlerRetryTimeout, must bound a slow/retried ScanSetting list")
	assert.Less(t, listAttempts, m.handlerMaxRetries, "the pass budget should cut retries short before handlerMaxRetries is reached")
	assert.Zero(t, countActions(client.Actions(), "update", "scansettings"), "no update should be attempted when the list itself never succeeded")
}

// TestReconcileNodeRolesBudgetRequeue covers the re-queue on a budget-exhausted pass: a pass that is cut short before
// every ScanSetting is processed must re-queue itself (one pending trigger) so the run() loop resumes the rest, while a
// pass that finishes within budget must not re-queue.
func TestReconcileNodeRolesBudgetRequeue(t *testing.T) {
	t.Run("re-queues when the pass budget is exhausted before every ScanSetting is processed", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		const (
			numScanSettings = 5
			updateDelay     = 150 * time.Millisecond
			passBudget      = 250 * time.Millisecond // fits ~2 updates, well under numScanSettings*updateDelay
		)
		var scanSettings []runtime.Object
		for i := range numScanSettings {
			scanSettings = append(scanSettings, newScanSetting(fmt.Sprintf("ss%d", i), []string{masterRole}, true))
		}
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), scanSettings...)
		m.handlerAPICallTimeout = time.Second
		m.handlerRetryTimeout = time.Second
		m.nodeRoleReconcilePassBudget = passBudget
		client.PrependReactor("update", "scansettings", func(k8stesting.Action) (bool, runtime.Object, error) {
			time.Sleep(updateDelay)
			return false, nil, nil
		})
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Len(t, m.roleReconcileTrigger, 1, "a budget-exhausted pass must re-queue itself to finish the remaining ScanSettings")
	})

	t.Run("does not re-queue when the pass finishes within budget", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
		client.ClearActions()

		m.reconcileNodeRoles()

		assert.Empty(t, m.roleReconcileTrigger, "a pass that completes within budget must not re-queue")
	})
}

// TestReconcileOnceReady covers the startup-race one-shot: it requests a single reconcile once compliance becomes
// ready, and returns without requesting one if the handler stops first.
func TestReconcileOnceReady(t *testing.T) {
	t.Run("requests a reconcile when compliance becomes ready", func(t *testing.T) {
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		ready := concurrency.NewSignal()
		m.complianceIsReady = &ready

		done := make(chan struct{})
		go func() { defer close(done); m.reconcileOnceReady() }()

		assert.Empty(t, m.roleReconcileTrigger, "nothing should be queued before compliance is ready")
		ready.Signal()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("reconcileOnceReady did not return after compliance became ready")
		}
		assert.Len(t, m.roleReconcileTrigger, 1, "becoming ready must request exactly one reconcile")
	})

	t.Run("returns without requesting a reconcile when the handler stops first", func(t *testing.T) {
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		ready := concurrency.NewSignal()
		m.complianceIsReady = &ready
		m.stopSignal.Signal()

		m.reconcileOnceReady()

		assert.Empty(t, m.roleReconcileTrigger, "a stopped handler must not request a reconcile")
	})
}

// TestEnableComplianceRequestsReconcile covers the re-enable path: MachineConfigPool events that arrive while
// compliance is disabled are dropped by reconcileNodeRoles' gate, so re-enabling must request one reconcile to
// catch up, mirroring reconcileOnceReady for the not-ready-at-startup gate.
func TestEnableComplianceRequestsReconcile(t *testing.T) {
	m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
	m.disabled = concurrency.NewSignal()
	m.disabled.Signal()
	// Buffered so enableCompliance's response send doesn't block without a run() loop draining it.
	m.response = make(chan *message.ExpiringMessage, 1)

	m.enableCompliance(&central.EnableComplianceRequest{Id: "test"})

	assert.False(t, m.disabled.IsDone(), "enable must clear the disabled signal")
	assert.Len(t, m.roleReconcileTrigger, 1, "re-enabling compliance must request exactly one reconcile")
}

// TestRunPeriodicTickReconciles proves the safety-net tick in run() actually drives a reconcile: with a short interval
// and no MachineConfigPool event or Central request, a drifted ScanSetting is still brought in line.
func TestRunPeriodicTickReconciles(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
	m, client := newReconcileHandler(reconcileNamespace, mcps("master", "worker", "infra"), newScanSetting("ss", []string{masterRole}, true))
	m.nodeRoleReconcileInterval = 50 * time.Millisecond
	defer m.stopSignal.Signal()

	go m.run()

	assert.Eventually(t, func() bool {
		obj, err := client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
			Namespace(reconcileNamespace).Get(context.Background(), "ss", v1.GetOptions{})
		if err != nil {
			return false
		}
		roles, _, err := unstructured.NestedStringSlice(obj.Object, "roles")
		return err == nil && assert.ObjectsAreEqual([]string{"infra", masterRole, workerRole}, roles)
	}, 2*time.Second, 20*time.Millisecond, "the periodic tick should reconcile the drifted ScanSetting with no other trigger")
}

// TestRequestReconcileCoalescesAndNeverBlocks covers requestReconcile's non-blocking send: repeated calls with
// nothing draining the channel must never block the caller (the informer's event-processing goroutine), and must
// coalesce into a single pending reconcile.
func TestRequestReconcileCoalescesAndNeverBlocks(t *testing.T) {
	m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 10 {
			m.requestReconcile()
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("requestReconcile blocked despite the capacity-1 trigger channel")
	}

	assert.Len(t, m.roleReconcileTrigger, 1, "repeated triggers must coalesce into a single pending reconcile")
}

// TestOnMCPUpdateOnlyTriggersOnRoleChange covers the chattiness guard on the informer's UpdateFunc: an MCP update
// that doesn't change the derived role (e.g. status churn) must not request a reconcile, but one that does must.
func TestOnMCPUpdateOnlyTriggersOnRoleChange(t *testing.T) {
	roleLabel := func(role string) map[string]string { return map[string]string{nodeRolePrefix + role: ""} }

	t.Run("unrelated field change does not trigger", func(t *testing.T) {
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		oldObj := newMCP("infra", roleLabel("infra"))
		newObj := oldObj.DeepCopy()
		require.NoError(t, unstructured.SetNestedField(newObj.Object, int64(3), "status", "machineCount"))

		m.onMCPUpdate(oldObj, newObj)

		assert.Empty(t, m.roleReconcileTrigger)
	})

	t.Run("role change triggers a reconcile", func(t *testing.T) {
		m := newTestHandler(fake.NewSimpleDynamicClient(runtime.NewScheme()))
		oldObj := newMCP("infra", roleLabel("infra"))
		newObj := newMCP("infra", roleLabel("other"))

		m.onMCPUpdate(oldObj, newObj)

		assert.Len(t, m.roleReconcileTrigger, 1)
	})
}

// TestStartMCPRoleInformerGating covers the OpenShift gate in startMCPRoleInformer: the informer must only start
// when MachineConfigPools are actually listable.
func TestStartMCPRoleInformerGating(t *testing.T) {
	t.Run("ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES off: no probe, no informer", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "false")
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		}, newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""}))
		m := newTestHandler(client)

		m.startMCPRoleInformer()

		assert.Nil(t, m.mcpLister)
		assert.Empty(t, client.Actions(), "the env var being off must skip the MachineConfigPool probe entirely")
	})

	t.Run("non-OpenShift cluster: MCP CRD absent, informer never starts", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		})
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, k8serrors.NewNotFound(machineConfigPoolGVR.GroupResource(), "")
		})
		m := newTestHandler(client)

		m.startMCPRoleInformer()

		assert.Nil(t, m.mcpLister, "the feature must stay a no-op when MachineConfigPools aren't listable")
		roles, err := m.discoverNodeRoles()
		assert.Error(t, err)
		assert.Empty(t, roles)
	})

	t.Run("transient list failure is retried until the probe recovers, then the informer starts", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		}, newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""}))
		// Fail the first few probes with a transient (non-NotFound) error, then recover. A NotFound would instead
		// disable discovery permanently (non-OpenShift), which is covered above.
		var listCalls atomic.Int32
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			if listCalls.Add(1) <= 5 {
				return true, nil, errors.New("connection refused")
			}
			return false, nil, nil // let the real tracker serve the probe and the informer's list/watch
		})
		m := newTestHandler(client)
		m.handlerMaxRetries = 1
		m.handlerRetryTimeout = 50 * time.Millisecond
		m.nodeRoleReconcileInterval = 50 * time.Millisecond // fast re-probe between exhausted fast-retries
		defer m.stopSignal.Signal()

		done := make(chan struct{})
		go func() { defer close(done); m.startMCPRoleInformer() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("startMCPRoleInformer did not finish after the probe recovered")
		}

		require.NotNil(t, m.getMCPLister(), "informer must start once the probe recovers, not stay disabled")
		require.True(t, m.waitForMCPCacheSync(time.Second))
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Equal(t, []string{"infra"}, roles)
	})

	t.Run("persistent non-NotFound failure keeps retrying until stop, never permanently disabling discovery", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		})
		var listCalls atomic.Int32
		client.PrependReactor("list", "machineconfigpools", func(k8stesting.Action) (bool, runtime.Object, error) {
			listCalls.Add(1)
			return true, nil, errors.New("connection refused")
		})
		m := newTestHandler(client)
		m.handlerMaxRetries = 1
		m.handlerRetryTimeout = 20 * time.Millisecond
		m.nodeRoleReconcileInterval = 20 * time.Millisecond

		done := make(chan struct{})
		go func() { defer close(done); m.startMCPRoleInformer() }()

		// It must keep re-probing instead of giving up after the first exhausted retry, while discovery stays
		// unavailable and create/update safely fall back.
		require.Eventually(t, func() bool { return listCalls.Load() >= 4 }, 2*time.Second, 10*time.Millisecond,
			"startMCPRoleInformer must keep re-probing MachineConfigPools, not disable discovery after one failure")
		assert.Nil(t, m.getMCPLister(), "informer must not start while the probe keeps failing")
		assert.Equal(t, []string{masterRole, workerRole}, m.discoverDefaultNodeRoles().rolesForCreate())

		// ...and it must exit promptly once the handler stops.
		m.stopSignal.Signal()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("startMCPRoleInformer did not return after stopSignal")
		}
	})

	t.Run("OpenShift cluster: MCPs listable, informer starts and syncs", func(t *testing.T) {
		t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")
		client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			machineConfigPoolGVR: "MachineConfigPoolList",
		}, newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""}))
		m := newTestHandler(client)
		defer m.stopSignal.Signal()

		m.startMCPRoleInformer()

		require.NotNil(t, m.mcpLister)
		require.True(t, m.waitForMCPCacheSync(time.Second))
		roles, err := m.discoverNodeRoles()
		require.NoError(t, err)
		assert.Equal(t, []string{"infra"}, roles)
	})
}

// TestMCPInformerTriggersReconcileOnAddAndDelete drives the real informer wiring end to end (every other informer
// test above either skips it or substitutes a synced lister) to prove MachineConfigPool Add and Delete events
// actually request a reconcile, unlike a periodic poll which would need no such wiring.
func TestMCPInformerTriggersReconcileOnAddAndDelete(t *testing.T) {
	t.Setenv(env.ComplianceAutodiscoverNodeRoles.EnvVar(), "true")

	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		machineConfigPoolGVR: "MachineConfigPoolList",
	})
	m := newTestHandler(client)
	defer m.stopSignal.Signal()

	m.startMCPRoleInformer()
	require.NotNil(t, m.mcpLister, "informer should have started: MachineConfigPools are listable")
	require.True(t, m.waitForMCPCacheSync(time.Second))

	drainTrigger := func() bool {
		select {
		case <-m.roleReconcileTrigger:
			return true
		default:
			return false
		}
	}

	mcp := newMCP("infra", map[string]string{nodeRolePrefix + "infra": ""})
	_, err := client.Resource(machineConfigPoolGVR).Create(context.Background(), mcp, v1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, drainTrigger, time.Second, 10*time.Millisecond, "AddFunc should have requested a reconcile")

	err = client.Resource(machineConfigPoolGVR).Delete(context.Background(), "infra", v1.DeleteOptions{})
	require.NoError(t, err)
	require.Eventually(t, drainTrigger, time.Second, 10*time.Millisecond, "DeleteFunc should have requested a reconcile")
}
