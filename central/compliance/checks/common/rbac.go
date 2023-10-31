package common

import (
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	setPkg "github.com/stackrox/rox/pkg/set"
)

// EffectiveAdmin is the access level of cluster admin.
var EffectiveAdmin = &storage.PolicyRule{
	Verbs:     []string{"*"},
	ApiGroups: []string{""},
	Resources: []string{"*"},
}

// CheckVolumeAccessIsLimited checks that not all service accounts can manipulate volumes.
func CheckVolumeAccessIsLimited(ctx framework.ComplianceContext) {
	// Collect a list of all known service accounts with bound permissions.
	allServiceAccounts := k8srbac.NewSubjectSet()
	for _, binding := range ctx.Data().K8sRoleBindings() {
		for _, subject := range binding.GetSubjects() {
			if subject.GetKind() == storage.SubjectKind_SERVICE_ACCOUNT {
				allServiceAccounts.Add(subject)
			}
		}
	}

	isServiceAccount := func(subject *storage.Subject) bool {
		return subject.Kind == storage.SubjectKind_SERVICE_ACCOUNT
	}

	subjectsWithPersistentVolumeAccess := listSubjectsWithAccess(isServiceAccount, ctx.Data().K8sRoles(), ctx.Data().K8sRoleBindings(), &storage.PolicyRule{
		Verbs:     []string{"*"},
		ApiGroups: []string{""},
		Resources: []string{"persistentvolumes"},
	})
	if allServiceAccounts.Cardinality() == subjectsWithPersistentVolumeAccess.Cardinality() {
		framework.Fail(ctx, "All service accounts have unlimited persistent volume access.")
		return
	}

	subjectsWithPersistentVolumeClaimsAccess := listSubjectsWithAccess(isServiceAccount, ctx.Data().K8sRoles(), ctx.Data().K8sRoleBindings(), &storage.PolicyRule{
		Verbs:     []string{"*"},
		ApiGroups: []string{""},
		Resources: []string{"persistentvolumeclaims"},
	})
	if allServiceAccounts.Cardinality() == subjectsWithPersistentVolumeClaimsAccess.Cardinality() {
		framework.Fail(ctx, "All service accounts have unlimited persistent volume claims access.")
		return
	}

	subjectsWithVolumeAttachmentAccess := listSubjectsWithAccess(isServiceAccount, ctx.Data().K8sRoles(), ctx.Data().K8sRoleBindings(), &storage.PolicyRule{
		Verbs:     []string{"*"},
		ApiGroups: []string{""},
		Resources: []string{"volumeattachments"},
	})
	if allServiceAccounts.Cardinality() == subjectsWithVolumeAttachmentAccess.Cardinality() {
		framework.Fail(ctx, "All service accounts have unlimited volume attachment access.")
		return
	}
	framework.Pass(ctx, "Persistent volume, persistent volume claim, and volume attachment resource accesses are limited.")
}

// AdministratorUsersPresent looks for users with name Admin or Administrator or something similar.
// These should be groups, not shared users.
func AdministratorUsersPresent(ctx framework.ComplianceContext) {
	for _, binding := range ctx.Data().K8sRoleBindings() {
		for _, subject := range binding.GetSubjects() {
			if subject.GetKind() == storage.SubjectKind_USER {
				if adminNames.Contains(strings.ToLower(subject.GetName())) {
					framework.Fail(ctx, "Use the GROUP subject kind instead of USER, when specifying administrative accounts.")
					return
				}
			}
		}
	}
	framework.Pass(ctx, "No shared administrator USERs found.")
}

var adminNames = setPkg.NewFrozenStringSet("admin", "administrator", "root")

// CheckDeploymentsDoNotHaveClusterAccess checks that no deployments are launched with effective cluster admin access.
func CheckDeploymentsDoNotHaveClusterAccess(ctx framework.ComplianceContext, pr *storage.PolicyRule) {
	clusterEvaluator := k8srbac.MakeClusterEvaluator(ctx.Data().K8sRoles(), ctx.Data().K8sRoleBindings())
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		// Check deployment
		if !isKubeSystem(deployment) && clusterEvaluator.ForSubject(k8srbac.GetSubjectForDeployment(deployment)).Grants(pr) {
			framework.Failf(ctx, "deployment has cluster access to %s, this should be scoped down where possible.", proto.MarshalTextString(pr))
		} else {
			framework.Pass(ctx, "No deployments have been launched with cluster admin level access.")
		}
	})

}

// LimitedUsersAndGroupsWithClusterAdminInterpretation interprets LimitedUsersAndGroupsWithClusterAdmin.
const LimitedUsersAndGroupsWithClusterAdminInterpretation = `Widespread use of the cluster-admin role or equivalent access is dangerous. StackRox checks that at most one User or Group is granted the cluster-admin role or equivalent access.`

// LimitedUsersAndGroupsWithClusterAdmin checks that only a single user or group exists with cluster admin access.
func LimitedUsersAndGroupsWithClusterAdmin(ctx framework.ComplianceContext) {
	serviceAccountsWithClusterAdmin := listSubjectsWithAccess(func(subject *storage.Subject) bool {
		return subject.Kind == storage.SubjectKind_USER || subject.Kind == storage.SubjectKind_GROUP
	}, ctx.Data().K8sRoles(), ctx.Data().K8sRoleBindings(), EffectiveAdmin)
	if serviceAccountsWithClusterAdmin.Cardinality() > 1 {
		framework.Fail(ctx, "Multiple User or Group subjects were found with cluster-admin-level access. Typically, a single Group subject is most appropriate.")
		return
	}
	framework.Pass(ctx, "One or fewer User or Group subjects were found with cluster-admin-level access.")
}

// Static helper functions.
///////////////////////////

func listSubjectsWithAccess(predicate func(sub *storage.Subject) bool, roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding, pr *storage.PolicyRule) k8srbac.SubjectSet {
	allSubjects := k8srbac.NewSubjectSet()
	for _, binding := range bindings {
		for _, subject := range binding.GetSubjects() {
			if predicate(subject) {
				allSubjects.Add(subject)
			}
		}
	}

	clusterEvaluator := k8srbac.MakeClusterEvaluator(roles, bindings)
	subjectsWithAccess := k8srbac.NewSubjectSet()
	for _, subject := range allSubjects.ToSlice() {
		if clusterEvaluator.ForSubject(subject).Grants(pr) {
			subjectsWithAccess.Add(subject)
		}
	}
	return subjectsWithAccess
}
