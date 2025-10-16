package datastore

import (
	"context"
	"testing"

	datastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	storeMocks "github.com/stackrox/rox/central/config/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestConfigDataStore(t *testing.T) {
	suite.Run(t, new(configDataStoreTestSuite))
}

type configDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx        context.Context
	hasReadCtx        context.Context
	hasWriteCtx       context.Context
	hasVMRequestsRead context.Context

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
	s.hasVMRequestsRead = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VulnerabilityManagementRequests)))
	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *configDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

var (
	samplePublicConfig = &storage.PublicConfig{
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
			Size:            storage.BannerConfig_MEDIUM,
			Color:           "0x88bbff",
			BackgroundColor: "0x0000ff",
		},
		Footer: &storage.BannerConfig{
			Enabled:         false,
			Text:            "All's well that ends better.",
			Size:            storage.BannerConfig_SMALL,
			Color:           "0x88bbff",
			BackgroundColor: "0x0000ff",
		},
		Telemetry: nil,
	}

	sampleConfig = &storage.Config{
		PublicConfig: samplePublicConfig,
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention:                      nil,
			ImageRetentionDurationDays:          7,
			ExpiredVulnReqRetentionDurationDays: 7,
			DecommissionedClusterRetention:      nil,
			ReportRetentionConfig:               nil,
			VulnerabilityExceptionConfig:        &storage.VulnerabilityExceptionConfig{},
		},
	}

	samplePlatformConfig = &storage.PlatformComponentConfig{
		NeedsReevaluation: true,
		Rules: []*storage.PlatformComponentConfig_Rule{
			defaultPlatformConfigSystemRule,
			defaultPlatformConfigLayeredProductsRule,
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

func (s *configDataStoreTestSuite) TestEnforcesGetVulnerabilityExceptionConfig() {
	s.storage.EXPECT().Get(gomock.Any()).Times(0)

	vmExceptionConfig, err := s.dataStore.GetVulnerabilityExceptionConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(vmExceptionConfig, "expected return value to be nil")
}

func (s *configDataStoreTestSuite) TestAllowsGetVulnerabilityExceptionConfig() {
	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, true, nil).Times(1)

	vmExceptionConfig, err := s.dataStore.GetVulnerabilityExceptionConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(vmExceptionConfig)

	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, true, nil).Times(1)

	vmExceptionConfig, err = s.dataStore.GetVulnerabilityExceptionConfig(s.hasVMRequestsRead)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(vmExceptionConfig)

	s.storage.EXPECT().Get(gomock.Any()).Return(sampleConfig, true, nil).Times(1)

	vmExceptionConfig, err = s.dataStore.GetVulnerabilityExceptionConfig(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(vmExceptionConfig)
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

func (s *configDataStoreTestSuite) TestGetPlatformComponentConfig() {
	pcc := &storage.PlatformComponentConfig{}
	pcc.SetNeedsReevaluation(true)
	pcc.SetRules([]*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	config := &storage.Config{}
	config.SetPublicConfig(sampleConfig.GetPublicConfig())
	config.SetPrivateConfig(sampleConfig.GetPrivateConfig())
	config.SetPlatformComponentConfig(pcc)
	s.storage.EXPECT().Get(gomock.Any()).Return(config, true, nil).Times(1)

	platformConfig, _, err := s.dataStore.GetPlatformComponentConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(platformConfig)
	s.True(platformConfig.GetNeedsReevaluation())
	s.Equal(2, len(platformConfig.GetRules()))
}

func (s *configDataStoreTestSuite) TestUpsertPlatformComponentConfig() {
	// Test when no update is required
	pcc := &storage.PlatformComponentConfig{}
	pcc.SetNeedsReevaluation(false)
	pcc.SetRules([]*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	config2 := &storage.Config{}
	config2.SetPublicConfig(sampleConfig.GetPublicConfig())
	config2.SetPrivateConfig(sampleConfig.GetPrivateConfig())
	config2.SetPlatformComponentConfig(pcc)
	s.storage.EXPECT().Get(gomock.Any()).Return(config2, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	config, err := s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	s.NoError(err, "expected no error trying to upsert basic config")
	s.NotNil(config)
	s.False(config.GetNeedsReevaluation())
	s.Equal(2, len(config.GetRules()))

	// Test when a re-evaluation should be triggered
	pcc2 := &storage.PlatformComponentConfig{}
	pcc2.SetNeedsReevaluation(false)
	pcc2.SetRules([]*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	config3 := &storage.Config{}
	config3.SetPublicConfig(sampleConfig.GetPublicConfig())
	config3.SetPrivateConfig(sampleConfig.GetPrivateConfig())
	config3.SetPlatformComponentConfig(pcc2)
	s.storage.EXPECT().Get(gomock.Any()).Return(config3, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	prn := &storage.PlatformComponentConfig_Rule_NamespaceRule{}
	prn.SetRegex(".*")
	pr := &storage.PlatformComponentConfig_Rule{}
	pr.SetName("new rule")
	pr.SetNamespaceRule(prn)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		pr,
	})
	s.NoError(err, "expected no error when upserting new rule")
	s.NotNil(config)
	s.True(config.GetNeedsReevaluation())
	s.Equal(3, len(config.GetRules()))

	// Test updating a system rule a couple ways
	pcc3 := &storage.PlatformComponentConfig{}
	pcc3.SetNeedsReevaluation(false)
	pcc3.SetRules([]*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	config4 := &storage.Config{}
	config4.SetPublicConfig(sampleConfig.GetPublicConfig())
	config4.SetPrivateConfig(sampleConfig.GetPrivateConfig())
	config4.SetPlatformComponentConfig(pcc3)
	s.storage.EXPECT().Get(gomock.Any()).Return(config4, true, nil).Times(1)
	prn2 := &storage.PlatformComponentConfig_Rule_NamespaceRule{}
	prn2.SetRegex("not the system regex")
	pr2 := &storage.PlatformComponentConfig_Rule{}
	pr2.SetName("system rule")
	pr2.SetNamespaceRule(prn2)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		pr2,
	})
	s.Error(err, "expected an error when trying to override the system regex")
	s.Nil(config)

	pcc4 := &storage.PlatformComponentConfig{}
	pcc4.SetNeedsReevaluation(false)
	pcc4.SetRules([]*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	config5 := &storage.Config{}
	config5.SetPublicConfig(sampleConfig.GetPublicConfig())
	config5.SetPrivateConfig(sampleConfig.GetPrivateConfig())
	config5.SetPlatformComponentConfig(pcc4)
	s.storage.EXPECT().Get(gomock.Any()).Return(config5, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		defaultPlatformConfigSystemRule,
	})
	s.NoError(err, "expected no error trying to add duplicate system rule")
	s.NotNil(config)
	s.False(config.GetNeedsReevaluation())
	s.Equal(2, len(config.GetRules()))

	// Test duplicating the layered products rule
	pcc5 := &storage.PlatformComponentConfig{}
	pcc5.SetNeedsReevaluation(false)
	pcc5.SetRules([]*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	config6 := &storage.Config{}
	config6.SetPublicConfig(sampleConfig.GetPublicConfig())
	config6.SetPrivateConfig(sampleConfig.GetPrivateConfig())
	config6.SetPlatformComponentConfig(pcc5)
	s.storage.EXPECT().Get(gomock.Any()).Return(config6, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	prn3 := &storage.PlatformComponentConfig_Rule_NamespaceRule{}
	prn3.SetRegex(".*")
	pr3 := &storage.PlatformComponentConfig_Rule{}
	pr3.SetName(defaultPlatformConfigLayeredProductsRule.GetName())
	pr3.SetNamespaceRule(prn3)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		pr3,
	})
	s.NoError(err, "expected no error trying to add duplicate system rule")
	s.NotNil(config)
	s.Equal(2, len(config.GetRules()))
}

var (
	customAlertRetention = &storage.PrivateConfig_AlertConfig{
		AlertConfig: &storage.AlertRetentionConfig{
			ResolvedDeployRetentionDurationDays:   DefaultDeployAlertRetention + 1,
			DeletedRuntimeRetentionDurationDays:   DefaultDeletedRuntimeAlertRetention + 1,
			AllRuntimeRetentionDurationDays:       DefaultRuntimeAlertRetention + 1,
			AttemptedDeployRetentionDurationDays:  DefaultAttemptedDeployAlertRetention + 1,
			AttemptedRuntimeRetentionDurationDays: DefaultAttemptedRuntimeAlertRetention + 1,
		},
	}

	customPrivateConfig = &storage.PrivateConfig{
		AlertRetention:                      customAlertRetention,
		ImageRetentionDurationDays:          DefaultImageRetention + 1,
		ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
		DecommissionedClusterRetention:      customDecommissionedClusterRetention,
		ReportRetentionConfig:               customReportRetentionConfig,
		VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
		AdministrationEventsConfig:          customAdministrationEventsConfig,
	}

	customVulnerabilityDeferralConfig = storage.VulnerabilityExceptionConfig_builder{
		ExpiryOptions: storage.VulnerabilityExceptionConfig_ExpiryOptions_builder{
			DayOptions: []*storage.DayOption{
				storage.DayOption_builder{
					NumDays: 15,
					Enabled: true,
				}.Build(),
				storage.DayOption_builder{
					NumDays: 31,
					Enabled: true,
				}.Build(),
				storage.DayOption_builder{
					NumDays: 61,
					Enabled: true,
				}.Build(),
				storage.DayOption_builder{
					NumDays: 91,
					Enabled: true,
				}.Build(),
			},
			FixableCveOptions: storage.VulnerabilityExceptionConfig_FixableCVEOptions_builder{
				AllFixable: true,
				AnyFixable: false,
			}.Build(),
			CustomDate: false,
			Indefinite: false,
		}.Build(),
	}.Build()

	customDecommissionedClusterRetention = &storage.DecommissionedClusterRetentionConfig{
		RetentionDurationDays: DefaultDecommissionedClusterRetentionDays + 1,
	}

	customReportRetentionConfig = &storage.ReportRetentionConfig{
		HistoryRetentionDurationDays:           DefaultReportHistoryRetentionWindow + 1,
		DownloadableReportRetentionDays:        DefaultDownloadableReportRetentionDays + 1,
		DownloadableReportGlobalRetentionBytes: DefaultDownloadableReportGlobalRetentionBytes + 1,
	}

	customAdministrationEventsConfig = &storage.AdministrationEventsConfig{
		RetentionDurationDays: DefaultAdministrationEventsRetention + 1,
	}
)

func TestValidateConfigAndPopulateMissingDefaults(t *testing.T) {
	testCases := map[string]struct {
		enabledFlags   []string
		disabledFlags  []string
		initialConfig  *storage.Config
		upsertedConfig *storage.Config
	}{
		"No Update for fully set config": {
			initialConfig: storage.Config_builder{
				PublicConfig:            samplePublicConfig,
				PrivateConfig:           customPrivateConfig,
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
			upsertedConfig: nil,
		},
		"Missing private config gets fully configured when Features activated": {
			enabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: storage.Config_builder{
				PublicConfig:  samplePublicConfig,
				PrivateConfig: nil,
			}.Build(),
			upsertedConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(defaultAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
					DecommissionedClusterRetention:      defaultDecommissionedClusterRetention,
					ReportRetentionConfig:               defaultReportRetentionConfig,
					VulnerabilityExceptionConfig:        defaultVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          defaultAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
		},
		"Missing private config gets partially configured when Features deactivated": {
			disabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: storage.Config_builder{
				PublicConfig:  samplePublicConfig,
				PrivateConfig: nil,
			}.Build(),
			upsertedConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(defaultAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
					DecommissionedClusterRetention:      defaultDecommissionedClusterRetention,
					ReportRetentionConfig:               defaultReportRetentionConfig,
					VulnerabilityExceptionConfig:        nil,
					AdministrationEventsConfig:          defaultAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
		},
		"Configure decommissioned cluster retention when missing": {
			initialConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      nil,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
			}.Build(),
			upsertedConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      defaultDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
		},
		"Configure report retention when missing": {
			initialConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               nil,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
			}.Build(),
			upsertedConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               defaultReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
		},
		"Configure vulnerability exception management when missing and Feature activated": {
			enabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        nil,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
			}.Build(),
			upsertedConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        defaultVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
		},
		"No update when vulnerability exception management is missing and Feature deactivated": {
			disabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        nil,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
			upsertedConfig: nil,
		},
		"Configure administration event management when missing": {
			enabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          nil,
				}.Build(),
			}.Build(),
			upsertedConfig: storage.Config_builder{
				PublicConfig: samplePublicConfig,
				PrivateConfig: storage.PrivateConfig_builder{
					AlertConfig:                         proto.ValueOrDefault(customAlertRetention.AlertConfig),
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          defaultAdministrationEventsConfig,
				}.Build(),
				PlatformComponentConfig: samplePlatformConfig,
			}.Build(),
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()

			datastore := datastoreMocks.NewMockDataStore(mockCtrl)
			datastore.EXPECT().
				GetConfig(gomock.Any()).
				Times(1).
				Return(testCase.initialConfig, nil)
			if testCase.upsertedConfig != nil {
				datastore.EXPECT().
					UpsertConfig(
						gomock.Any(),
						protomock.GoMockMatcherEqualMessage(testCase.upsertedConfig),
					).
					Return(nil)
			}
			for _, featureFlag := range testCase.disabledFlags {
				t.Setenv(featureFlag, "false")
			}
			for _, featureFlag := range testCase.enabledFlags {
				t.Setenv(featureFlag, "true")
			}

			validateConfigAndPopulateMissingDefaults(datastore)
		})
	}
}
