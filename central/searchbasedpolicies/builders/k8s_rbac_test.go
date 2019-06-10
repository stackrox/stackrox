package builders

import (
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	bindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockClusters        *clusterMocks.MockDataStore
	mocksRoles          *roleMocks.MockDataStore
	mockBindings        *bindingMocks.MockDataStore
	mockServiceAccounts *serviceAccountMocks.MockDataStore

	tested *K8sRBACQueryBuilder
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockClusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.mocksRoles = roleMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockBindings = bindingMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockServiceAccounts = serviceAccountMocks.NewMockDataStore(suite.mockCtrl)

	suite.tested = &K8sRBACQueryBuilder{
		Clusters:        suite.mockClusters,
		K8sRoles:        suite.mocksRoles,
		K8sBindings:     suite.mockBindings,
		ServiceAccounts: suite.mockServiceAccounts,
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestConstructCorrectQuery() {
	clusters := []*storage.Cluster{
		{
			Id: "c1",
		},
		{
			Id: "c2",
		},
	}
	isInCluster1 := search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery()
	isInCluster2 := search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c2").ProtoQuery()

	serviceAccountsC1 := []*storage.ServiceAccount{
		{
			Name:      "sa1",
			Namespace: "n1",
			ClusterId: "c1",
		},
		{
			Name:      "sa2",
			Namespace: "n1",
			ClusterId: "c1",
		},
		{
			Name:      "sa1",
			Namespace: "n2",
			ClusterId: "c1",
		},
	}
	serviceAccountsC2 := []*storage.ServiceAccount{
		{
			Name:      "sa1",
			Namespace: "n1",
			ClusterId: "c2",
		},
	}

	rolesC1 := []*storage.K8SRole{
		{
			Id:        "r1",
			ClusterId: "c1",
			Namespace: "n1",
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
		{
			Id:        "r2",
			ClusterId: "c1",
			Namespace: "n2",
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"create"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
		{
			Id:          "r3",
			ClusterId:   "c1",
			Namespace:   "n1",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					ApiGroups: []string{""},
				},
			},
		},
	}
	rolesC2 := []*storage.K8SRole{
		{
			Id:        "r4",
			ClusterId: "c2",
			Namespace: "n1",
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
		{
			Id:        "r5",
			ClusterId: "c2",
			Namespace: "n1",
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"create"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
	}

	bindingsC1 := []*storage.K8SRoleBinding{
		{
			Id:        "b1",
			RoleId:    "r1",
			Namespace: "n1",
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa2",
					Namespace: "n1",
				},
			},
		},
		{
			Id:        "b2",
			RoleId:    "r2",
			Namespace: "n2",
			Subjects: []*storage.Subject{ // give c1:sa1:n1 write access in n2
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa1",
					Namespace: "n2",
				},
			},
		},
		{
			Id:          "b3",
			RoleId:      "r3",
			Namespace:   "n1",
			ClusterRole: true,
			Subjects: []*storage.Subject{ // give c1:sa1:n1 cluster admin
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa1",
					Namespace: "n1",
				},
			},
		},
	}
	bindingsC2 := []*storage.K8SRoleBinding{
		{
			Id:        "b4",
			RoleId:    "r4",
			Namespace: "n1",
		},
		{
			Id:        "b5",
			RoleId:    "r5",
			Namespace: "n1",
			Subjects: []*storage.Subject{ // give c2:sa1:n1 write access in the same namespace
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa1",
					Namespace: "n1",
				},
			},
		},
	}

	// Test service accounts have more permissions than NONE.
	suite.mockClusters.EXPECT().GetClusters(rbacReadingCtx).Return(clusters, nil)

	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster1).Return(serviceAccountsC1, nil)
	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster2).Return(serviceAccountsC2, nil)

	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster1).Return(rolesC1, nil)
	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster2).Return(rolesC2, nil)

	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster1).Return(bindingsC1, nil)
	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster2).Return(bindingsC2, nil)

	fields := &storage.PolicyFields{
		PermissionPolicy: &storage.PermissionPolicy{
			PermissionLevel: storage.PermissionLevel_NONE,
		},
	}
	outputQuery, _, err := suite.tested.Query(fields, nil)
	suite.NoError(err, "")
	suite.Equal(search.NewDisjunctionQuery(
		search.NewConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
			search.NewDisjunctionQuery(
				search.NewConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
				),
				search.NewConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa2").ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
				),
				search.NewConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, "n2").ProtoQuery(),
				),
			),
		),
		search.NewConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c2").ProtoQuery(),
			search.NewConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
			),
		),
	), outputQuery, "query didn't match expectation")

	// Test service accounts have more permissions than DEFAULT.
	suite.mockClusters.EXPECT().GetClusters(rbacReadingCtx).Return(clusters, nil)

	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster1).Return(serviceAccountsC1, nil)
	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster2).Return(serviceAccountsC2, nil)

	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster1).Return(rolesC1, nil)
	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster2).Return(rolesC2, nil)

	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster1).Return(bindingsC1, nil)
	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster2).Return(bindingsC2, nil)

	fields = &storage.PolicyFields{
		PermissionPolicy: &storage.PermissionPolicy{
			PermissionLevel: storage.PermissionLevel_DEFAULT,
		},
	}
	outputQuery, _, err = suite.tested.Query(fields, nil)
	suite.NoError(err, "")
	suite.Equal(search.NewDisjunctionQuery(
		search.NewConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
			search.NewDisjunctionQuery(
				search.NewConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
				),
				search.NewConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, "n2").ProtoQuery(),
				),
			),
		),
		search.NewConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c2").ProtoQuery(),
			search.NewConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
			),
		),
	), outputQuery, "query didn't match expectation")

	// Test service accounts have more permissions than ELEVATED_IN_NAMESPACE.
	suite.mockClusters.EXPECT().GetClusters(rbacReadingCtx).Return(clusters, nil)

	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster1).Return(serviceAccountsC1, nil)
	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster2).Return(serviceAccountsC2, nil)

	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster1).Return(rolesC1, nil)
	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster2).Return(rolesC2, nil)

	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster1).Return(bindingsC1, nil)
	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster2).Return(bindingsC2, nil)

	fields = &storage.PolicyFields{
		PermissionPolicy: &storage.PermissionPolicy{
			PermissionLevel: storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
		},
	}
	outputQuery, _, err = suite.tested.Query(fields, nil)
	suite.NoError(err, "")
	suite.Equal(search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
		search.NewConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
			search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
		),
	), outputQuery, "query didn't match expectation")

	// Test service accounts have CLUSTER_ADMIN permissions.
	suite.mockClusters.EXPECT().GetClusters(rbacReadingCtx).Return(clusters, nil)

	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster1).Return(serviceAccountsC1, nil)
	suite.mockServiceAccounts.EXPECT().SearchRawServiceAccounts(rbacReadingCtx, isInCluster2).Return(serviceAccountsC2, nil)

	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster1).Return(rolesC1, nil)
	suite.mocksRoles.EXPECT().SearchRawRoles(rbacReadingCtx, isInCluster2).Return(rolesC2, nil)

	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster1).Return(bindingsC1, nil)
	suite.mockBindings.EXPECT().SearchRawRoleBindings(rbacReadingCtx, isInCluster2).Return(bindingsC2, nil)

	fields = &storage.PolicyFields{
		PermissionPolicy: &storage.PermissionPolicy{
			PermissionLevel: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE,
		},
	}
	outputQuery, _, err = suite.tested.Query(fields, nil)
	suite.NoError(err, "")
	suite.Equal(search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
		search.NewConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "sa1").ProtoQuery(),
			search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery(),
		),
	), outputQuery, "query didn't match expectation")
}

func (suite *PipelineTestSuite) TestServiceAccountBucketing() {
	roles := []*storage.K8SRole{
		{
			Id:        "r1",
			ClusterId: "c1",
			Namespace: "n1",
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
		{
			Id:        "r2",
			ClusterId: "c1",
			Namespace: "n2",
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"create"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
		{
			Id:          "r3",
			ClusterId:   "c1",
			ClusterRole: true,
			Rules: []*storage.PolicyRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					ApiGroups: []string{""},
				},
			},
		},
	}

	bindings := []*storage.K8SRoleBinding{
		{
			Id:        "b1",
			RoleId:    "r1",
			Namespace: "n1",
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa1",
					Namespace: "n1",
				},
			},
		},
		{
			Id:        "b2",
			RoleId:    "r2",
			Namespace: "n2",
			Subjects: []*storage.Subject{ // give c1:sa1:n1 write access in n2
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa2",
					Namespace: "n1",
				},
			},
		},
		{
			Id:          "b3",
			RoleId:      "r3",
			Namespace:   "n1",
			ClusterRole: true,
			Subjects: []*storage.Subject{ // give c1:sa1:n1 cluster admin
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "sa3",
					Namespace: "n1",
				},
			},
		},
	}

	bucketEval := newBucketEvaluator(roles, bindings)
	suite.Equal(storage.PermissionLevel_DEFAULT, bucketEval.getBucket(&storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      "sa1",
		Namespace: "n1",
	}))
	suite.Equal(storage.PermissionLevel_ELEVATED_IN_NAMESPACE, bucketEval.getBucket(&storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      "sa2",
		Namespace: "n1",
	}))
	suite.Equal(storage.PermissionLevel_CLUSTER_ADMIN, bucketEval.getBucket(&storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      "sa3",
		Namespace: "n1",
	}))
}
