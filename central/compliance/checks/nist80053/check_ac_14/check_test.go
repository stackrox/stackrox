package checkac14

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/mock/gomock"
)

func setupMockCtx(ctrl *gomock.Controller, k8sRoles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) (framework.ComplianceContext, *testutils.EvidenceRecords) {
	mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)

	mockData.EXPECT().K8sRoles().AnyTimes().Return(k8sRoles)
	mockData.EXPECT().K8sRoleBindings().AnyTimes().Return(roleBindings)

	return mockCtx, records
}

func createRoleAndBindToSubject(clusterRole bool, ns string, subjectName string, subjectKind storage.SubjectKind, rules []*storage.PolicyRule) (*storage.K8SRole, *storage.K8SRoleBinding) {
	roleID := uuid.NewV4().String()
	role := &storage.K8SRole{
		Id:          roleID,
		Name:        roleID,
		Namespace:   ns,
		ClusterRole: clusterRole,
		Rules:       rules,
	}
	bindingID := uuid.NewV4().String()
	roleBinding := &storage.K8SRoleBinding{
		Id:          bindingID,
		Name:        bindingID,
		Namespace:   ns,
		ClusterRole: clusterRole,
		Subjects: []*storage.Subject{
			{
				Name:      subjectName,
				Namespace: ns,
				Kind:      subjectKind,
			},
		},
		RoleId: roleID,
	}

	return role, roleBinding
}

type testCase struct {
	desc            string
	k8sRoles        []*storage.K8SRole
	k8sRoleBindings []*storage.K8SRoleBinding
	shouldPass      bool
}

func TestCheckAC14(t *testing.T) {
	t.Parallel()

	acceptableRole, acceptableBinding := createRoleAndBindToSubject(true, "", systemUnauthenticatedSubject, storage.SubjectKind_GROUP, []*storage.PolicyRule{
		{
			Verbs:           []string{"get"},
			NonResourceUrls: []string{"/healthz"},
		},
	})

	unrelatedRole, unrelatedBinding := createRoleAndBindToSubject(true, "", "unrelated", storage.SubjectKind_GROUP, []*storage.PolicyRule{
		{
			Verbs:     []string{"*"},
			Resources: []string{"*"},
			ApiGroups: []string{"*"},
		},
	})

	netpolRole, netpolBinding := createRoleAndBindToSubject(true, "", systemUnauthenticatedSubject, storage.SubjectKind_GROUP, []*storage.PolicyRule{
		{
			Verbs:     []string{"get"},
			ApiGroups: []string{"extensions/v1beta1"},
			Resources: []string{"networkpolicies"},
		},
	})

	namespacedRole, namespacedBinding := createRoleAndBindToSubject(false, "fake-ns", systemUnauthenticatedSubject, storage.SubjectKind_GROUP, []*storage.PolicyRule{
		{
			Verbs:     []string{"get"},
			ApiGroups: []string{"extensions/v1beta1"},
			Resources: []string{"networkpolicies"},
		},
	})

	for _, tc := range []testCase{
		{
			"Only acceptable",
			[]*storage.K8SRole{acceptableRole},
			[]*storage.K8SRoleBinding{acceptableBinding},
			true,
		},
		{
			"Only unrelated",
			[]*storage.K8SRole{unrelatedRole},
			[]*storage.K8SRoleBinding{unrelatedBinding},
			true,
		},
		{
			"Acceptable and unrelated",
			[]*storage.K8SRole{acceptableRole, unrelatedRole},
			[]*storage.K8SRoleBinding{acceptableBinding, unrelatedBinding},
			true,
		},
		{
			"Netpol role only",
			[]*storage.K8SRole{netpolRole},
			[]*storage.K8SRoleBinding{netpolBinding},
			false,
		},
		{
			"Netpol role",
			[]*storage.K8SRole{acceptableRole, unrelatedRole, netpolRole},
			[]*storage.K8SRoleBinding{acceptableBinding, unrelatedBinding, netpolBinding},
			false,
		},
		{
			"Namespaced role",
			[]*storage.K8SRole{acceptableRole, unrelatedRole, namespacedRole},
			[]*storage.K8SRoleBinding{acceptableBinding, unrelatedBinding, namespacedBinding},
			false,
		},
	} {
		c := tc
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCtx, records := setupMockCtx(ctrl, c.k8sRoles, c.k8sRoleBindings)
			checkNoExtraPrivilegesForUnauthenticated(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}

}
