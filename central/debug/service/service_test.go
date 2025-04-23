package service

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	configMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deleRegMocks "github.com/stackrox/rox/central/delegatedregistryconfig/datastore/mocks"
	"github.com/stackrox/rox/central/globaldb"
	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	logMocks "github.com/stackrox/rox/central/logimbue/store/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	pkgmocks "github.com/stackrox/rox/pkg/mocks/github.com/jackc/pgx/v5/mocks"
	"github.com/stackrox/rox/pkg/postgres/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestDebugService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(debugServiceTestSuite))
}

type debugServiceTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	noneCtx  context.Context

	groupsMock        *groupMocks.MockDataStore
	rolesMock         *roleMocks.MockDataStore
	notifiersMock     *notifierMocks.MockDataStore
	configMock        *configMocks.MockDataStore
	deleRegConfigMock *deleRegMocks.MockDataStore
	service           *serviceImpl
}

func (s *debugServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.noneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.groupsMock = groupMocks.NewMockDataStore(s.mockCtrl)
	s.rolesMock = roleMocks.NewMockDataStore(s.mockCtrl)
	s.notifiersMock = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.configMock = configMocks.NewMockDataStore(s.mockCtrl)
	s.deleRegConfigMock = deleRegMocks.NewMockDataStore(s.mockCtrl)

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
		deleRegConfigDS:      s.deleRegConfigMock,
	}
}

func (s *debugServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *debugServiceTestSuite) TestGetGroups() {
	s.groupsMock.EXPECT().ForEach(gomock.Any(), gomock.Any()).Return(errors.New("Test"))
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
	s.groupsMock.EXPECT().ForEach(gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, fn func(group *storage.Group) error) error {
			return fn(expectedGroups[0])
		})
	actualGroups, err := s.service.getGroups(s.noneCtx)
	ag, ok := actualGroups.([]*storage.Group)
	s.Require().True(ok)
	s.NoError(err)
	protoassert.SlicesEqual(s.T(), expectedGroups, ag)
}

func (s *debugServiceTestSuite) TestGetRoles() {
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
	actual := actualRoles.([]*diagResolvedRole)
	for i, e := range expectedRoles {
		a := actual[i]
		protoassert.Equal(s.T(), e.Role, a.Role)
		protoassert.Equal(s.T(), e.AccessScope, a.AccessScope)
		s.Equal(e.PermissionSet, a.PermissionSet)
	}
}

func (s *debugServiceTestSuite) TestGetNotifiers() {
	s.notifiersMock.EXPECT().ForEachScrubbedNotifier(gomock.Any(), gomock.Any()).Return(errors.New("Test"))
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
	s.notifiersMock.EXPECT().ForEachScrubbedNotifier(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Notifier) error) error {
			return fn(expectedNotifiers[0])
		},
	)
	actualNotifiers, err := s.service.getNotifiers(s.noneCtx)

	s.NoError(err)
	protoassert.SlicesEqual(s.T(), expectedNotifiers, actualNotifiers.([]*storage.Notifier))
}

func (s *debugServiceTestSuite) TestGetConfig() {
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
	ac, ok := actualConfig.(*storage.Config)
	s.Require().True(ok)

	s.NoError(err)
	protoassert.Equal(s.T(), expectedConfig, ac)
}

func (s *debugServiceTestSuite) TestGetBundle() {
	stubTime := time.Date(2023, 03, 14, 0, 0, 0, 0, time.UTC)
	now = func() time.Time {
		return stubTime
	}

	w := httptest.NewRecorder()
	testutils.SetVersion(s.T(), testutils.GetExampleVersion(s.T()))
	db := mocks.NewMockDB(s.mockCtrl)
	pgxRows := pkgmocks.NewRows([]string{"server_version"}).AddRow("15.1").ToPgxRows()
	db.EXPECT().QueryRow(gomock.Any(), "SHOW server_version;").Return(pgxRows)
	globaldb.SetPostgresTest(s.T(), db)

	s.configMock.EXPECT().GetConfig(gomock.Any()).Return(&storage.Config{}, nil)
	s.deleRegConfigMock.EXPECT().GetConfig(gomock.Any()).Return(&storage.DelegatedRegistryConfig{}, true, nil)
	s.service.writeZippedDebugDump(context.Background(), w, "debug.zip", debugDumpOptions{
		logs:              0,
		telemetryMode:     noTelemetry,
		withCPUProfile:    false,
		withLogImbue:      false,
		withAccessControl: false,
		withNotifiers:     false,
		withCentral:       false,
		clusters:          nil,
		since:             time.Now(),
	})

	s.Equal(http.StatusOK, w.Code)

	body := w.Body.Bytes()
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	s.Require().NoError(err)

	s.Assert().Len(zipReader.File, 3)
	for _, zipFile := range zipReader.File {
		s.T().Log("Reading file:", zipFile.Name)
		s.Assert().Equal(stubTime, zipFile.Modified.UTC())
	}
}

func (s *debugServiceTestSuite) TestGetLogs() {
	logs := []*storage.LogImbue{
		{
			Log: []byte("not a json"),
		},
		{
			Log: []byte(`{"json": "object"}`),
		},
		{
			Log: []byte(`{"weirdstring" : "**&&^^%%$$"}`),
		},
		{
			Log: []byte(`[{},{"hey": "hehehehey"}]`),
		},
	}

	logStore := logMocks.NewMockStore(s.mockCtrl)
	logStore.EXPECT().Walk(gomock.Any(), gomock.Any()).Do(
		func(_ context.Context, fn func(*storage.LogImbue) error) {
			for _, l := range logs {
				err := fn(l)
				s.Require().NoError(err)
			}
		}).Return(nil)
	s.service.store = logStore

	buf := &bytes.Buffer{}
	writer := newZipWriter(buf)
	err := s.service.getLogImbue(context.Background(), writer)
	s.Require().NoError(err)

	s.Require().NoError(writer.Close())
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	s.Require().NoError(err)

	s.Assert().Len(zipReader.File, 1)
	file := zipReader.File[0]
	s.Assert().Equal(file.Name, "logimbue-data.json")
	open, err := file.Open()
	s.Require().NoError(err)
	all, err := io.ReadAll(open)
	s.Require().NoError(err)
	s.Assert().JSONEq(`[
	  {
		"encodingError":"json: error calling MarshalJSON for type json.RawMessage: invalid character 'o' in literal null (expecting 'u')",
		"raw":"not a json"
	  },
	  {
		"json":"object"
	  },
	  {
		"weirdstring":"**\u0026\u0026^^%%$$"
	  },
	  [
		{},
		{
		  "hey":"hehehehey"
		}
	  ]
	]`, string(all))
}

func (s *debugServiceTestSuite) TestGetLogsWhenThereAreNoLogs() {
	logStore := logMocks.NewMockStore(s.mockCtrl)
	logStore.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
	s.service.store = logStore

	buf := &bytes.Buffer{}
	writer := newZipWriter(buf)
	err := s.service.getLogImbue(context.Background(), writer)
	s.Require().NoError(err)

	s.Require().NoError(writer.Close())
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	s.Require().NoError(err)

	s.Assert().Len(zipReader.File, 1)
	file := zipReader.File[0]
	s.Assert().Equal(file.Name, "logimbue-data.json")
	open, err := file.Open()
	s.Require().NoError(err)
	all, err := io.ReadAll(open)
	s.Require().NoError(err)
	s.Assert().JSONEq(`[]`, string(all))
}

func (s *debugServiceTestSuite) TestGetLogsWhenThereAreWriteErrors() {
	logStore := logMocks.NewMockStore(s.mockCtrl)
	logStore.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(errors.New("some error"))
	s.service.store = logStore

	buf := &bytes.Buffer{}
	writer := newZipWriter(buf)
	err := s.service.getLogImbue(context.Background(), writer)
	s.Assert().Error(err)
	s.Require().NoError(writer.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	s.Require().NoError(err)

	s.Assert().Len(zipReader.File, 1)
	file := zipReader.File[0]
	s.Assert().Equal(file.Name, "logimbue-data.json")
	open, err := file.Open()
	s.Require().NoError(err)
	all, err := io.ReadAll(open)
	s.Require().NoError(err)
	s.Assert().Equal(`[`, string(all))
}
