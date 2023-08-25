package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// ReadResourceVerbs verbs are all possible verbs in a PolicyRule that give some read access.
var ReadResourceVerbs = set.NewFrozenStringSet("get", "list", "watch")

// WriteResourceVerbs verbs are all possible verbs in a PolicyRule that give some write access.
var WriteResourceVerbs = set.NewFrozenStringSet("create", "bind", "patch", "update", "delete", "deletecollection")

// ResourceVerbs verbs are all possible verbs in a PolicyRule that give access.
var ResourceVerbs = WriteResourceVerbs.Union(ReadResourceVerbs)

// DefaultLabel key/value pair that identifies default Kubernetes roles and role bindings
var DefaultLabel = struct {
	Key   string
	Value string
}{Key: "kubernetes.io/bootstrapping", Value: "rbac-defaults"}

// DefaultServiceAccountName is the name of service accounts that get created by default.
const DefaultServiceAccountName = "default"

// IsDefaultRoleBinding identifies default role bindings.
// TODO(): Need to wire labels for this.
func IsDefaultRoleBinding(roleBinding *storage.K8SRoleBinding) bool {
	return roleBinding.GetLabels()[DefaultLabel.Key] == DefaultLabel.Value
}

// IsClusterRoleBinding identifies cluster role bindings.
func IsClusterRoleBinding(binding *storage.K8SRoleBinding) bool {
	// A binding is only binding cluster-wide if cluster role is true _and_ namespace is empty. Namespaced
	// bindings may bind to cluster roles as well.
	return binding.GetClusterRole() && binding.GetNamespace() == ""
}

// EffectiveAdmin represents the rule that grants admin if granted by a policy rule.
var EffectiveAdmin = &storage.PolicyRule{
	Verbs:     []string{"*"},
	ApiGroups: []string{""},
	Resources: []string{"*"},
}

// xxxxxAnything, represents being able to xxxxx on any single resource type. If you can "get deployments", then
// ruleSet.Grants(getAnything) will return true for instance. You do not need to be able to get EVERYTHING, just
// ANYTHING.

// GetAnything is the permission that if granted means something in some api group can have 'get' called on it.
var GetAnything = &storage.PolicyRule{
	Verbs: []string{"get"},
}

// ListAnything is the permission that if granted means something in some api group can have 'list' called on it.
var ListAnything = &storage.PolicyRule{
	Verbs: []string{"list"},
}

// WatchAnything is the permission that if granted means something in some api group can have 'watch' called on it.
var WatchAnything = &storage.PolicyRule{
	Verbs: []string{"watch"},
}

// CreateAnything is the permission that if granted means something in some api group can have 'create' called on it.
var CreateAnything = &storage.PolicyRule{
	Verbs: []string{"create"},
}

// BindAnything is the permission that if granted means something in some api group can have 'bind' called on it.
var BindAnything = &storage.PolicyRule{
	Verbs: []string{"bind"},
}

// PatchAnything is the permission that if granted means something in some api group can have 'patch' called on it.
var PatchAnything = &storage.PolicyRule{
	Verbs: []string{"patch"},
}

// UpdateAnything is the permission that if granted means something in some api group can have 'update' called on it.
var UpdateAnything = &storage.PolicyRule{
	Verbs: []string{"update"},
}

// DeleteAnything is the permission that if granted means something in some api group can have 'delete' called on it.
var DeleteAnything = &storage.PolicyRule{
	Verbs: []string{"delete"},
}

// DeletecollectionAnything is the permission that if granted means something in some api group can have 'deletecollection' called on it.
var DeletecollectionAnything = &storage.PolicyRule{
	Verbs: []string{"deletecollection"},
}
