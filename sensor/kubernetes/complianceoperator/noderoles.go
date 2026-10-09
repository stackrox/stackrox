package complianceoperator

import (
	"context"
	"sort"
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

// nodeRolePrefix is the node-role label prefix used by Compliance Operator to map a ScanSetting role to the nodes it
// targets. It must stay in sync with Compliance Operator's pkg/utils.nodeRolePrefix.
const nodeRolePrefix = "node-role.kubernetes.io/"

// machineConfigPoolGVR is the GVR for MachineConfigPools. These only exist on OpenShift.
var machineConfigPoolGVR = schema.GroupVersionResource{
	Group:    "machineconfiguration.openshift.io",
	Version:  "v1",
	Resource: "machineconfigpools",
}

// getFirstNodeRole extracts the node role from an MCP's spec.nodeSelector.matchLabels, mirroring Compliance Operator's
// GetFirstNodeRole: the first key with the node-role prefix, returned with the prefix trimmed. Returns "" if none.
func getFirstNodeRole(matchLabels map[string]string) string {
	for k := range matchLabels {
		if strings.HasPrefix(k, nodeRolePrefix) {
			return strings.TrimPrefix(k, nodeRolePrefix)
		}
	}
	return ""
}

// defaultNodeRoles computes the node roles for a ScanSetting, bounding the discovery List so it can never block the
// single-writer handler loop.
func (m *handlerImpl) defaultNodeRoles() []string {
	ctx, cancel := context.WithTimeout(m.ctx(), m.handlerAPICallTimeout)
	defer cancel()
	return m.discoverNodeRoles(ctx)
}

// discoverNodeRoles returns the node roles to target when creating or updating a ScanSetting.
//
// When ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is off, it returns the hardcoded master+worker roles without touching the
// cluster (today's behavior). When on, it lists the cluster's MachineConfigPools and derives one role per pool from its
// nodeSelector (including zero-node pools, skipping pools with no node-role label). Discovery must never make applying a
// scan config fail: on any error (MCP API absent, list failure, or no roles found) it logs a warning and falls back to
// master+worker. It never returns an empty role list.
func (m *handlerImpl) discoverNodeRoles(ctx context.Context) []string {
	fallback := []string{masterRole, workerRole}
	if !env.ComplianceAutodiscoverNodeRoles.BooleanSetting() {
		return fallback
	}

	mcpList, err := m.client.Resource(machineConfigPoolGVR).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Warnf("Could not list MachineConfigPools to auto-discover node roles, falling back to %v: %v", fallback, err)
		return fallback
	}

	roles := set.NewStringSet()
	for i := range mcpList.Items {
		matchLabels, _, _ := unstructured.NestedStringMap(mcpList.Items[i].Object, "spec", "nodeSelector", "matchLabels")
		if role := getFirstNodeRole(matchLabels); role != "" {
			roles.Add(role)
		}
	}

	if roles.Cardinality() == 0 {
		log.Warnf("No node roles discovered from MachineConfigPools, falling back to %v", fallback)
		return fallback
	}

	return roles.AsSortedSlice(func(i, j string) bool { return i < j })
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

	desiredRoles := m.defaultNodeRoles()
	sort.Strings(desiredRoles)
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
		return errors.Wrapf(err, "setting roles on scansettings/%s", scanSetting.GetName())
	}
	return nil
}
