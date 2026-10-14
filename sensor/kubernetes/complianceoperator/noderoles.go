package complianceoperator

import (
	"context"
	"slices"
	"strings"

	"github.com/pkg/errors"
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

// adaptiveDefaultNodeRoles decides the node roles to target when creating or updating a ScanSetting. It returns an
// error only when ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES is enabled and discoverNodeRoles itself failed to list
// MachineConfigPools. Every other outcome is a success and returns a non-empty role list:
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
