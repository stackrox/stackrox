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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// nodeRolePrefix is the node-role label prefix Compliance Operator uses (pkg/utils.nodeRolePrefix) to map a
// ScanSetting role to the nodes it targets.
const nodeRolePrefix = "node-role.kubernetes.io/"

// allRolesRole is Compliance Operator's reserved ScanSetting role value meaning "every node", which CO rejects
// when combined with any other role.
const allRolesRole = "@all"

const (
	// mcpInformerStartupSyncTimeout bounds the startMCPRoleInformer goroutine's wait for the initial cache sync
	// before it logs a slow-sync warning.
	mcpInformerStartupSyncTimeout = 10 * time.Second

	// mcpCacheSyncCallTimeout bounds the sync check discoverNodeRoles performs on every create/update, so a cache
	// still catching up (e.g. right after Start()) can never block the single-writer handler loop more than this long.
	mcpCacheSyncCallTimeout = 2 * time.Second

	// defaultNodeRoleReconcilePassBudget bounds how long a single reconcileNodeRoles pass may run across all
	// ScanSettings. Reconciliation runs on the same single-writer loop that processes Central requests, so an
	// unbounded pass (one retry-timeout-bounded update per ScanSetting) would delay request processing by
	// multiples of handlerRetryTimeout as the number of managed ScanSettings grows. Kept below
	// componentProcessTimeout (sensor/common/sensor/component_queue.go, 30s, not exported).
	defaultNodeRoleReconcilePassBudget = 20 * time.Second

	// minNodeRoleReconcileInterval is the floor for the periodic reconcile interval, so a too-small configured value
	// can't spin the single-writer run() loop.
	minNodeRoleReconcileInterval = 2 * time.Minute
)

// resolveNodeRoleReconcileInterval returns the configured reconcile interval, clamped to minNodeRoleReconcileInterval.
func resolveNodeRoleReconcileInterval() time.Duration {
	if interval := env.ComplianceNodeRoleReconcileInterval.DurationSetting(); interval > minNodeRoleReconcileInterval {
		return interval
	}
	return minNodeRoleReconcileInterval
}

// machineConfigPoolGVR is the GVR for MachineConfigPools. These only exist on OpenShift.
var machineConfigPoolGVR = schema.GroupVersionResource{
	Group:    "machineconfiguration.openshift.io",
	Version:  "v1",
	Resource: "machineconfigpools",
}

// getFirstNodeRole extracts the node role from an MCP's spec.nodeSelector.matchLabels.
// There should be only one entry - if there are more, we pick the first.
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

// mcpRole returns the node role derived from a MachineConfigPool informer object.
func mcpRole(obj interface{}) string {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	matchLabels, _, _ := unstructured.NestedStringMap(u.Object, "spec", "nodeSelector", "matchLabels")
	return getFirstNodeRole(matchLabels)
}

// fallbackDefaultNodeRoles are the static node roles used when discovery is off, unavailable, or finds nothing.
func fallbackDefaultNodeRoles() []string {
	return []string{masterRole, workerRole}
}

// nodeRoleOutcome classifies a discovery attempt so each caller picks its own policy.
type nodeRoleOutcome int

const (
	// nodeRolesDisabled: ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is off. The fallback is the answer (legacy behavior).
	nodeRolesDisabled nodeRoleOutcome = iota
	// nodeRolesDiscovered: discovery found at least one role.
	nodeRolesDiscovered
	// nodeRolesEmpty: synced cache, but no MachineConfigPool defines a node role.
	nodeRolesEmpty
	// nodeRolesUnavailable: informer missing or cache not synced (non-OpenShift, transient failure, or startup).
	nodeRolesUnavailable
)

// roleDiscoveryResult is the outcome of discoverDefaultNodeRoles.
type roleDiscoveryResult struct {
	roles   []string // set only when outcome == nodeRolesDiscovered
	outcome nodeRoleOutcome
	err     error // set only when outcome == nodeRolesUnavailable, for logging
}

// discoverDefaultNodeRoles runs discovery and classifies the result.
func (m *handlerImpl) discoverDefaultNodeRoles() roleDiscoveryResult {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return roleDiscoveryResult{outcome: nodeRolesDisabled}
	}
	roles, err := m.discoverNodeRoles()
	switch {
	case err != nil:
		return roleDiscoveryResult{outcome: nodeRolesUnavailable, err: err}
	case len(roles) == 0:
		return roleDiscoveryResult{outcome: nodeRolesEmpty}
	default:
		return roleDiscoveryResult{roles: roles, outcome: nodeRolesDiscovered}
	}
}

// reason renders an outcome for a log line.
func (d roleDiscoveryResult) reason() string {
	switch d.outcome {
	case nodeRolesDisabled:
		return "node-role auto-discovery is disabled"
	case nodeRolesEmpty:
		return "no MachineConfigPool defines a node role"
	default:
		if d.err != nil {
			return d.err.Error()
		}
		return "MachineConfigPool informer unavailable"
	}
}

// rolesForCreate picks the roles for a new ScanSetting: discovered roles, else the fallback (nothing to preserve).
func (d roleDiscoveryResult) rolesForCreate() []string {
	if d.outcome == nodeRolesDiscovered {
		return d.roles
	}
	if d.outcome != nodeRolesDisabled {
		log.Warnf("Could not auto-discover node roles, falling back to %v: %s", fallbackDefaultNodeRoles(), d.reason())
	}
	return fallbackDefaultNodeRoles()
}

// rolesForUpdate picks the roles for a possibly-existing ScanSetting. Discovered roles win; flag off overwrites with the
// fallback (legacy). On a non-authoritative outcome an existing ScanSetting keeps its roles, so a transient failure or
// empty cache can't strip a custom role; a not-yet-existing one still gets the fallback.
func (d roleDiscoveryResult) rolesForUpdate(existing *unstructured.Unstructured) []string {
	switch d.outcome {
	case nodeRolesDiscovered:
		return d.roles
	case nodeRolesDisabled:
		return fallbackDefaultNodeRoles()
	default:
		if existing != nil {
			if currentRoles, _, err := unstructured.NestedStringSlice(existing.Object, "roles"); err == nil && len(currentRoles) > 0 {
				log.Warnf("Could not auto-discover node roles, preserving existing roles %v on namespaces/%s/scansettings/%s: %s",
					currentRoles, existing.GetNamespace(), existing.GetName(), d.reason())
				return currentRoles
			}
		}
		log.Warnf("Could not auto-discover node roles, falling back to %v: %s", fallbackDefaultNodeRoles(), d.reason())
		return fallbackDefaultNodeRoles()
	}
}

// discoverNodeRoles derives one role per MachineConfigPool from the MachineConfigPool informer's cache.
func (m *handlerImpl) discoverNodeRoles() ([]string, error) {
	lister := m.getMCPLister()
	if lister == nil {
		return nil, errors.New("MachineConfigPool informer is not available")
	}
	if !m.waitForMCPCacheSync(mcpCacheSyncCallTimeout) {
		return nil, errors.New("MachineConfigPool informer cache is not synced")
	}

	objs, err := lister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "listing MachineConfigPools from cache")
	}

	roles := set.NewStringSet()
	for _, obj := range objs {
		// allRolesRole is a reserved ScanSetting role value CO treats specially (targets every node) and rejects
		// combined with any other role. It should never show up as an actual node-role label, but skip it
		// defensively so a ScanSetting update is never rejected if it somehow did.
		if role := mcpRole(obj); role != "" && role != allRolesRole {
			roles.Add(role)
		}
	}

	return roles.AsSortedSlice(func(i, j string) bool { return i < j }), nil
}

func (m *handlerImpl) getMCPLister() cache.GenericLister {
	m.mcpMu.RLock()
	defer m.mcpMu.RUnlock()
	return m.mcpLister
}

func (m *handlerImpl) getMCPInformerSynced() cache.InformerSynced {
	m.mcpMu.RLock()
	defer m.mcpMu.RUnlock()
	return m.mcpInformerSynced
}

func (m *handlerImpl) setMCPInformer(lister cache.GenericLister, synced cache.InformerSynced) {
	m.mcpMu.Lock()
	defer m.mcpMu.Unlock()
	m.mcpLister = lister
	m.mcpInformerSynced = synced
}

// probeMachineConfigPools issues a single, non-retried List call to detect the case where MachineConfigPools don't exist
// at all (e.g. a non-OpenShift cluster).
func (m *handlerImpl) probeMachineConfigPools(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, m.handlerAPICallTimeout)
	defer cancel()
	_, err := m.client.Resource(machineConfigPoolGVR).List(callCtx, v1.ListOptions{})
	return errors.Wrap(err, "listing MachineConfigPools")
}

// startMCPRoleInformer wires up the MachineConfigPool informer that backs node-role auto-discovery.
func (m *handlerImpl) startMCPRoleInformer() {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return
	}

	// A single, non-retried probe first: on non-OpenShift clusters the MachineConfigPool CRD is permanently
	// absent (NotFound).
	if err := m.probeMachineConfigPools(m.ctx()); err != nil {
		if kubeAPIErr.IsNotFound(err) {
			log.Infof("MachineConfigPools are not available, likely not an OpenShift cluster; node-role auto-discovery disabled, using %v", fallbackDefaultNodeRoles())
			return
		}
		// Anything else might be transient (e.g. the API server briefly unreachable during startup): retry
		// before giving up, unlike the permanent case above.
		if err := m.callWithRetry(m.probeMachineConfigPools); err != nil {
			log.Warnf("Could not list MachineConfigPools, node-role auto-discovery disabled, using %v: %v", fallbackDefaultNodeRoles(), err)
			return
		}
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

	m.setMCPInformer(informer.Lister(), informer.Informer().HasSynced)

	factory.Start(m.stopSignal.Done())
	if !m.waitForMCPCacheSync(mcpInformerStartupSyncTimeout) {
		log.Warnf("Timed out waiting for the MachineConfigPool informer cache to sync; node-role auto-discovery will use the fallback %v until it finishes syncing", fallbackDefaultNodeRoles())
	}
}

// onMCPUpdate is the MachineConfigPool informer's UpdateFunc. Compliance Operator maps a role 1:1 to a node-role
// label, so only a change to the pool's derived role can affect node-role discovery; every other field change
// (e.g. status, machineCount, unrelated labels) can be ignored.
func (m *handlerImpl) onMCPUpdate(oldObj, newObj interface{}) {
	if mcpRole(oldObj) == mcpRole(newObj) {
		return
	}
	m.requestReconcile()
}

// requestReconcile coalesces MachineConfigPool events into a single pending node-role reconcile: a non-blocking
// send on the capacity-1 roleReconcileTrigger channel, drained by run().
func (m *handlerImpl) requestReconcile() {
	select {
	case m.roleReconcileTrigger <- struct{}{}:
	default:
	}
}

// waitForMCPCacheSync bounds how long a caller waits for the MachineConfigPool informer's cache to sync, combining
// the given timeout with the handler's lifetime (m.stopSignal).
func (m *handlerImpl) waitForMCPCacheSync(timeout time.Duration) bool {
	synced := m.getMCPInformerSynced()
	if synced == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(m.ctx(), timeout)
	defer cancel()
	return cache.WaitForCacheSync(ctx.Done(), synced)
}

// reconcileNodeRoles recomputes the auto-discovered node roles and brings every ACS-managed ScanSetting in line with the
// current cluster topology, so new or removed MachineConfigPools are reflected without the user recreating scan configs.
func (m *handlerImpl) reconcileNodeRoles() {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return
	}
	if m.disabled.IsDone() || !m.complianceIsReady.IsDone() {
		// A trigger that lands here is dropped; it is re-requested when compliance becomes ready
		// (reconcileOnceReady) or is re-enabled (enableCompliance), and the periodic tick is a further
		// backstop. Log it so the drop is visible when troubleshooting at Debug level.
		log.Debugf("Skipping node-role reconciliation: compliance not active (disabled=%t, ready=%t)",
			m.disabled.IsDone(), m.complianceIsReady.IsDone())
		return
	}
	if m.complianceOperatorInfo.GetNamespace() == "" {
		return
	}

	disc := m.discoverDefaultNodeRoles()
	if disc.outcome != nodeRolesDiscovered {
		// Skip if roles were not discovered, to avoid removing roles that were previously present in ScanSettings.
		// Without this, transient errors in discovery would temporarily yield incorrect ScanSettings.
		log.Warnf("Skipping node-role reconciliation: %s", disc.reason())
		return
	}
	desiredRoles := disc.roles

	// Bound the whole pass, including the initial ScanSetting list below, not just each individual update:
	// without this, a reconcile pass over N ScanSettings could occupy the single-writer loop for up to
	// handlerRetryTimeout (the list) plus N*handlerRetryTimeout (the updates), see nodeRoleReconcilePassBudget.
	passCtx, cancel := context.WithTimeout(m.ctx(), m.nodeRoleReconcilePassBudget)
	defer cancel()

	scanSettings, err := m.getResourcesInClusterCtx(passCtx, complianceoperator.ScanSetting)
	if err != nil {
		// The compliance operator CRDs may be absent (e.g. non-OpenShift). Log and return gracefully; never crash.
		log.Warnf("Could not list ScanSettings to reconcile node roles: %v", err)
		return
	}

	var errList errorhelpers.ErrorList
	for _, scanSetting := range scanSettings {
		if err := m.reconcileScanSettingRoles(passCtx, scanSetting, desiredRoles); err != nil {
			errList.AddError(err)
		}
	}
	if err := errList.ToError(); err != nil {
		log.Errorf("Failed to reconcile node roles for one or more ScanSettings: %v", err)
	}

	// Budget ran out before every ScanSetting was processed: re-queue to finish the rest. Coalesced and convergent, so
	// it cannot spin. A stopSignal also surfaces here, where the re-queue is harmless (run() is exiting).
	if passCtx.Err() != nil {
		log.Debugf("Node-role reconcile pass exceeded its %s budget; re-queuing", m.nodeRoleReconcilePassBudget)
		m.requestReconcile()
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

	log.Debugf("Reconciling node roles on namespaces/%s/scansettings/%s: %v -> %v", namespace, name, currentRoles, desiredRoles)

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
