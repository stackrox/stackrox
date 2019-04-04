package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// ReadResourceVerbs verbs are all possible verbs in a PolicyRule that give some read access.
var ReadResourceVerbs = set.NewStringSet("get", "list", "watch")

// WriteResourceVerbs verbs are all possible verbs in a PolicyRule that give some write access.
var WriteResourceVerbs = set.NewStringSet("create", "patch", "update", "delete", "deletecollection")

// ResourceVerbs verbs are all possible verbs in a PolicyRule that give access.
var ResourceVerbs = set.NewStringSet(WriteResourceVerbs.Union(ReadResourceVerbs).AsSlice()...)

// ReadURLVerbs verbs are all possible verbs in a PolicyRule that give some read access to a raw URL suffix.
var ReadURLVerbs = set.NewStringSet("get", "head")

// WriteURLVerbs verbs are all possible verbs in a PolicyRule that give some write access to a raw URL suffix.
var WriteURLVerbs = set.NewStringSet("post", "put", "patch", "delete")

// URLVerbs verbs are all possible verbs in a PolicyRule that give some access to a raw URL suffix.
var URLVerbs = set.NewStringSet(WriteURLVerbs.Union(ReadURLVerbs).AsSlice()...)

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
