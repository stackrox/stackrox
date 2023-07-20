package service

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	bindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	saMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	saID = "id1"

	expectedSA = &storage.ServiceAccount{
		Id:        saID,
		Name:      "serviceaccountname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}

	listDeployment = &storage.ListDeployment{
		Id:        "deploymentId",
		Name:      "deployment",
		ClusterId: "cluster",
		Namespace: "namespace",
	}

	role = &storage.K8SRole{
		Id:          "role1",
		Name:        "role1",
		ClusterId:   "cluster",
		Namespace:   "namespace",
		ClusterRole: false,
	}
	clusterRole = &storage.K8SRole{
		Id:          "role2",
		Name:        "role2",
		ClusterId:   "cluster",
		ClusterRole: true,
	}

	rolebinding = &storage.K8SRoleBinding{
		RoleId: "role1",
		Subjects: []*storage.Subject{
			{
				ClusterId: "cluster",
				Name:      "serviceaccountname",
				Namespace: "namespace",
				Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
			},
		},
		ClusterRole: false,
		Namespace:   "namespace",
		Id:          "binding1",
	}

	clusterRoleBinding = &storage.K8SRoleBinding{
		RoleId: "role2",
		Subjects: []*storage.Subject{
			{
				ClusterId: "cluster",
				Name:      "serviceaccountname",
				Namespace: "namespace",
				Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
			},
		},
		ClusterRole: true,
		Id:          "binding2",
	}

	namespaceMetadata = &storage.NamespaceMetadata{
		Name: "namespace",
	}
)

func TestServiceAccountService(t *testing.T) {
	suite.Run(t, new(ServiceAccountServiceTestSuite))
}

type ServiceAccountServiceTestSuite struct {
	suite.Suite

	mockServiceAccountStore *saMocks.MockDataStore
	mockDeploymentStore     *deploymentMocks.MockDataStore
	mockRoleStore           *roleMocks.MockDataStore
	mockBindingStore        *bindingMocks.MockDataStore
	mockNamespaceStore      *namespaceMocks.MockDataStore
	service                 Service

	mockCtrl *gomock.Controller

	ctx context.Context
}

func (suite *ServiceAccountServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockServiceAccountStore = saMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockDeploymentStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockRoleStore = roleMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockBindingStore = bindingMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockNamespaceStore = namespaceMocks.NewMockDataStore(suite.mockCtrl)

	suite.service = New(suite.mockServiceAccountStore, suite.mockBindingStore, suite.mockRoleStore,
		suite.mockDeploymentStore, suite.mockNamespaceStore)

	suite.ctx = sac.WithAllAccess(context.Background())
}

// Test happy path for getting service accounts
func (suite *ServiceAccountServiceTestSuite) TestGetServiceAccount() {

	suite.setupMocks()

	suite.mockServiceAccountStore.EXPECT().GetServiceAccount(gomock.Any(), saID).Return(expectedSA, true, nil)

	sa, err := suite.service.GetServiceAccount(suite.ctx, &v1.ResourceByID{Id: saID})
	suite.NoError(err)
	suite.Equal(expectedSA, sa.SaAndRole.ServiceAccount)
	suite.Equal(1, len(sa.SaAndRole.DeploymentRelationships))
	suite.Equal(listDeployment.GetName(), sa.SaAndRole.DeploymentRelationships[0].GetName())
	suite.Equal(1, len(sa.SaAndRole.ScopedRoles))
	suite.Equal(1, len(sa.SaAndRole.ClusterRoles))
	suite.Equal("namespace", sa.SaAndRole.ScopedRoles[0].Namespace)
}

// Test that when we fail to find a service account, an error is returned.
func (suite *ServiceAccountServiceTestSuite) TestGetSAWithStoreSANotExists() {
	saID := "id1"

	suite.mockServiceAccountStore.EXPECT().GetServiceAccount(gomock.Any(), saID).Return((*storage.ServiceAccount)(nil), false, nil)

	_, err := suite.service.GetServiceAccount(suite.ctx, &v1.ResourceByID{Id: saID})
	suite.Error(err)
}

// Test that when we fail to read the db for a secret, an error is returned.
func (suite *ServiceAccountServiceTestSuite) TestGetSAWithStoreSAFailure() {
	saID := "id1"

	expectedErr := errors.New("failure")
	suite.mockServiceAccountStore.EXPECT().GetServiceAccount(gomock.Any(), saID).Return((*storage.ServiceAccount)(nil), true, expectedErr)

	_, actualErr := suite.service.GetServiceAccount(suite.ctx, &v1.ResourceByID{Id: saID})
	suite.Error(actualErr)
}

// Test happy path for searching secrets and relationships
func (suite *ServiceAccountServiceTestSuite) TestSearchServiceAccount() {
	suite.setupMocks()

	suite.mockServiceAccountStore.EXPECT().SearchRawServiceAccounts(gomock.Any(), gomock.Any()).Return([]*storage.ServiceAccount{expectedSA}, nil)

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, expectedSA.ClusterId).
		AddExactMatches(search.Namespace, expectedSA.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, expectedSA.GetName()).ProtoQuery()

	suite.mockDeploymentStore.EXPECT().SearchListDeployments(gomock.Any(), q).AnyTimes().Return([]*storage.ListDeployment{listDeployment}, nil)

	_, err := suite.service.ListServiceAccounts(suite.ctx, &v1.RawQuery{})
	suite.NoError(err)
}

// Test that when searching fails, that error is returned.
func (suite *ServiceAccountServiceTestSuite) TestSearchServiceAccountFailure() {
	expectedError := errors.New("failure")

	suite.mockServiceAccountStore.EXPECT().SearchRawServiceAccounts(gomock.Any(), gomock.Any()).Return(nil, expectedError)

	_, actualErr := suite.service.ListServiceAccounts(suite.ctx, &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))
}

func (suite *ServiceAccountServiceTestSuite) setupMocks() {

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, expectedSA.ClusterId).
		AddExactMatches(search.Namespace, expectedSA.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, expectedSA.GetName()).ProtoQuery()

	suite.mockDeploymentStore.EXPECT().SearchListDeployments(gomock.Any(), q).Return([]*storage.ListDeployment{listDeployment}, nil)

	suite.mockRoleStore.EXPECT().GetRole(gomock.Any(), "role1").AnyTimes().Return(role, true, nil)
	suite.mockRoleStore.EXPECT().GetRole(gomock.Any(), "role2").AnyTimes().Return(clusterRole, true, nil)

	namespaceQ := search.NewQueryBuilder().AddExactMatches(search.ClusterID, "cluster").ProtoQuery()
	suite.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any(), namespaceQ).AnyTimes().
		Return([]*storage.NamespaceMetadata{namespaceMetadata}, nil)

	clusterScopeQuery := search.NewQueryBuilder().
		AddBoolsHighlighted(search.ClusterRole, true).
		AddStringsHighlighted(search.RoleID, search.WildcardString).
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "").
		AddExactMatches(search.SubjectName, expectedSA.Name).
		AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
		ProtoQuery()
	suite.mockBindingStore.EXPECT().Search(gomock.Any(), clusterScopeQuery).AnyTimes().
		Return([]search.Result{
			{
				ID: clusterRoleBinding.GetId(),
				Matches: map[string][]string{
					"k8srolebinding.role_id":      {clusterRoleBinding.GetRoleId()},
					"k8srolebinding.cluster_role": {strconv.FormatBool(clusterRoleBinding.GetClusterRole())},
				},
			},
		}, nil)

	namespaceScopeQuery := search.NewQueryBuilder().
		AddStringsHighlighted(search.RoleID, search.WildcardString).
		AddBoolsHighlighted(search.ClusterRole, true).
		AddBoolsHighlighted(search.ClusterRole, false).
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SubjectName, expectedSA.Name).
		AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
		ProtoQuery()
	suite.mockBindingStore.EXPECT().Search(gomock.Any(), namespaceScopeQuery).AnyTimes().
		Return([]search.Result{
			{
				ID: rolebinding.GetId(),
				Matches: map[string][]string{
					"k8srolebinding.role_id":      {rolebinding.GetRoleId()},
					"k8srolebinding.cluster_role": {strconv.FormatBool(rolebinding.GetClusterRole())},
				},
			},
		}, nil)

	suite.mockBindingStore.EXPECT().GetManyRoleBindings(gomock.Any(), []string{"binding1"}).AnyTimes().Return(
		[]*storage.K8SRoleBinding{rolebinding}, nil, nil)
	suite.mockBindingStore.EXPECT().GetManyRoleBindings(gomock.Any(), []string{"binding2"}).AnyTimes().Return(
		[]*storage.K8SRoleBinding{clusterRoleBinding}, nil, nil)

}
