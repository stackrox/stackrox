package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/config/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestConfigDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(configDataStoreTestSuite))
}

type configDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *configDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *configDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

var (
	sampleConfig = &storage.Config{
		PublicConfig: &storage.PublicConfig{
			LoginNotice: &storage.LoginNotice{
				Enabled: false,
				Text: "You step onto the road, and if you don't keep your feet, " +
					"there's no knowing where you might be swept off to.",
			},
			Header: &storage.BannerConfig{
				Enabled: false,
				Text: "Home is behind, the world ahead, and there " +
					"are many paths to tread through shadows to the edge of night, " +
					"until the stars are all alight.",
				Size_:           10,
				Color:           "0x88bbff",
				BackgroundColor: "0x0000ff",
			},
			Footer: &storage.BannerConfig{
				Enabled:         false,
				Text:            "All's well that ends better.",
				Size_:           10,
				Color:           "0x88bbff",
				BackgroundColor: "0x0000ff",
			},
			Telemetry: nil,
		},
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention:                      nil,
			ImageRetentionDurationDays:          7,
			ExpiredVulnReqRetentionDurationDays: 7,
			DecommissionedClusterRetention:      nil,
			ReportRetentionConfig:               nil,
			VulnerabilityExceptionConfig:        nil,
		},
	}
)

func (s *configDataStoreTestSuite) TestAllowsGetPublic() {
	getPublicConfigCache().Purge()
	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, true, nil).Times(1)

	publicCfg, err := s.dataStore.GetPublicConfig()
	s.NoError(err, "expected no error trying to read")
	s.NotNil(publicCfg)
}

func (s *configDataStoreTestSuite) TestEnforcesGetPrivate() {
	s.storage.EXPECT().Get(gomock.Any()).Times(0)

	privateConfigNone, err := s.dataStore.GetPrivateConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(privateConfigNone, "expected return value to be nil")
}

func (s *configDataStoreTestSuite) TestAllowsGetPrivate() {
	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, true, nil).Times(1)

	privateConfigRead, err := s.dataStore.GetPrivateConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(privateConfigRead)

	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, true, nil).Times(1)

	privateConfigWrite, err := s.dataStore.GetPrivateConfig(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(privateConfigWrite)

}

func (s *configDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().Get(gomock.Any()).Times(0)

	configForNone, err := s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(configForNone, "expected return value to be nil")
}

func (s *configDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, false, nil).Times(1)

	configForRead, err := s.dataStore.GetConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(configForRead)

	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, false, nil).Times(1)

	configForWrite, err := s.dataStore.GetConfig(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(configForWrite)
}

func (s *configDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertConfig(s.hasNoneCtx, &storage.Config{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertConfig(s.hasReadCtx, &storage.Config{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *configDataStoreTestSuite) TestAllowsUpdate() {
	getPublicConfigCache().Purge()

	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpsertConfig(s.hasWriteCtx, &storage.Config{})
	s.NoError(err, "expected no error trying to write with permissions")

	publicConfig, found := getPublicConfigCache().Get(publicConfigKey)
	s.True(found)
	s.Nil(publicConfig)

	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	newUpdateErr := s.dataStore.UpsertConfig(s.hasWriteCtx, sampleConfig)
	s.NoError(newUpdateErr, "expected no error trying to rewrite with permissions")

	updatedPublicConfig, updatedFound := getPublicConfigCache().Get(publicConfigKey)
	s.True(updatedFound)
	s.NotNil(updatedPublicConfig)
}
