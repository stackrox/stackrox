package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	configMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestDebugService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DebugServiceTestSuite))
}

type DebugServiceTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	noneCtx  context.Context

	groupsMock    *groupMocks.MockDataStore
	rolesMock     *roleMocks.MockDataStore
	notifiersMock *notifierMocks.MockDataStore
	configMock    *configMocks.MockDataStore

	service *serviceImpl
}

func (s *DebugServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.noneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.groupsMock = groupMocks.NewMockDataStore(s.mockCtrl)
	s.rolesMock = roleMocks.NewMockDataStore(s.mockCtrl)
	s.notifiersMock = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.configMock = configMocks.NewMockDataStore(s.mockCtrl)

	s.service = &serviceImpl{
		clusters:             nil,
		sensorConnMgr:        nil,
		telemetryGatherer:    nil,
		store:                nil,
		authzTraceSink:       nil,
		authProviderRegistry: nil,
		groupDataStore:       s.groupsMock,
		roleDataStore:        s.rolesMock,
		configDataStore:      s.configMock,
		notifierDataStore:    s.notifiersMock,
	}
}

func (s *DebugServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *DebugServiceTestSuite) TestGetGroups() {
	s.groupsMock.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("Test"))
	_, err := s.service.getGroups(s.noneCtx)
	s.Error(err, "expected error propagation")

	expectedGroups := []*storage.Group{
		{
			RoleName: "test",
			Props: &storage.GroupProperties{
				AuthProviderId: "1",
				Key:            "test",
				Value:          "1",
			},
		},
	}
	s.groupsMock.EXPECT().GetAll(gomock.Any()).Return(expectedGroups, nil)
	actualGroups, err := s.service.getGroups(s.noneCtx)

	s.NoError(err)
	s.Equal(expectedGroups, actualGroups)
}

func (s *DebugServiceTestSuite) TestGetRoles() {
	s.rolesMock.EXPECT().GetAllRoles(gomock.Any()).Return(nil, errors.New("Test"))
	_, err := s.service.getRoles(s.noneCtx)
	s.Error(err, "expected error propagation")

	allRoles := []*storage.Role{
		{
			Name: "Test",
		},
	}
	s.rolesMock.EXPECT().GetAllRoles(gomock.Any()).Return(allRoles, nil)

	resolvedRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	s.rolesMock.EXPECT().GetAndResolveRole(gomock.Any(), allRoles[0].Name).Return(resolvedRole, nil)
	resolvedRole.EXPECT().GetPermissions().Return(map[string]storage.Access{
		"TestNone":      0,
		"TestRead":      1,
		"TestReadWrite": 2,
	})
	expectedAccessScope := storage.SimpleAccessScope{
		Name: "TestScope",
	}
	resolvedRole.EXPECT().GetAccessScope().Return(&expectedAccessScope)
	actualRoles, err := s.service.getRoles(s.noneCtx)

	expectedRoles := []*diagResolvedRole{
		{
			Role: allRoles[0],
			PermissionSet: map[string]string{
				"TestNone":      storage.Access_NO_ACCESS.String(),
				"TestRead":      storage.Access_READ_ACCESS.String(),
				"TestReadWrite": storage.Access_READ_WRITE_ACCESS.String(),
			},
			AccessScope: &expectedAccessScope,
		},
	}

	s.NoError(err)
	s.EqualValues(expectedRoles, actualRoles)
}

func (s *DebugServiceTestSuite) TestGetNotifiers() {
	s.notifiersMock.EXPECT().GetScrubbedNotifiers(gomock.Any()).Return(nil, errors.New("Test"))
	_, err := s.service.getNotifiers(s.noneCtx)
	s.Error(err, "expected error propagation")

	expectedNotifiers := []*storage.Notifier{
		{
			Name: "test",
			Config: &storage.Notifier_Pagerduty{
				Pagerduty: &storage.PagerDuty{
					ApiKey: "******",
				},
			},
		},
	}
	s.notifiersMock.EXPECT().GetScrubbedNotifiers(gomock.Any()).Return(expectedNotifiers, nil)
	actualNotifiers, err := s.service.getNotifiers(s.noneCtx)

	s.NoError(err)
	s.EqualValues(expectedNotifiers, actualNotifiers)
}

func (s *DebugServiceTestSuite) TestGetConfig() {
	s.configMock.EXPECT().GetConfig(gomock.Any()).Return(nil, errors.New("Test"))
	_, err := s.service.getConfig(s.noneCtx)
	s.Error(err, "expected error propagation")

	expectedConfig := &storage.Config{
		PublicConfig: &storage.PublicConfig{
			LoginNotice: &storage.LoginNotice{
				Text: "test",
			},
		},
		PrivateConfig: &storage.PrivateConfig{
			ImageRetentionDurationDays: 1,
		},
	}
	s.configMock.EXPECT().GetConfig(gomock.Any()).Return(expectedConfig, nil)
	actualConfig, err := s.service.getConfig(s.noneCtx)

	s.NoError(err)
	s.Equal(expectedConfig, actualConfig)
}
