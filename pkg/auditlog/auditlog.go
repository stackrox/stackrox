package auditlog

import "github.com/stackrox/rox/pkg/set"

// AllPolicyVerbs is the complete set of API verbs that are valid in audit log
// policy criteria. WATCH and LIST are never valid policy verbs, so they are
// excluded here but always denied by the collection layer.
var AllPolicyVerbs = set.NewFrozenStringSet("CREATE", "DELETE", "GET", "PATCH", "UPDATE")

// AllowedVerbsPerResource is the canonical source of truth for which API verbs
// the compliance agent forwards for each audit log resource. The keys are the
// proto enum names (uppercase, underscore-separated) used by the policy engine.
//
// The compliance agent derives its deny lists from this map. The policy
// validator uses it to reject verb+resource combinations that would never fire.
// The UI maintains a TypeScript copy that must stay in sync.
var AllowedVerbsPerResource = map[string]set.FrozenStringSet{
	"SECRETS":                      set.NewFrozenStringSet("CREATE", "DELETE", "GET", "PATCH", "UPDATE"),
	"CONFIGMAPS":                   set.NewFrozenStringSet("CREATE", "DELETE", "GET", "PATCH", "UPDATE"),
	"CLUSTER_ROLES":                set.NewFrozenStringSet("CREATE", "DELETE", "PATCH", "UPDATE"),
	"CLUSTER_ROLE_BINDINGS":        set.NewFrozenStringSet("CREATE", "DELETE", "PATCH", "UPDATE"),
	"NETWORK_POLICIES":             set.NewFrozenStringSet("CREATE", "DELETE", "PATCH", "UPDATE"),
	"SECURITY_CONTEXT_CONSTRAINTS": set.NewFrozenStringSet("CREATE", "DELETE", "PATCH", "UPDATE"),
	"EGRESS_FIREWALLS":             set.NewFrozenStringSet("CREATE", "DELETE", "PATCH", "UPDATE"),
	"EVENTS":                       set.NewFrozenStringSet("DELETE"),
}

// KubeNameToProtoName maps lowercase Kubernetes API resource names (as they
// appear in the audit log) to the proto enum names used in policy criteria.
var KubeNameToProtoName = map[string]string{
	"secrets":                    "SECRETS",
	"configmaps":                 "CONFIGMAPS",
	"clusterroles":               "CLUSTER_ROLES",
	"clusterrolebindings":        "CLUSTER_ROLE_BINDINGS",
	"networkpolicies":            "NETWORK_POLICIES",
	"securitycontextconstraints": "SECURITY_CONTEXT_CONSTRAINTS",
	"egressfirewalls":            "EGRESS_FIREWALLS",
	"events":                     "EVENTS",
}

// AllowedResourceNames returns the lowercase Kubernetes API resource names
// that are configured for audit log collection.
func AllowedResourceNames() []string {
	names := make([]string, 0, len(KubeNameToProtoName))
	for kubeName := range KubeNameToProtoName {
		names = append(names, kubeName)
	}
	return names
}

// DeniedVerbsByKubeResource returns a map from lowercase Kubernetes resource
// name to the set of verbs that should NOT be forwarded. Each deny set
// includes WATCH and LIST (always denied) plus any policy-valid verbs not in
// the allowed set for that resource.
func DeniedVerbsByKubeResource() map[string]set.FrozenStringSet {
	result := make(map[string]set.FrozenStringSet, len(KubeNameToProtoName))
	for kubeName, protoName := range KubeNameToProtoName {
		allowed, ok := AllowedVerbsPerResource[protoName]
		if !ok {
			continue
		}
		denied := set.NewStringSet("WATCH", "LIST")
		for _, verb := range AllPolicyVerbs.AsSlice() {
			if !allowed.Contains(verb) {
				denied.Add(verb)
			}
		}
		result[kubeName] = denied.Freeze()
	}
	return result
}
