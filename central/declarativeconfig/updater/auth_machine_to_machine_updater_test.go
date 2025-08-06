//go:build sql_integration

package updater

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/mock/gomock"

	m2mDataStore "github.com/stackrox/rox/central/auth/datastore"
	m2mDataStoreMocks "github.com/stackrox/rox/central/auth/datastore/mocks"
	m2mStore "github.com/stackrox/rox/central/auth/store"
	healthDataStore "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	roleDataStoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

// region Postgres tests

func TestAuthMachineToMachineUpdater(t *testing.T) {
	suite.Run(t, new(authMachineToMachineTestSuite))
}

type authMachineToMachineTestSuite struct {
	suite.Suite

	ctx         context.Context
	db          *pgtest.TestPostgres
	updater     ResourceUpdater
	m2mConfigDS m2mDataStore.DataStore
	roleDS      roleDataStore.DataStore

	mockCtrl *gomock.Controller
}

func (s *authMachineToMachineTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.mockCtrl = gomock.NewController(s.T())
	mockSet := mocks.NewMockTokenExchangerSet(s.mockCtrl)
	mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSet.EXPECT().RemoveTokenExchanger(gomock.Any()).Return(nil).AnyTimes()
	mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, true).AnyTimes()
	mockSet.EXPECT().RollbackExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	s.db = pgtest.ForT(s.T())
	s.roleDS = roleDataStore.GetTestPostgresDataStore(s.T(), s.db)
	m2mStorage := m2mStore.New(s.db)
	s.m2mConfigDS = m2mDataStore.New(m2mStorage, s.roleDS, mockSet, nil)
	healthDS := healthDataStore.GetTestPostgresDataStore(s.T(), s.db)
	s.updater = newAuthM2MConfigUpdater(s.m2mConfigDS, healthDS)
}

func (s *authMachineToMachineTestSuite) TearDownTest() {
	s.db.Close()
	s.mockCtrl.Finish()
}

func (s *authMachineToMachineTestSuite) TestUpsert() {
	for name, tc := range map[string]struct {
		testInit      func()
		inputMessage  protocompat.Message
		expectedError error
	}{
		"invalid message type should yield an error": {
			inputMessage: &storage.SimpleAccessScope{
				Id: "some-object-of-wrong-type",
			},
			expectedError: errox.InvariantViolation,
		},
		"valid message should be upserted": {
			testInit: func() {
				err := s.roleDS.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
					Id:     uuid.NewTestUUID(1).String(),
					Name:   "Test Access Scope",
					Rules:  &storage.SimpleAccessScope_Rules{},
					Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				})
				s.Require().NoError(err)
				err = s.roleDS.AddPermissionSet(s.ctx, &storage.PermissionSet{
					Id:     uuid.NewTestUUID(2).String(),
					Name:   "Test Permission Set",
					Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				})
				s.Require().NoError(err)
				err = s.roleDS.AddRole(s.ctx, &storage.Role{
					Name:            "Test Role",
					PermissionSetId: uuid.NewTestUUID(2).String(),
					AccessScopeId:   uuid.NewTestUUID(1).String(),
					Traits:          &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				})
				s.Require().NoError(err)
			},
			inputMessage: &storage.AuthMachineToMachineConfig{
				Id:                      uuid.NewTestUUID(3).String(),
				Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
				TokenExpirationDuration: "20m",
				Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "sub",
						ValueExpression: "system:serviceaccount:stackrox:config-controller",
						Role:            "Test Role",
					},
				},
				Issuer: "https://kubernetes.default.svc",
			},
		},
	} {
		s.Run(name, func() {
			if tc.testInit != nil {
				tc.testInit()
			}
			err := s.updater.Upsert(s.ctx, tc.inputMessage)
			s.ErrorIs(err, tc.expectedError)
			if tc.expectedError == nil {
				m2mConfig, ok := tc.inputMessage.(*storage.AuthMachineToMachineConfig)
				s.Require().True(ok)
				fetched, exists, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, m2mConfig.GetId())
				s.NoError(err)
				s.True(exists)
				protoassert.Equal(s.T(), m2mConfig, fetched)
			}
		})
	}
}

func (s *authMachineToMachineTestSuite) TestDelete_Success() {
	s.Require().NoError(s.roleDS.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
		Id:     uuid.NewTestUUID(1).String(),
		Name:   "Test Access Scope",
		Rules:  &storage.SimpleAccessScope_Rules{},
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.roleDS.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:     uuid.NewTestUUID(2).String(),
		Name:   "Test Permission Set",
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.roleDS.AddRole(s.ctx, &storage.Role{
		Name:            "Test Role",
		PermissionSetId: uuid.NewTestUUID(2).String(),
		AccessScopeId:   uuid.NewTestUUID(1).String(),
		Traits:          &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	m2mConfig := &storage.AuthMachineToMachineConfig{
		Id:                      uuid.NewTestUUID(3).String(),
		Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		TokenExpirationDuration: "30m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "system:serviceaccount:stackrox:config-controller",
				Role:            "Test Role",
			},
		},
		Issuer: "https://kubernetes.default.svc",
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}
	_, err := s.m2mConfigDS.UpsertAuthM2MConfig(s.ctx, m2mConfig)
	s.Require().NoError(err)

	imperativeM2MConfig := &storage.AuthMachineToMachineConfig{
		Id:                      uuid.NewTestUUID(4).String(),
		Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		TokenExpirationDuration: "30m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "system:serviceaccount:stackrox:config-controller",
				Role:            "Test Role",
			},
		},
		Issuer: "https://central.stackrox.svc",
		Traits: &storage.Traits{Origin: storage.Traits_IMPERATIVE},
	}
	_, err = s.m2mConfigDS.UpsertAuthM2MConfig(s.ctx, imperativeM2MConfig)
	s.Require().NoError(err)

	// Deletion run keeping the m2m config
	deleteFailedIDs, deleteErr := s.updater.DeleteResources(s.ctx, m2mConfig.GetId())
	s.NoError(deleteErr)
	s.Empty(deleteFailedIDs)

	fetched1, found1, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, m2mConfig.GetId())
	s.NoError(err)
	s.True(found1)
	protoassert.Equal(s.T(), m2mConfig, fetched1)

	fetched2, found2, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, imperativeM2MConfig.GetId())
	s.NoError(err)
	s.True(found2)
	protoassert.Equal(s.T(), imperativeM2MConfig, fetched2)

	// Declaratively remove the m2m config
	deleteAllFailedIDs, deleteAllErr := s.updater.DeleteResources(s.ctx)
	s.NoError(deleteAllErr)
	s.Empty(deleteAllFailedIDs)

	fetched3, found3, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, m2mConfig.GetId())
	s.NoError(err)
	s.False(found3)
	s.Nil(fetched3)

	fetched4, found4, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, imperativeM2MConfig.GetId())
	s.NoError(err)
	s.True(found4)
	protoassert.Equal(s.T(), imperativeM2MConfig, fetched4)
}

// endregion Postgres tests

// region Mocked tests

func TestMockedAuthMachineToMachineUpdater(t *testing.T) {
	suite.Run(t, new(authMachineToMachineMockedTestSuite))
}

type authMachineToMachineMockedTestSuite struct {
	suite.Suite

	ctx     context.Context
	db      *pgtest.TestPostgres
	updater ResourceUpdater

	m2mConfigDS *m2mDataStoreMocks.MockDataStore
	roleDS      *roleDataStoreMocks.MockDataStore

	mockCtrl *gomock.Controller
}

func (s *authMachineToMachineMockedTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.mockCtrl = gomock.NewController(s.T())
	mockSet := mocks.NewMockTokenExchangerSet(s.mockCtrl)
	mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSet.EXPECT().RemoveTokenExchanger(gomock.Any()).Return(nil).AnyTimes()
	mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, true).AnyTimes()
	mockSet.EXPECT().RollbackExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	s.roleDS = roleDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.m2mConfigDS = m2mDataStoreMocks.NewMockDataStore(s.mockCtrl)

	s.db = pgtest.ForT(s.T())
	healthDS := healthDataStore.GetTestPostgresDataStore(s.T(), s.db)
	s.updater = newAuthM2MConfigUpdater(s.m2mConfigDS, healthDS)
}

func (s *authMachineToMachineMockedTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
	s.db.Close()
}

func (s *authMachineToMachineMockedTestSuite) TestDeleteResources_Error() {
	imperativeTraits := &storage.Traits{Origin: storage.Traits_IMPERATIVE}
	declarativeTraits := &storage.Traits{Origin: storage.Traits_DECLARATIVE}

	const (
		defaultK8sIssuer      = "https://kubernetes.default.svc"
		stackroxK8sIssuer     = "https://kubernetes.stackrox.svc"
		stackroxCentralIssuer = "https://central.stackrox.svc"

		imperativeRole   = "imperative role"
		declarativeRole1 = "declarative role 1"
		declarativeRole2 = "declarative role 2"
	)

	imperativeConfig := getBasicM2mConfig(imperativeTraits, defaultK8sIssuer, imperativeRole)
	declarativeConfig1 := getBasicM2mConfig(declarativeTraits, stackroxK8sIssuer, declarativeRole1)
	declarativeConfig2 := getBasicM2mConfig(declarativeTraits, stackroxCentralIssuer, declarativeRole2)

	orphanedDeclarativeConfig2 := declarativeConfig2.CloneVT()
	orphanedDeclarativeConfig2.Traits = &storage.Traits{Origin: storage.Traits_DECLARATIVE_ORPHANED}

	testM2MDataStoreError := errors.New("test m2m datastore error")
	for name, tc := range map[string]struct {
		prepare            func()
		expectedErrors     []error
		expectedFailureLen int
	}{
		"m2m datastore foreach error is propagated": {
			prepare: func() {
				s.m2mConfigDS.EXPECT().
					ForEachAuthM2MConfig(gomock.Any(), gomock.Any()).
					Times(1).
					Return(testM2MDataStoreError)
			},
			expectedErrors: []error{testM2MDataStoreError},
		},
		"m2m successful walk and successful delete result in success": {
			prepare: func() {
				s.m2mConfigDS.EXPECT().
					ForEachAuthM2MConfig(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, f func(config *storage.AuthMachineToMachineConfig) error) error {
						for _, m2mConfig := range []*storage.AuthMachineToMachineConfig{
							imperativeConfig,
							declarativeConfig1,
							declarativeConfig2,
						} {
							err := f(m2mConfig)
							if err != nil {
								return err
							}
						}
						return nil
					})
				s.m2mConfigDS.EXPECT().
					RemoveAuthM2MConfig(gomock.Any(), declarativeConfig1.GetId()).
					Times(1).
					Return(nil)
				s.m2mConfigDS.EXPECT().
					RemoveAuthM2MConfig(gomock.Any(), declarativeConfig2.GetId()).
					Times(1).
					Return(nil)
			},
		},
		"Orphaned deletion error causes the orphaned config to be re-upserted as such": {
			prepare: func() {
				s.m2mConfigDS.EXPECT().
					ForEachAuthM2MConfig(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, f func(config *storage.AuthMachineToMachineConfig) error) error {
						return f(declarativeConfig2)
					})
				s.m2mConfigDS.EXPECT().
					RemoveAuthM2MConfig(gomock.Any(), declarativeConfig2.GetId()).
					Times(1).
					Return(errox.ReferencedByAnotherObject)
				s.m2mConfigDS.EXPECT().
					UpsertAuthM2MConfig(gomock.Any(), protomock.GoMockMatcherEqualMessage(orphanedDeclarativeConfig2)).
					Times(1).
					Return(nil, nil)
			},
			expectedErrors:     []error{errox.ReferencedByAnotherObject},
			expectedFailureLen: 1,
		},
		"Orphaned deletion error causes re-upsert and propagates both orphaned and upsert errors": {
			prepare: func() {
				s.m2mConfigDS.EXPECT().
					ForEachAuthM2MConfig(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, f func(config *storage.AuthMachineToMachineConfig) error) error {
						return f(declarativeConfig2)
					})
				s.m2mConfigDS.EXPECT().
					RemoveAuthM2MConfig(gomock.Any(), declarativeConfig2.GetId()).
					Times(1).
					Return(errox.ReferencedByAnotherObject)
				s.m2mConfigDS.EXPECT().
					UpsertAuthM2MConfig(gomock.Any(), protomock.GoMockMatcherEqualMessage(orphanedDeclarativeConfig2)).
					Times(1).
					Return(nil, testM2MDataStoreError)
			},
			expectedErrors:     []error{errox.ReferencedByAnotherObject, testM2MDataStoreError},
			expectedFailureLen: 1,
		},
		"Other deletion error is propagated without further processing of the failed config": {
			prepare: func() {
				s.m2mConfigDS.EXPECT().
					ForEachAuthM2MConfig(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, f func(config *storage.AuthMachineToMachineConfig) error) error {
						return f(declarativeConfig2)
					})
				s.m2mConfigDS.EXPECT().
					RemoveAuthM2MConfig(gomock.Any(), declarativeConfig2.GetId()).
					Times(1).
					Return(testM2MDataStoreError)
			},
			expectedErrors:     []error{testM2MDataStoreError},
			expectedFailureLen: 1,
		},
	} {
		s.Run(name, func() {
			tc.prepare()
			deletionFailureIDs, deletionErr := s.updater.DeleteResources(s.ctx)
			if len(tc.expectedErrors) == 0 {
				s.Nil(deletionErr)
			}
			for _, err := range tc.expectedErrors {
				s.ErrorIs(deletionErr, err)
			}
			s.Len(deletionFailureIDs, tc.expectedFailureLen)
		})
	}
}

func getBasicM2mConfig(traits *storage.Traits, issuer string, targetRoles ...string) *storage.AuthMachineToMachineConfig {
	m2mConfig := &storage.AuthMachineToMachineConfig{
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

// endregion Mocked tests
