package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// ReadResourceVerbs verbs are all possible verbs in a PolicyRule that give some read access.
var ReadResourceVerbs = set.NewStringSet("get", "list", "watch")

// WriteResourceVerbs verbs are all possible verbs in a PolicyRule that give some write access.
var WriteResourceVerbs = set.NewStringSet("create", "bind", "patch", "update", "delete", "deletecollection")

// ResourceVerbs verbs are all possible verbs in a PolicyRule that give access.
var ResourceVerbs = set.NewStringSet(WriteResourceVerbs.Union(ReadResourceVerbs).AsSlice()...)

// ReadURLVerbs verbs are all possible verbs in a PolicyRule that give some read access to a raw URL suffix.
var ReadURLVerbs = set.NewStringSet("get", "head")

// WriteURLVerbs verbs are all possible verbs in a PolicyRule that give some write access to a raw URL suffix.
var WriteURLVerbs = set.NewStringSet("post", "put", "patch", "delete")

// URLVerbs verbs are all possible verbs in a PolicyRule that give some access to a raw URL suffix.
var URLVerbs = set.NewStringSet(WriteURLVerbs.Union(ReadURLVerbs).AsSlice()...)

// DefaultLabel key/value pair that identifies default Kubernetes roles and role bindings
var DefaultLabel = struct {
	Key   string
	Value string
}{Key: "kubernetes.io/bootstrapping", Value: "rbac-defaults"}

// IsDefaultRole identifies default roles.
// TODO(): Need to wire labels for this.
func IsDefaultRole(role *storage.K8SRole) bool {
	return role.GetLabels()[DefaultLabel.Key] == DefaultLabel.Value
}

// IsDefaultRoleBinding identifies default role bindings.
// TODO(): Need to wire labels for this.
func IsDefaultRoleBinding(roleBinding *storage.K8SRoleBinding) bool {
	return roleBinding.GetLabels()[DefaultLabel.Key] == DefaultLabel.Value
}

// DefaultServiceAccountName is the name of service accounts that get created by default.
const DefaultServiceAccountName = "default"

// IsDefaultServiceAccountSubject identifies subjects that are default service accounts.
func IsDefaultServiceAccountSubject(sub *storage.Subject) bool {
	return sub.GetKind() == storage.SubjectKind_SERVICE_ACCOUNT && sub.GetName() == DefaultServiceAccountName
}

// IsReadOnlyPolicyRule returns if the rule is 'read only', that is only allows reading of it's resource.
func IsReadOnlyPolicyRule(rule *storage.PolicyRule) bool {
	for _, verb := range rule.GetVerbs() {
		if !ReadResourceVerbs.Contains(verb) {
			return false
		}
	}
	return true
}

// IsWriteOnlyPolicyRule returns if the rule is 'write only', that is only allows writing of it's resource.
func IsWriteOnlyPolicyRule(rule *storage.PolicyRule) bool {
	for _, verb := range rule.GetVerbs() {
		if !WriteResourceVerbs.Contains(verb) {
			return false
		}
	}
	return true
}

// CanWriteAResource returns if there is any core api resource that can be written in the policy set.
func CanWriteAResource(ruleSet PolicyRuleSet) bool {
	return ruleSet.Grants(CreateAnything) ||
		ruleSet.Grants(BindAnything) ||
		ruleSet.Grants(PatchAnything) ||
		ruleSet.Grants(UpdateAnything) ||
		ruleSet.Grants(DeleteAnything) ||
		ruleSet.Grants(DeletecollectionAnything)
}

// CanReadAResource returns if there is any core api resource that can be read in the policy set.
func CanReadAResource(ruleSet PolicyRuleSet) bool {
	return ruleSet.Grants(GetAnything) ||
		ruleSet.Grants(ListAnything) ||
		ruleSet.Grants(WatchAnything)
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

// GetAnything is the permission that if granted means something in the core api can have 'get' called on it.
var GetAnything = &storage.PolicyRule{
	Verbs:     []string{"get"},
	ApiGroups: []string{""},
}

// ListAnything is the permission that if granted means something in the core api can have 'list' called on it.
var ListAnything = &storage.PolicyRule{
	Verbs:     []string{"list"},
	ApiGroups: []string{""},
}

// WatchAnything is the permission that if granted means something in the core api can have 'watch' called on it.
var WatchAnything = &storage.PolicyRule{
	Verbs:     []string{"watch"},
	ApiGroups: []string{""},
}

// CreateAnything is the permission that if granted means something in the core api can have 'create' called on it.
var CreateAnything = &storage.PolicyRule{
	Verbs:     []string{"create"},
	ApiGroups: []string{""},
}

// BindAnything is the permission that if granted means something in the core api can have 'bind' called on it.
var BindAnything = &storage.PolicyRule{
	Verbs:     []string{"bind"},
	ApiGroups: []string{""},
}

// PatchAnything is the permission that if granted means something in the core api can have 'patch' called on it.
var PatchAnything = &storage.PolicyRule{
	Verbs:     []string{"patch"},
	ApiGroups: []string{""},
}

// UpdateAnything is the permission that if granted means something in the core api can have 'update' called on it.
var UpdateAnything = &storage.PolicyRule{
	Verbs:     []string{"update"},
	ApiGroups: []string{""},
}

// DeleteAnything is the permission that if granted means something in the core api can have 'delete' called on it.
var DeleteAnything = &storage.PolicyRule{
	Verbs:     []string{"delete"},
	ApiGroups: []string{""},
}

// DeletecollectionAnything is the permission that if granted means something in the core api can have 'deletecollection' called on it.
var DeletecollectionAnything = &storage.PolicyRule{
	Verbs:     []string{"deletecollection"},
	ApiGroups: []string{""},
}
