//go:build sql_integration

package updater

import (
	"context"
	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/mock/gomock"
	"testing"

	m2mDataStore "github.com/stackrox/rox/central/auth/datastore"
	m2mStore "github.com/stackrox/rox/central/auth/store"
	healthDataStore "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

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
					Id:    uuid.NewTestUUID(1).String(),
					Name:  "Test Access Scope",
					Rules: &storage.SimpleAccessScope_Rules{},
				})
				s.Require().NoError(err)
				err = s.roleDS.AddPermissionSet(s.ctx, &storage.PermissionSet{
					Id:   uuid.NewTestUUID(2).String(),
					Name: "Test Permission Set",
				})
				s.Require().NoError(err)
				err = s.roleDS.AddRole(s.ctx, &storage.Role{
					Name:            "Test Role",
					PermissionSetId: uuid.NewTestUUID(2).String(),
					AccessScopeId:   uuid.NewTestUUID(1).String(),
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
		Id:    uuid.NewTestUUID(1).String(),
		Name:  "Test Access Scope",
		Rules: &storage.SimpleAccessScope_Rules{},
	}))
	s.Require().NoError(s.roleDS.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:   uuid.NewTestUUID(2).String(),
		Name: "Test Permission Set",
	}))
	s.Require().NoError(s.roleDS.AddRole(s.ctx, &storage.Role{
		Name:            "Test Role",
		PermissionSetId: uuid.NewTestUUID(2).String(),
		AccessScopeId:   uuid.NewTestUUID(1).String(),
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

	// Deletion run keeping the m2m config
	deleteFailedIDs, deleteErr := s.updater.DeleteResources(s.ctx, m2mConfig.GetId())
	s.NoError(deleteErr)
	s.Empty(deleteFailedIDs)

	fetched1, found1, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, m2mConfig.GetId())
	s.NoError(err)
	s.True(found1)
	protoassert.Equal(s.T(), m2mConfig, fetched1)

	// Declaratively remove the m2m config
	deleteAllFailedIDs, deleteAllErr := s.updater.DeleteResources(s.ctx)
	s.NoError(deleteAllErr)
	s.Empty(deleteAllFailedIDs)

	fetched2, found2, err := s.m2mConfigDS.GetAuthM2MConfig(s.ctx, m2mConfig.GetId())
	s.NoError(err)
	s.False(found2)
	s.Nil(fetched2)
}
