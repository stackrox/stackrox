package complianceoperator

import (
	"context"
	"strings"

	"github.com/stackrox/rox/pkg/env"
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
