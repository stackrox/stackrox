package analysis

import (
	"github.com/stackrox/rox/generated/storage"
)

// Label key/value pair that identifies default Kubernetes roles and role bindings
var defaultLabel = struct {
	Key   string
	Value string
}{Key: "kubernetes.io/bootstrapping", Value: "rbac-defaults"}

// IsDefaultRole identifies default roles.
// TODO(): Need to wire labels for this.
func IsDefaultRole(role *storage.K8SRole) bool {
	return role.GetLabels()[defaultLabel.Key] == defaultLabel.Value
}

// IsDefaultRoleBinding identifies default role bindings.
// TODO(): Need to wire labels for this.
func IsDefaultRoleBinding(roleBinding *storage.K8SRoleBinding) bool {
	return roleBinding.GetLabels()[defaultLabel.Key] == defaultLabel.Value
}

// Name of service accounts that get created by default.
const defaultServiceAccountName = "default"

// IsDefaultServiceAccountSubject identifies subjects that are default service accounts.
func IsDefaultServiceAccountSubject(sub *storage.Subject) bool {
	return sub.GetKind() == storage.SubjectKind_SERVICE_ACCOUNT && sub.GetName() == defaultServiceAccountName
}
