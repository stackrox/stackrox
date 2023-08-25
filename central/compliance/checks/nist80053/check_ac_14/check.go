package checkac14

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	pkgCommon "github.com/stackrox/rox/pkg/compliance/checks/common"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/set"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:AC_14"

	interpretationText = pkgCommon.IsRBACConfiguredCorrectlyInterpretation + `

StackRox also checks that unauthenticated users are only given RBAC access to API methods or URLs that are generally considered safe.`

	systemUnauthenticatedSubject = `system:unauthenticated`
)

var (
	// Pulled from https://github.com/kubernetes/kubernetes/blob/e41bb325c2453fc373826f5cd2b8d9b106038d2f/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#L218
	// We might need to update this as Kube makes updates.
	allowedNonResourceURLs = set.NewFrozenStringSet("/livez", "/readyz", "/healthz", "/version", "/version/")
)

func checkClusterRoleIsSafe(ctx framework.ComplianceContext, clusterRole *storage.K8SRole) bool {
	for _, rule := range clusterRole.GetRules() {
		// Kube doesn't allow this, but no sense in us panic-ing.
		if len(rule.GetVerbs()) == 0 {
			continue
		}
		for _, verb := range rule.GetVerbs() {
			if verb != "get" {
				framework.Failf(ctx, "ClusterRole %q allows unauthenticated users to %q", clusterRole.GetName(), verb)
				return false
			}
		}
		if len(rule.GetApiGroups()) > 0 || len(rule.GetResourceNames()) > 0 || len(rule.GetResources()) > 0 {
			framework.Failf(ctx, "ClusterRole %q allows access to API resources to unauthenticated users", clusterRole.GetName())
			return false
		}
		for _, nonResourceURL := range rule.GetNonResourceUrls() {
			if !allowedNonResourceURLs.Contains(nonResourceURL) {
				framework.Failf(ctx, "ClusterRole %q allows access to non-resource URL %q to unauthenticated users, which is unnecessary", clusterRole.GetName(), nonResourceURL)
				return false
			}
		}
	}
	return true
}

func checkRoleIsSafe(ctx framework.ComplianceContext, role *storage.K8SRole) bool {
	if role.GetClusterRole() {
		return checkClusterRoleIsSafe(ctx, role)
	}
	// Namespaced roles. _Any_ namespaced role that has only one rule is too much to allow an anonymous user.
	if len(role.GetRules()) > 0 {
		framework.Failf(ctx, "Role %q in namespace %q is assigned to unauthenticated users, which is unnecessary", role.GetName(), role.GetNamespace())
		return false
	}
	return true
}

func roleIsInSets(role *storage.K8SRole, clusterRoleIds set.StringSet, namespaceRoleIDs map[string]set.StringSet) bool {
	if role.GetClusterRole() {
		return clusterRoleIds.Contains(role.GetId())
	}
	return namespaceRoleIDs[role.GetNamespace()].Contains(role.GetId())
}

func checkNoExtraPrivilegesForUnauthenticated(ctx framework.ComplianceContext) {
	k8sRoleBindings := ctx.Data().K8sRoleBindings()
	clusterRoleIDs := set.NewStringSet()
	namespaceRoleIDs := make(map[string]set.StringSet)
	for _, binding := range k8sRoleBindings {
		for _, subject := range binding.GetSubjects() {
			if subject.GetName() == systemUnauthenticatedSubject && subject.GetKind() == storage.SubjectKind_GROUP {
				if k8srbac.IsClusterRoleBinding(binding) {
					clusterRoleIDs.Add(binding.GetRoleId())
				} else {
					namespacedRoleIDSet, found := namespaceRoleIDs[binding.GetNamespace()]
					if !found {
						namespacedRoleIDSet = set.NewStringSet()
						namespaceRoleIDs[binding.GetNamespace()] = namespacedRoleIDSet
					}
					namespacedRoleIDSet.Add(binding.GetRoleId())
				}
				break
			}
		}
	}

	configValid := true
	for _, role := range ctx.Data().K8sRoles() {
		if roleIsInSets(role, clusterRoleIDs, namespaceRoleIDs) {
			valid := checkRoleIsSafe(ctx, role)
			if !valid {
				configValid = false
			}
		}
	}
	if configValid {
		framework.Pass(ctx, "Unauthenticated users are given only the minimal required permissions.")
	}
}

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"K8sRoles", "K8sRoleBindings", "HostScraped"},
			InterpretationText: interpretationText,
		}, func(ctx framework.ComplianceContext) {
			checkNoExtraPrivilegesForUnauthenticated(ctx)
		})
}
