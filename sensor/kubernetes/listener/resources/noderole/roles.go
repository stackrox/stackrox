package noderole

import (
	"strings"

	mcpv1 "github.com/openshift/api/machineconfiguration/v1"
	"github.com/stackrox/rox/pkg/set"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// nodeRoleLabelPrefix is the prefix of the node labels that mark the role of
	// a node, e.g. node-role.kubernetes.io/infra="". The compliance operator
	// selects the nodes to scan with these labels, so the roles written into
	// ScanSettings must match the label suffix.
	nodeRoleLabelPrefix = "node-role.kubernetes.io/"

	// machineConfigRoleLabelKey is the label that
	// MachineConfigPool.spec.machineConfigSelector uses to select the
	// MachineConfigs of a role, e.g. machineconfiguration.openshift.io/role=infra.
	machineConfigRoleLabelKey = "machineconfiguration.openshift.io/role"
)

// RolesFromMachineConfigPool returns the node roles targeted by the given
// MachineConfigPool.
//
// Roles are primarily derived from the node-role.kubernetes.io/<role> labels
// referenced by spec.nodeSelector, as those are the labels the compliance
// operator matches nodes against. The roles referenced by
// spec.machineConfigSelector via the machineconfiguration.openshift.io/role
// label are included as well, so pools whose nodeSelector references no node
// role label (e.g. an empty catch-all selector) still contribute a role.
// Both matchLabels and matchExpressions are considered.
func RolesFromMachineConfigPool(mcp *mcpv1.MachineConfigPool) set.StringSet {
	roles := set.NewStringSet()
	if mcp == nil {
		return roles
	}
	addNodeSelectorRoles(roles, mcp.Spec.NodeSelector)
	addMachineConfigSelectorRoles(roles, mcp.Spec.MachineConfigSelector)
	return roles
}

// addNodeSelectorRoles adds the roles referenced by the node role labels of a
// node selector, e.g. node-role.kubernetes.io/infra="" adds "infra".
func addNodeSelectorRoles(roles set.StringSet, selector *metav1.LabelSelector) {
	if selector == nil {
		return
	}
	for label := range selector.MatchLabels {
		addNodeLabelRole(roles, label)
	}
	for _, req := range selector.MatchExpressions {
		// Negative operators select nodes without the role label and Gt/Lt
		// only apply to numeric values, which node role labels never have.
		if req.Operator != metav1.LabelSelectorOpIn && req.Operator != metav1.LabelSelectorOpExists {
			continue
		}
		addNodeLabelRole(roles, req.Key)
	}
}

func addNodeLabelRole(roles set.StringSet, label string) {
	role, ok := strings.CutPrefix(label, nodeRoleLabelPrefix)
	if !ok || role == "" {
		return
	}
	roles.Add(role)
}

// addMachineConfigSelectorRoles adds the roles referenced by the
// machineconfiguration.openshift.io/role label of a machine config selector,
// e.g. machineconfiguration.openshift.io/role=infra adds "infra".
func addMachineConfigSelectorRoles(roles set.StringSet, selector *metav1.LabelSelector) {
	if selector == nil {
		return
	}
	addMachineConfigRole(roles, selector.MatchLabels[machineConfigRoleLabelKey])
	for _, req := range selector.MatchExpressions {
		if req.Key != machineConfigRoleLabelKey || req.Operator != metav1.LabelSelectorOpIn {
			continue
		}
		for _, role := range req.Values {
			addMachineConfigRole(roles, role)
		}
	}
}

func addMachineConfigRole(roles set.StringSet, role string) {
	if role != "" {
		roles.Add(role)
	}
}
