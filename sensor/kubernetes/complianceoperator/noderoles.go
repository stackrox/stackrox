package complianceoperator

import (
	"context"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nodeRolePrefix is the node-role label prefix Compliance Operator uses (pkg/utils.nodeRolePrefix) to map a
// ScanSetting role to the nodes it targets. We can't check that constant directly (unexported, and importing the
// package that holds it pulls in a large unrelated dependency tree), so TestComplianceV2AutodiscoverNodeRoles
// verifies this two ways instead: against real Node labels, and against Compliance Operator's own node-targeting
// behavior (does a scan for a discovered role actually match real Nodes).
const nodeRolePrefix = "node-role.kubernetes.io/"

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

// fallbackDefaultNodeRoles are the static node roles used when ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is off, or as
// a safe default when discovery is enabled but fails or finds nothing.
func fallbackDefaultNodeRoles() []string {
	return []string{masterRole, workerRole}
}

// defaultNodeRoles computes the node roles for a new ScanSetting, bounding the discovery List so it can never block
// the single-writer handler loop. It never fails: if adaptiveDefaultNodeRoles returns an error, it falls back to
// fallbackDefaultNodeRoles and logs why.
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
// It returns an error only when ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is enabled and discoverNodeRoles itself
// failed to list MachineConfigPools, so the reconciler can skip instead of clobbering existing ScanSetting roles
// with a fallback forced by a transient failure. Every other outcome is a success and returns a non-empty role list:
//   - ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES off: fallbackDefaultNodeRoles(), without calling discoverNodeRoles or
//     touching the cluster at all.
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

// discoverNodeRoles lists the cluster's MachineConfigPools and derives one role per pool from its nodeSelector.
// Returns an error if MachineConfigPools can't be listed.
func (m *handlerImpl) discoverNodeRoles() ([]string, error) {
	var mcpList *unstructured.UnstructuredList
	err := m.callWithRetry(func(ctx context.Context) error {
		var err error
		mcpList, err = m.client.Resource(machineConfigPoolGVR).List(ctx, v1.ListOptions{})
		return errors.Wrap(err, "listing MachineConfigPools")
	})
	if err != nil {
		return nil, err
	}

	roles := set.NewStringSet()
	for i := range mcpList.Items {
		matchLabels, _, _ := unstructured.NestedStringMap(mcpList.Items[i].Object, "spec", "nodeSelector", "matchLabels")
		if role := getFirstNodeRole(matchLabels); role != "" {
			roles.Add(role)
		}
	}

	return roles.AsSortedSlice(func(i, j string) bool { return i < j }), nil
}

// reconcileNodeRoles recomputes the auto-discovered node roles and brings every ACS-managed ScanSetting in line with the
// current cluster topology, so new or removed MachineConfigPools are reflected without the user recreating scan configs.
// It runs on the single-writer run() loop and is a no-op unless ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is enabled.
func (m *handlerImpl) reconcileNodeRoles() {
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return
	}
	// Mirror the gating used by the create/update/sync paths: do nothing while compliance is disabled, before the
	// compliance operator is ready, or while its namespace is unknown.
	if m.disabled.IsDone() || !m.complianceIsReady.IsDone() {
		return
	}
	if m.complianceOperatorInfo.GetNamespace() == "" {
		return
	}

	desiredRoles, err := m.adaptiveDefaultNodeRoles()
	if err != nil {
		// Falling back to master+worker here would strip every custom role (e.g. "infra") from managed ScanSettings
		// on a transient API failure, churning Compliance Operator's per-role scans until the next successful tick.
		// Skip this round instead; the existing ScanSettings are left as-is and get re-evaluated on the next tick.
		log.Warnf("Skipping node-role reconciliation: could not list MachineConfigPools to discover node roles: %v", err)
		return
	}
	// desiredRoles is already sorted: adaptiveDefaultNodeRoles returns either discoverNodeRoles's sorted output or
	// fallbackDefaultNodeRoles's {master, worker} literal, which is sorted too. Kept sorted here because it's what
	// gets persisted into ScanSetting.roles below, for deterministic output.
	desiredSet := set.NewStringSet(desiredRoles...)

	scanSettings, err := m.getResourcesInCluster(complianceoperator.ScanSetting)
	if err != nil {
		// The compliance operator CRDs may be absent (e.g. non-OpenShift). Log and return gracefully; never crash.
		log.Warnf("Could not list ScanSettings to reconcile node roles: %v", err)
		return
	}

	var errList errorhelpers.ErrorList
	for _, scanSetting := range scanSettings {
		if err := m.reconcileScanSettingRoles(scanSetting, desiredRoles, desiredSet); err != nil {
			errList.AddError(err)
		}
	}
	if err := errList.ToError(); err != nil {
		log.Errorf("Failed to reconcile node roles for one or more ScanSettings: %v", err)
	}
}

// reconcileScanSettingRoles updates a single ScanSetting's top-level roles to the desired set when they differ. The
// comparison is order-insensitive so an equivalent role set never triggers an update (idempotent, avoids CO churn).
func (m *handlerImpl) reconcileScanSettingRoles(scanSetting unstructured.Unstructured, desiredRoles []string, desiredSet set.Set[string]) error {
	name := scanSetting.GetName()
	namespace := scanSetting.GetNamespace()

	currentRoles, _, err := unstructured.NestedStringSlice(scanSetting.Object, "roles")
	if err != nil {
		return errors.Wrapf(err, "reading roles from namespaces/%s/scansettings/%s", namespace, name)
	}
	if desiredSet.Equal(set.NewStringSet(currentRoles...)) {
		return nil
	}

	updated := scanSetting.DeepCopy()
	if err := setScanSettingRoles(updated, desiredRoles); err != nil {
		return err
	}

	resI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(namespace)
	return m.callWithRetryWithOnConflictCallback(
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

// setScanSettingRoles writes the desired roles to the ScanSetting's top-level `roles` field (json tag `roles,omitempty`
// on v1alpha1.ScanSetting), not under `spec`.
func setScanSettingRoles(scanSetting *unstructured.Unstructured, roles []string) error {
	if err := unstructured.SetNestedStringSlice(scanSetting.Object, roles, "roles"); err != nil {
		return errors.Wrapf(err, "setting roles on namespaces/%s/scansettings/%s", scanSetting.GetNamespace(), scanSetting.GetName())
	}
	return nil
}
