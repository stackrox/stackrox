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
)

func TestConfigDataStore(t *testing.T) {
	t.Parallel()
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
	s.storage.EXPECT().Get(gomock.Any()).Return(&storage.Config{
		PublicConfig:  sampleConfig.PublicConfig,
		PrivateConfig: sampleConfig.PrivateConfig,
		PlatformComponentConfig: &storage.PlatformComponentConfig{
			NeedsReevaluation: true,
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
		},
	}, true, nil).Times(1)

	platformConfig, _, err := s.dataStore.GetPlatformComponentConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.NotNil(platformConfig)
	s.True(platformConfig.NeedsReevaluation)
	s.Equal(2, len(platformConfig.Rules))
}

func (s *configDataStoreTestSuite) TestUpsertPlatformComponentConfig() {
	// Test when no update is required
	s.storage.EXPECT().Get(gomock.Any()).Return(&storage.Config{
		PublicConfig:  sampleConfig.PublicConfig,
		PrivateConfig: sampleConfig.PrivateConfig,
		PlatformComponentConfig: &storage.PlatformComponentConfig{
			NeedsReevaluation: false,
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	config, err := s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
	})
	s.NoError(err, "expected no error trying to upsert basic config")
	s.NotNil(config)
	s.False(config.NeedsReevaluation)
	s.Equal(2, len(config.Rules))

	// Test when a re-evaluation should be triggered
	s.storage.EXPECT().Get(gomock.Any()).Return(&storage.Config{
		PublicConfig:  sampleConfig.PublicConfig,
		PrivateConfig: sampleConfig.PrivateConfig,
		PlatformComponentConfig: &storage.PlatformComponentConfig{
			NeedsReevaluation: false,
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		{
			Name: "new rule",
			NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
				Regex: ".*",
			},
		},
	})
	s.NoError(err, "expected no error when upserting new rule")
	s.NotNil(config)
	s.True(config.NeedsReevaluation)
	s.Equal(3, len(config.Rules))

	// Test updating a system rule a couple ways
	s.storage.EXPECT().Get(gomock.Any()).Return(&storage.Config{
		PublicConfig:  sampleConfig.PublicConfig,
		PrivateConfig: sampleConfig.PrivateConfig,
		PlatformComponentConfig: &storage.PlatformComponentConfig{
			NeedsReevaluation: false,
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
		},
	}, true, nil).Times(1)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		{
			Name: "system rule",
			NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
				Regex: "not the system regex",
			},
		},
	})
	s.Error(err, "expected an error when trying to override the system regex")
	s.Nil(config)

	s.storage.EXPECT().Get(gomock.Any()).Return(&storage.Config{
		PublicConfig:  sampleConfig.PublicConfig,
		PrivateConfig: sampleConfig.PrivateConfig,
		PlatformComponentConfig: &storage.PlatformComponentConfig{
			NeedsReevaluation: false,
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		defaultPlatformConfigSystemRule,
	})
	s.NoError(err, "expected no error trying to add duplicate system rule")
	s.NotNil(config)
	s.False(config.NeedsReevaluation)
	s.Equal(2, len(config.Rules))

	// Test duplicating the layered products rule
	s.storage.EXPECT().Get(gomock.Any()).Return(&storage.Config{
		PublicConfig:  sampleConfig.PublicConfig,
		PrivateConfig: sampleConfig.PrivateConfig,
		PlatformComponentConfig: &storage.PlatformComponentConfig{
			NeedsReevaluation: false,
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	config, err = s.dataStore.UpsertPlatformComponentConfigRules(s.hasWriteCtx, []*storage.PlatformComponentConfig_Rule{
		defaultPlatformConfigSystemRule,
		defaultPlatformConfigLayeredProductsRule,
		{
			Name: defaultPlatformConfigLayeredProductsRule.Name,
			NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
				Regex: ".*",
			},
		},
	})
	s.NoError(err, "expected no error trying to add duplicate system rule")
	s.NotNil(config)
	s.Equal(2, len(config.Rules))
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

	customVulnerabilityDeferralConfig = &storage.VulnerabilityExceptionConfig{
		ExpiryOptions: &storage.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: []*storage.DayOption{
				{
					NumDays: 15,
					Enabled: true,
				},
				{
					NumDays: 31,
					Enabled: true,
				},
				{
					NumDays: 61,
					Enabled: true,
				},
				{
					NumDays: 91,
					Enabled: true,
				},
			},
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: true,
				AnyFixable: false,
			},
			CustomDate: false,
			Indefinite: false,
		},
	}

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
			initialConfig: &storage.Config{
				PublicConfig:  samplePublicConfig,
				PrivateConfig: customPrivateConfig,
			},
			upsertedConfig: nil,
		},
		"Missing private config gets fully configured when Features activated": {
			enabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: &storage.Config{
				PublicConfig:  samplePublicConfig,
				PrivateConfig: nil,
			},
			upsertedConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      defaultAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
					DecommissionedClusterRetention:      defaultDecommissionedClusterRetention,
					ReportRetentionConfig:               defaultReportRetentionConfig,
					VulnerabilityExceptionConfig:        defaultVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          defaultAdministrationEventsConfig,
				},
			},
		},
		"Missing private config gets partially configured when Features deactivated": {
			disabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: &storage.Config{
				PublicConfig:  samplePublicConfig,
				PrivateConfig: nil,
			},
			upsertedConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      defaultAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
					DecommissionedClusterRetention:      defaultDecommissionedClusterRetention,
					ReportRetentionConfig:               defaultReportRetentionConfig,
					VulnerabilityExceptionConfig:        nil,
					AdministrationEventsConfig:          defaultAdministrationEventsConfig,
				},
			},
		},
		"Configure decommissioned cluster retention when missing": {
			initialConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      nil,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
			upsertedConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      defaultDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
		},
		"Configure report retention when missing": {
			initialConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               nil,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
			upsertedConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               defaultReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
		},
		"Configure vulnerability exception management when missing and Feature activated": {
			enabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        nil,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
			upsertedConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        defaultVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
		},
		"No update when vulnerability exception management is missing and Feature deactivated": {
			disabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        nil,
					AdministrationEventsConfig:          customAdministrationEventsConfig,
				},
			},
			upsertedConfig: nil,
		},
		"Configure administration event management when missing": {
			enabledFlags: []string{features.UnifiedCVEDeferral.EnvVar()},
			initialConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          nil,
				},
			},
			upsertedConfig: &storage.Config{
				PublicConfig: samplePublicConfig,
				PrivateConfig: &storage.PrivateConfig{
					AlertRetention:                      customAlertRetention,
					ImageRetentionDurationDays:          DefaultImageRetention + 1,
					ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention + 1,
					DecommissionedClusterRetention:      customDecommissionedClusterRetention,
					ReportRetentionConfig:               customReportRetentionConfig,
					VulnerabilityExceptionConfig:        customVulnerabilityDeferralConfig,
					AdministrationEventsConfig:          defaultAdministrationEventsConfig,
				},
			},
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
