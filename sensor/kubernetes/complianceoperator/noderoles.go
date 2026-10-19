package complianceoperator

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// nodeRolePrefix is the node-role label prefix Compliance Operator uses (pkg/utils.nodeRolePrefix) to map a
// ScanSetting role to the nodes it targets. We can't check that constant directly (unexported, and importing the
// package that holds it pulls in a large unrelated dependency tree), so TestComplianceV2AutodiscoverNodeRoles
// verifies this two ways instead: against real Node labels, and against Compliance Operator's own node-targeting
// behavior (does a scan for a discovered role actually match real Nodes).
const nodeRolePrefix = "node-role.kubernetes.io/"

const (
	// mcpInformerStartupSyncTimeout bounds how long Start() waits for the MachineConfigPool informer's initial
	// List+Watch to populate its cache, so a slow or unreachable API server can never block Sensor startup
	// indefinitely. The informer keeps running and syncing in the background after this returns; a timeout here
	// only means discoverNodeRoles falls back to master+worker until the cache catches up, and a later
	// MachineConfigPool event reconciles it.
	mcpInformerStartupSyncTimeout = 10 * time.Second

	// mcpCacheSyncCallTimeout bounds the sync check discoverNodeRoles performs on every create/update, so a cache
	// that is still catching up (e.g. right after Start()) can never block the single-writer handler loop for
	// more than this long.
	mcpCacheSyncCallTimeout = 2 * time.Second

	// defaultNodeRoleReconcilePassBudget bounds how long a single reconcileNodeRoles pass may run across all
	// ScanSettings. Reconciliation runs on the same single-writer loop that processes Central requests, so an
	// unbounded pass (one retry-timeout-bounded update per ScanSetting) would delay request processing by
	// multiples of handlerRetryTimeout as the number of managed ScanSettings grows. Kept below
	// componentProcessTimeout (sensor/common/sensor/component_queue.go, 30s, not exported) so a request already
	// queued when a reconcile pass starts still gets handled before its own ProcessMessage deadline expires.
	defaultNodeRoleReconcilePassBudget = 20 * time.Second
)

// machineConfigPoolGVR is the GVR for MachineConfigPools. These only exist on OpenShift.
var machineConfigPoolGVR = schema.GroupVersionResource{
	Group:    "machineconfiguration.openshift.io",
	Version:  "v1",
	Resource: "machineconfigpools",
}

// getFirstNodeRole extracts the node role from an MCP's spec.nodeSelector.matchLabels: the first key with the
// node-role prefix, with the prefix trimmed. Returns "" if none. Like Compliance Operator's own GetFirstNodeRole,
// but deterministic: CO picks an arbitrary key from a map iteration when a pool carries multiple node-role labels
// (an edge case CO's own source flags as unhandled), whereas this sorts first so the same key wins every time.
func getFirstNodeRole(matchLabels map[string]string) string {
	var keys []string
	for k := range matchLabels {
		if strings.HasPrefix(k, nodeRolePrefix) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	slices.Sort(keys)
	return strings.TrimPrefix(keys[0], nodeRolePrefix)
}

// mcpRole returns the node role derived from a MachineConfigPool informer object, or "" if obj isn't an
// *unstructured.Unstructured or carries no node-role label.
func mcpRole(obj interface{}) string {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	matchLabels, _, _ := unstructured.NestedStringMap(u.Object, "spec", "nodeSelector", "matchLabels")
	return getFirstNodeRole(matchLabels)
}

// fallbackDefaultNodeRoles are the static node roles used when ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is off, or as
// a safe default when discovery is enabled but fails or finds nothing.
func fallbackDefaultNodeRoles() []string {
	return []string{masterRole, workerRole}
}

// defaultNodeRoles computes the node roles for a new ScanSetting. It never fails: if adaptiveDefaultNodeRoles
// returns an error, it falls back to fallbackDefaultNodeRoles and logs why.
func (m *handlerImpl) defaultNodeRoles() []string {
	roles, err := m.adaptiveDefaultNodeRoles()
	if err != nil {
		fallback := fallbackDefaultNodeRoles()
		log.Warnf("Could not auto-discover node roles, falling back to %v: %v", fallback, err)
		return fallback
	}
	return roles
}

// resolveNodeRolesForUpdate decides the node roles for an update to a ScanSetting that may already exist, given a
// discovery outcome shared across every ScanSetting in the same request or sync batch (so discovery is only run
// once, not once per ScanSetting). existing is the ScanSetting's current state, or nil for one that doesn't exist
// yet. On a discovery failure, a ScanSetting that already has roles keeps them instead of losing them to
// fallbackDefaultNodeRoles: that previously made a transient MachineConfigPool List failure stop targeting roles
// (e.g. "infra") that a prior successful discovery had added.
func resolveNodeRolesForUpdate(discoveredRoles []string, discoverErr error, existing *unstructured.Unstructured) []string {
	if discoverErr == nil {
		return discoveredRoles
	}
	if existing != nil {
		if currentRoles, _, err := unstructured.NestedStringSlice(existing.Object, "roles"); err == nil && len(currentRoles) > 0 {
			log.Warnf("Could not auto-discover node roles, preserving existing roles %v on namespaces/%s/scansettings/%s: %v",
				currentRoles, existing.GetNamespace(), existing.GetName(), discoverErr)
			return currentRoles
		}
	}
	fallback := fallbackDefaultNodeRoles()
	log.Warnf("Could not auto-discover node roles, falling back to %v: %v", fallback, discoverErr)
	return fallback
}

// adaptiveDefaultNodeRoles decides the node roles to target when creating, updating, or reconciling a ScanSetting.
// It returns an error only when discoverNodeRoles itself failed (no informer available or its cache isn't synced),
// so the reconciler can skip instead of clobbering existing ScanSetting roles with a fallback forced by a
// transient failure. Every other outcome is a success and returns a non-empty role list:
//   - ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES off: fallbackDefaultNodeRoles(), without touching the cluster at all.
//   - discovery succeeds but finds no roles (e.g. no MachineConfigPools exist): fallbackDefaultNodeRoles().
//   - discovery succeeds and finds roles: those roles.
func (m *handlerImpl) adaptiveDefaultNodeRoles() ([]string, error) {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return fallbackDefaultNodeRoles(), nil
	}

	roles, err := m.discoverNodeRoles()
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		log.Warnf("No node roles discovered from MachineConfigPools, falling back to %v", fallbackDefaultNodeRoles())
		return fallbackDefaultNodeRoles(), nil
	}
	return roles, nil
}

// discoverNodeRoles derives one role per MachineConfigPool from the MachineConfigPool informer's cache (never a
// live List: the informer and its cache are set up by startMCPRoleInformer, called from Start()). Returns an error
// when the informer isn't available at all (ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES off, non-OpenShift, or it
// failed to start) or its cache hasn't synced within mcpCacheSyncCallTimeout.
func (m *handlerImpl) discoverNodeRoles() ([]string, error) {
	if m.mcpLister == nil {
		return nil, errors.New("MachineConfigPool informer is not available")
	}
	if !m.waitForMCPCacheSync(mcpCacheSyncCallTimeout) {
		return nil, errors.New("MachineConfigPool informer cache is not synced")
	}

	objs, err := m.mcpLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "listing MachineConfigPools from cache")
	}

	roles := set.NewStringSet()
	for _, obj := range objs {
		if role := mcpRole(obj); role != "" {
			roles.Add(role)
		}
	}

	return roles.AsSortedSlice(func(i, j string) bool { return i < j }), nil
}

// startMCPRoleInformer wires up the MachineConfigPool informer that backs node-role auto-discovery. It is a no-op
// (and leaves m.mcpLister nil, so discoverNodeRoles reports an error and every caller falls back to master+worker)
// when ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is off, or when MachineConfigPools aren't listable at all (e.g. a
// non-OpenShift cluster, where the CRD doesn't exist). It must never block Start() indefinitely or make it fail:
// every problem is logged and degrades gracefully instead.
func (m *handlerImpl) startMCPRoleInformer() {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return
	}

	// Probe whether MachineConfigPools exist before starting an informer for them: on non-OpenShift clusters the
	// CRD is absent, and a SharedIndexInformer for a resource that can never be listed would just retry forever
	// in the background for no benefit.
	if err := m.callWithRetry(func(ctx context.Context) error {
		_, err := m.client.Resource(machineConfigPoolGVR).List(ctx, v1.ListOptions{})
		return err
	}); err != nil {
		if kubeAPIErr.IsNotFound(err) || meta.IsNoMatchError(err) {
			log.Infof("MachineConfigPools are not available, likely not an OpenShift cluster; node-role auto-discovery disabled, using %v", fallbackDefaultNodeRoles())
		} else {
			log.Warnf("Could not list MachineConfigPools, node-role auto-discovery disabled, using %v: %v", fallbackDefaultNodeRoles(), err)
		}
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(m.client, 0)
	informer := factory.ForResource(machineConfigPoolGVR)
	if _, err := informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(interface{}) { m.requestReconcile() },
		DeleteFunc: func(interface{}) { m.requestReconcile() },
		UpdateFunc: m.onMCPUpdate,
	}); err != nil {
		log.Warnf("Could not register MachineConfigPool event handler, node-role auto-discovery disabled, using %v: %v", fallbackDefaultNodeRoles(), err)
		return
	}

	m.mcpLister = informer.Lister()
	m.mcpInformerSynced = informer.Informer().HasSynced

	factory.Start(m.stopSignal.Done())
	if !m.waitForMCPCacheSync(mcpInformerStartupSyncTimeout) {
		log.Warnf("Timed out waiting for the MachineConfigPool informer cache to sync; node-role auto-discovery will use the fallback %v until it finishes syncing", fallbackDefaultNodeRoles())
	}
}

// onMCPUpdate is the MachineConfigPool informer's UpdateFunc. Compliance Operator maps a role 1:1 to a node-role
// label, so only a change to the pool's derived role can affect node-role discovery; every other field change
// (e.g. status, machineCount, unrelated labels) is noise that would otherwise trigger a reconcile pass for nothing.
func (m *handlerImpl) onMCPUpdate(oldObj, newObj interface{}) {
	if mcpRole(oldObj) == mcpRole(newObj) {
		return
	}
	m.requestReconcile()
}

// requestReconcile coalesces MachineConfigPool events into a single pending node-role reconcile: a non-blocking
// send on the capacity-1 roleReconcileTrigger channel, drained by run(). A reconcile already pending absorbs any
// further request without blocking the informer's event-processing goroutine, since reconcileNodeRoles always
// reads the latest cluster state whenever it eventually runs.
func (m *handlerImpl) requestReconcile() {
	select {
	case m.roleReconcileTrigger <- struct{}{}:
	default:
	}
}

// waitForMCPCacheSync bounds how long a caller waits for the MachineConfigPool informer's cache to sync, combining
// the given timeout with the handler's lifetime (m.stopSignal). Returns false immediately if the informer was
// never started (m.mcpInformerSynced is nil).
func (m *handlerImpl) waitForMCPCacheSync(timeout time.Duration) bool {
	if m.mcpInformerSynced == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(m.ctx(), timeout)
	defer cancel()
	return cache.WaitForCacheSync(ctx.Done(), m.mcpInformerSynced)
}

// reconcileNodeRoles recomputes the auto-discovered node roles and brings every ACS-managed ScanSetting in line with the
// current cluster topology, so new or removed MachineConfigPools are reflected without the user recreating scan configs.
// It runs on the single-writer run() loop, triggered by requestReconcile (fed by the MachineConfigPool informer's
// event handlers), and is a no-op unless ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is enabled.
func (m *handlerImpl) reconcileNodeRoles() {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return
	}
	if m.disabled.IsDone() || !m.complianceIsReady.IsDone() {
		return
	}
	if m.complianceOperatorInfo.GetNamespace() == "" {
		return
	}

	desiredRoles, err := m.adaptiveDefaultNodeRoles()
	if err != nil {
		// Falling back to master+worker here would strip every custom role (e.g. "infra") from managed ScanSettings
		// on a transient API failure, churning Compliance Operator's per-role scans until the next successful
		// reconcile. Skip this round instead; the existing ScanSettings are left as-is and get re-evaluated the
		// next time a MachineConfigPool event (or cache sync) triggers a reconcile.
		log.Warnf("Skipping node-role reconciliation: %v", err)
		return
	}

	scanSettings, err := m.getResourcesInCluster(complianceoperator.ScanSetting)
	if err != nil {
		// The compliance operator CRDs may be absent (e.g. non-OpenShift). Log and return gracefully; never crash.
		log.Warnf("Could not list ScanSettings to reconcile node roles: %v", err)
		return
	}

	// Bound the whole pass, not just each individual update: without this, a reconcile pass over N ScanSettings
	// could occupy the single-writer loop for up to N*handlerRetryTimeout, see nodeRoleReconcilePassBudget.
	passCtx, cancel := context.WithTimeout(m.ctx(), m.nodeRoleReconcilePassBudget)
	defer cancel()

	var errList errorhelpers.ErrorList
	for _, scanSetting := range scanSettings {
		if err := m.reconcileScanSettingRoles(passCtx, scanSetting, desiredRoles); err != nil {
			errList.AddError(err)
		}
	}
	if err := errList.ToError(); err != nil {
		log.Errorf("Failed to reconcile node roles for one or more ScanSettings: %v", err)
	}
}

// reconcileScanSettingRoles updates a single ScanSetting's top-level roles to the desired set when they differ. The
// comparison is order-insensitive so an equivalent role set never triggers an update (idempotent, avoids CO churn).
// ctx bounds the update: see nodeRoleReconcilePassBudget on its caller.
func (m *handlerImpl) reconcileScanSettingRoles(ctx context.Context, scanSetting unstructured.Unstructured, desiredRoles []string) error {
	name := scanSetting.GetName()
	namespace := scanSetting.GetNamespace()

	currentRoles, _, err := unstructured.NestedStringSlice(scanSetting.Object, "roles")
	if err != nil {
		return errors.Wrapf(err, "reading roles from namespaces/%s/scansettings/%s", namespace, name)
	}
	if set.NewStringSet(desiredRoles...).Equal(set.NewStringSet(currentRoles...)) {
		return nil
	}

	updated := scanSetting.DeepCopy()
	if err := setScanSettingRoles(updated, desiredRoles); err != nil {
		return err
	}

	resI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(namespace)
	return m.callWithRetryWithOnConflictCallbackCtx(ctx,
		func(ctx context.Context) error {
			_, err := resI.Update(ctx, updated, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update roles on namespaces/%s/scansettings/%s", namespace, name)
		},
		func(ctx context.Context) error {
			current, err := resI.Get(ctx, name, v1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "unable to get namespaces/%s/scansettings/%s", namespace, name)
			}
			updated = current
			return setScanSettingRoles(updated, desiredRoles)
		})
}

// setScanSettingRoles writes the desired roles to the ScanSetting's top-level `roles` field
func setScanSettingRoles(scanSetting *unstructured.Unstructured, roles []string) error {
	if err := unstructured.SetNestedStringSlice(scanSetting.Object, roles, "roles"); err != nil {
		return errors.Wrapf(err, "setting roles on namespaces/%s/scansettings/%s", scanSetting.GetNamespace(), scanSetting.GetName())
	}
	return nil
}
