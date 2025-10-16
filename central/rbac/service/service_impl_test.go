package service

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	bindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestRbacService(t *testing.T) {
	suite.Run(t, new(RbacServiceTestSuite))
}

type RbacServiceTestSuite struct {
	suite.Suite

	mockRoleStore     *roleMocks.MockDataStore
	mockBindingsStore *bindingMocks.MockDataStore

	service Service

	mockCtrl *gomock.Controller
}

func (suite *RbacServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockRoleStore = roleMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockBindingsStore = bindingMocks.NewMockDataStore(suite.mockCtrl)

	suite.service = New(suite.mockRoleStore, suite.mockBindingsStore)
}

// Test happy path for getting roles
func (suite *RbacServiceTestSuite) TestGetRole() {
	roleID := "id1"

	expectedRole := &storage.K8SRole{}
	expectedRole.SetId(roleID)
	expectedRole.SetName("rolename")
	expectedRole.SetClusterId("cluster")
	expectedRole.SetNamespace("namespace")
	suite.mockRoleStore.EXPECT().GetRole(gomock.Any(), roleID).Return(expectedRole, true, nil)

	rbid := &v1.ResourceByID{}
	rbid.SetId(roleID)
	response, err := suite.service.GetRole((context.Context)(nil), rbid)
	suite.NoError(err)
	protoassert.Equal(suite.T(), response.GetRole(), expectedRole)
}

// Test that when we fail to find a k8s role, an error is returned.
func (suite *RbacServiceTestSuite) TestGetRolesWithStoreRoleNotExists() {
	roleID := "id1"

	suite.mockRoleStore.EXPECT().GetRole(gomock.Any(), roleID).Return((*storage.K8SRole)(nil), false, nil)

	rbid := &v1.ResourceByID{}
	rbid.SetId(roleID)
	_, err := suite.service.GetRole((context.Context)(nil), rbid)
	suite.Error(err)
}

// Test that when we fail to read the db for a k8s role, an error is returned.
func (suite *RbacServiceTestSuite) TestGetSecretsWithStoreSecretFailure() {
	secretID := "id1"

	expectedErr := errors.New("failure")
	suite.mockRoleStore.EXPECT().GetRole(gomock.Any(), secretID).Return((*storage.K8SRole)(nil), true, expectedErr)

	rbid := &v1.ResourceByID{}
	rbid.SetId(secretID)
	_, actualErr := suite.service.GetRole((context.Context)(nil), rbid)
	suite.Error(actualErr)
}

// Test happy path for searching k8s role
func (suite *RbacServiceTestSuite) TestSearchRole() {
	k8SRole := &storage.K8SRole{}
	k8SRole.SetId("id1")
	expectedReturns := []*storage.K8SRole{
		k8SRole,
	}

	suite.mockRoleStore.EXPECT().SearchRawRoles(gomock.Any(), gomock.Any()).Return(expectedReturns, nil)

	_, err := suite.service.ListRoles((context.Context)(nil), &v1.RawQuery{})
	suite.NoError(err)
}

// Test that when searching fails, that error is returned.
func (suite *RbacServiceTestSuite) TestSearchRoleFailure() {
	expectedError := errors.New("failure")

	suite.mockRoleStore.EXPECT().SearchRawRoles(gomock.Any(), gomock.Any()).Return(([]*storage.K8SRole)(nil), expectedError)

	_, actualErr := suite.service.ListRoles((context.Context)(nil), &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))
}

// Test happy path for getting role bindings
func (suite *RbacServiceTestSuite) TestGetRoleBinding() {
	bindingID := "id1"

	expectedRoleBinding := &storage.K8SRoleBinding{}
	expectedRoleBinding.SetId(bindingID)
	expectedRoleBinding.SetName("bindingName")
	expectedRoleBinding.SetClusterId("cluster")
	expectedRoleBinding.SetNamespace("namespace")
	suite.mockBindingsStore.EXPECT().GetRoleBinding(gomock.Any(), bindingID).Return(expectedRoleBinding, true, nil)

	rbid := &v1.ResourceByID{}
	rbid.SetId(bindingID)
	response, err := suite.service.GetRoleBinding((context.Context)(nil), rbid)
	suite.NoError(err)
	protoassert.Equal(suite.T(), expectedRoleBinding, response.GetBinding())
}

// Test that when we fail to find a k8s role binding, an error is returned.
func (suite *RbacServiceTestSuite) TestGetRoleBindingsNotExists() {
	bindingID := "id1"

	suite.mockBindingsStore.EXPECT().GetRoleBinding(gomock.Any(), bindingID).Return((*storage.K8SRoleBinding)(nil), false, nil)

	rbid := &v1.ResourceByID{}
	rbid.SetId(bindingID)
	_, err := suite.service.GetRoleBinding((context.Context)(nil), rbid)
	suite.Error(err)
}

// Test that when we fail to read the db for a k8s role binding, an error is returned.
func (suite *RbacServiceTestSuite) TestGetRoleBindingFailure() {
	bindingID := "id1"

	expectedErr := errors.New("failure")
	suite.mockBindingsStore.EXPECT().GetRoleBinding(gomock.Any(), bindingID).Return((*storage.K8SRoleBinding)(nil), true, expectedErr)

	rbid := &v1.ResourceByID{}
	rbid.SetId(bindingID)
	_, actualErr := suite.service.GetRoleBinding((context.Context)(nil), rbid)
	suite.Error(actualErr)
}

// Test happy path for searching k8s role binding
func (suite *RbacServiceTestSuite) TestSearchRoleBinding() {
	k8srb := &storage.K8SRoleBinding{}
	k8srb.SetId("id1")
	expectedReturns := []*storage.K8SRoleBinding{
		k8srb,
	}

	suite.mockBindingsStore.EXPECT().SearchRawRoleBindings(gomock.Any(), gomock.Any()).Return(expectedReturns, nil)

	_, err := suite.service.ListRoleBindings((context.Context)(nil), &v1.RawQuery{})
	suite.NoError(err)
}

// Test that when searching fails, that error is returned.
func (suite *RbacServiceTestSuite) TestSearchRoleBindingFailure() {
	expectedError := errors.New("failure")

	suite.mockBindingsStore.EXPECT().SearchRawRoleBindings(gomock.Any(), gomock.Any()).Return(([]*storage.K8SRoleBinding)(nil), expectedError)

	_, actualErr := suite.service.ListRoleBindings((context.Context)(nil), &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))
}
