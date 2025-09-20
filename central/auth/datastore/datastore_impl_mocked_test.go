package datastore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/central/auth/store"
	mockAuthStore "github.com/stackrox/rox/central/auth/store/mocks"
	mockRoleDataStore "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testRole1        = "New-Admin"
	testRole2        = "Super-Admin"
	testRole3        = "Super Continuous Integration"
	configController = "Configuration Controller"
	testIssuer       = m2m.KubernetesTokenIssuer

	missingRoleName = "missing role"
)

var (
	testRoles = set.NewFrozenStringSet(testRole1, testRole2, testRole3, configController)

	declarativeTraits = &storage.Traits{Origin: storage.Traits_DECLARATIVE}
	imperativeTraits  = &storage.Traits{Origin: storage.Traits_IMPERATIVE}
)

func TestAuthDatastoreMocked(t *testing.T) {
	suite.Run(t, new(datastoreMockedTestSuite))
}

type datastoreMockedTestSuite struct {
	suite.Suite

	ctx            context.Context
	declarativeCtx context.Context

	mockCtrl *gomock.Controller
	mockSet  *mocks.MockTokenExchangerSet

	authStore     store.Store
	authDataStore DataStore
	roleDataStore *mockRoleDataStore.MockDataStore
}

func (s *datastoreMockedTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access),
		),
	)

	s.declarativeCtx = declarativeconfig.WithModifyDeclarativeResource(s.ctx)

	s.mockCtrl = gomock.NewController(s.T())

	s.roleDataStore = mockRoleDataStore.NewMockDataStore(s.mockCtrl)

	s.mockSet = mocks.NewMockTokenExchangerSet(s.mockCtrl)
	s.mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.mockSet.EXPECT().RemoveTokenExchanger(gomock.Any()).Return(nil).AnyTimes()
	s.mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, true).AnyTimes()
	s.mockSet.EXPECT().RollbackExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	s.authStore = mockAuthStore.NewMockStore(s.mockCtrl)
	s.authDataStore = New(s.authStore, s.roleDataStore, s.mockSet)
}

func (s *datastoreMockedTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *datastoreMockedTestSuite) TestVerifyConfigRoleExists() {

	testStoreError := errors.New("test store error")

	const existingDeclarativeRole1 = "existing declarative role 1"

	declarativeRole1 := &storage.Role{
		Name:            existingDeclarativeRole1,
		AccessScopeId:   uuid.NewTestUUID(1).String(),
		PermissionSetId: uuid.NewTestUUID(2).String(),
		Traits:          declarativeTraits.CloneVT(),
	}

	const existingDeclarativeRole2 = "existing declarative role 2"

	declarativeRole2 := &storage.Role{
		Name:            existingDeclarativeRole2,
		AccessScopeId:   uuid.NewTestUUID(3).String(),
		PermissionSetId: uuid.NewTestUUID(4).String(),
		Traits:          declarativeTraits.CloneVT(),
	}

	const existingImperativeRole = "existing imperative role"

	imperativeRole := &storage.Role{
		Name:            existingImperativeRole,
		AccessScopeId:   uuid.NewTestUUID(5).String(),
		PermissionSetId: uuid.NewTestUUID(6).String(),
		Traits:          imperativeTraits.CloneVT(),
	}

	const missingRole = "missing role"
	const missingImperativeRole = "missing imperative role"
	const missingDeclarativeRole = "missing declarative role"

	const testID = "test ID"

	for name, tc := range map[string]struct {
		prepare           func()
		m2mConfig         *storage.AuthMachineToMachineConfig
		expectedError     error
		expectedErrorText string
	}{
		"machine to machine declarative config referencing existing role declarative role succeeds": {
			prepare: func() {
				s.roleDataStore.EXPECT().
					GetManyRoles(gomock.Any(), gomock.InAnyOrder([]string{existingDeclarativeRole1})).
					Times(1).
					Return([]*storage.Role{declarativeRole1}, nil, nil)
			},
			m2mConfig: getBasicM2mConfig(declarativeTraits, testID, m2m.KubernetesTokenIssuer, existingDeclarativeRole1),
		},
		"machine to machine declarative config referencing missing role declarative role triggers error": {
			prepare: func() {
				s.roleDataStore.EXPECT().
					GetManyRoles(gomock.Any(), gomock.InAnyOrder([]string{missingRole})).
					Times(1).
					Return(nil, []string{missingRoleName}, nil)
			},
			m2mConfig:     getBasicM2mConfig(declarativeTraits, testID, m2m.KubernetesTokenIssuer, missingRole),
			expectedError: errox.InvalidArgs,
		},
		"machine to machine declarative config referencing at least one missing role triggers error": {
			prepare: func() {
				s.roleDataStore.EXPECT().
					GetManyRoles(
						gomock.Any(),
						gomock.InAnyOrder([]string{
							existingDeclarativeRole1,
							existingDeclarativeRole2,
							missingImperativeRole,
						}),
					).
					Times(1).
					Return([]*storage.Role{declarativeRole1, declarativeRole2}, []string{missingImperativeRole}, nil)
			},
			m2mConfig: getBasicM2mConfig(
				declarativeTraits,
				testID,
				m2m.KubernetesTokenIssuer,
				existingDeclarativeRole1,
				existingDeclarativeRole2,
				missingImperativeRole,
			),
			expectedError: errox.InvalidArgs,
		},
		"machine to machine declarative config referencing at least one missing role and one imperative role triggers error": {
			prepare: func() {
				s.roleDataStore.EXPECT().
					GetManyRoles(
						gomock.Any(),
						gomock.InAnyOrder([]string{
							existingDeclarativeRole1,
							missingDeclarativeRole,
							existingImperativeRole,
						}),
					).
					Times(1).
					Return([]*storage.Role{declarativeRole1, imperativeRole}, []string{missingDeclarativeRole}, nil)
			},
			m2mConfig: getBasicM2mConfig(
				declarativeTraits,
				testID,
				m2m.KubernetesTokenIssuer,
				existingDeclarativeRole1,
				missingDeclarativeRole,
				existingImperativeRole,
			),
			expectedError: errox.InvalidArgs,
			expectedErrorText: "imperative roles [existing imperative role] and missing roles [missing declarative role] can't be referenced by non-imperative " +
				"auth machine to machine configuration \"test ID\" for issuer \"https://kubernetes.default.svc\"",
		},
		"machine to machine declarative config referencing at least one imperative role triggers error": {
			prepare: func() {
				s.roleDataStore.EXPECT().
					GetManyRoles(
						gomock.Any(),
						gomock.InAnyOrder([]string{
							existingDeclarativeRole1,
							existingDeclarativeRole2,
							existingImperativeRole,
						}),
					).
					Times(1).
					Return([]*storage.Role{declarativeRole1, declarativeRole2, imperativeRole}, nil, nil)
			},
			m2mConfig: getBasicM2mConfig(
				declarativeTraits,
				testID,
				m2m.KubernetesTokenIssuer,
				existingDeclarativeRole1,
				existingDeclarativeRole2,
				existingImperativeRole,
			),
			expectedError: errox.InvalidArgs,
		},
		"store error is propagated": {
			prepare: func() {
				s.roleDataStore.EXPECT().
					GetManyRoles(gomock.Any(), gomock.InAnyOrder([]string{existingDeclarativeRole1})).
					Times(1).
					Return(nil, nil, testStoreError)
			},
			m2mConfig:     getBasicM2mConfig(declarativeTraits, testID, m2m.KubernetesTokenIssuer, existingDeclarativeRole1),
			expectedError: testStoreError,
		},
	} {
		s.Run(name, func() {
			authDataStore, ok := s.authDataStore.(*datastoreImpl)
			s.Require().True(ok)
			tc.prepare()
			verifyErr := authDataStore.verifyReferencedConfigRoles(s.declarativeCtx, tc.m2mConfig)
			s.ErrorIs(verifyErr, tc.expectedError)
			if tc.expectedErrorText != "" {
				s.ErrorContains(verifyErr, tc.expectedErrorText)
			}
		})
	}
}

func getBasicM2mConfig(traits *storage.Traits, id string, issuer string, targetRoles ...string) *storage.AuthMachineToMachineConfig {
	m2mConfig := &storage.AuthMachineToMachineConfig{
		Id:                      id,
		Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		TokenExpirationDuration: "20m",
		Issuer:                  issuer,
		Traits:                  traits.CloneVT(),
	}
	mappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0, len(targetRoles))
	for ix, role := range targetRoles {
		mappings = append(mappings, &storage.AuthMachineToMachineConfig_Mapping{
			Key:             "sub",
			ValueExpression: fmt.Sprintf("system:serviceaccount:stackrox:config-controller%d", ix),
			Role:            role,
		})
	}
	m2mConfig.Mappings = mappings
	return m2mConfig
}
