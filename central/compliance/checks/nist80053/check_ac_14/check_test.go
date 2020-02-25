package checkac14

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type result struct {
	status  framework.Status
	message string
}

func setupMockCtx(receivedResults *[]result, ctrl *gomock.Controller, k8sRoles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) framework.ComplianceContext {
	mockCtx := mocks.NewMockComplianceContext(ctrl)

	mockCtx.EXPECT().RecordEvidence(gomock.Any(), gomock.Any()).AnyTimes().Do(func(status framework.Status, message string) {
		*receivedResults = append(*receivedResults, result{status, message})
	})
	mockData := mocks.NewMockComplianceDataRepository(ctrl)
	mockCtx.EXPECT().Data().AnyTimes().Return(mockData)
	mockData.EXPECT().K8sRoles().AnyTimes().Return(k8sRoles)
	mockData.EXPECT().K8sRoleBindings().AnyTimes().Return(roleBindings)

	return mockCtx
}

func passed(receivedResults []result, t *testing.T) bool {
	require.NotEmpty(t, receivedResults)
	for _, r := range receivedResults {
		if r.status != framework.PassStatus {
			return false
		}
	}
	return true
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

	acceptableRole, acceptableBinding := createRoleAndBindToSubject(true, "", systemUnauthenciatedSubject, storage.SubjectKind_GROUP, []*storage.PolicyRule{
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

	netpolRole, netpolBinding := createRoleAndBindToSubject(true, "", systemUnauthenciatedSubject, storage.SubjectKind_GROUP, []*storage.PolicyRule{
		{
			Verbs:     []string{"get"},
			ApiGroups: []string{"extensions/v1beta1"},
			Resources: []string{"networkpolicies"},
		},
	})

	namespacedRole, namespacedBinding := createRoleAndBindToSubject(false, "fake-ns", systemUnauthenciatedSubject, storage.SubjectKind_GROUP, []*storage.PolicyRule{
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

			var receivedResults []result
			mockCtx := setupMockCtx(&receivedResults, ctrl, c.k8sRoles, c.k8sRoleBindings)
			checkNoExtraPrivilegesForUnauthenticated(mockCtx)
			assert.Equal(t, c.shouldPass, passed(receivedResults, t))
		})
	}

}
