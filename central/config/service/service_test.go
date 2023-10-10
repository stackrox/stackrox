//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/config/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	defaultExceptionCfg = &storage.VulnerabilityExceptionConfig{
		ExpiryOptions: &storage.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: []*storage.DayOption{
				{
					NumDays: 14,
					Enabled: true,
				},
				{
					NumDays: 30,
					Enabled: true,
				},
				{
					NumDays: 60,
					Enabled: true,
				},
				{
					NumDays: 90,
					Enabled: true,
				},
			},
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: true,
				AnyFixable: true,
			},
			CustomDate: false,
		},
	}
)

func TestConfigService(t *testing.T) {
	suite.Run(t, new(configServiceTestSuite))
}

type configServiceTestSuite struct {
	suite.Suite

	ctx context.Context

	db        *pgtest.TestPostgres
	dataStore datastore.DataStore
	srv       Service
}

func (s *configServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.UnifiedCVEDeferral.EnvVar(), "true")
	if !features.UnifiedCVEDeferral.Enabled() {
		s.T().Skipf("Skip test because %s=false", features.UnifiedCVEDeferral.EnvVar())
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())
	s.dataStore = datastore.NewForTest(s.T(), s.db.DB)
	s.srv = New(s.dataStore)
}

func (s *configServiceTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *configServiceTestSuite) TestNotFound() {
	// Not found because Singleton() was not called and default configuration was not initialize.
	cfg, err := s.srv.GetVulnerabilityExceptionConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.EqualValues(&v1.GetVulnerabilityExceptionConfigResponse{}, cfg)
}

func (s *configServiceTestSuite) TestexceptionConfigOps() {
	initialCfg := &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			ImageRetentionDurationDays:   90,
			VulnerabilityExceptionConfig: defaultExceptionCfg,
		},
	}
	// Insert initial record.
	err := s.dataStore.UpsertConfig(s.ctx, initialCfg)
	s.NoError(err)

	// Verify the initial record exists.
	expected := VulnerabilityExceptionConfigStorageToV1(defaultExceptionCfg)
	cfg, err := s.srv.GetVulnerabilityExceptionConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.EqualValues(expected, cfg.GetConfig())

	// Invalid Update.
	updatedExceptionCfg := initialCfg.Clone().GetPrivateConfig().GetVulnerabilityExceptionConfig()
	updatedExceptionCfg.ExpiryOptions.DayOptions = nil
	req := &v1.UpdateVulnerabilityExceptionConfigRequest{
		Config: VulnerabilityExceptionConfigStorageToV1(updatedExceptionCfg),
	}
	_, err = s.srv.UpdateVulnerabilityExceptionConfig(s.ctx, req)
	s.Error(err)

	// Verify vulnerability exception configuration was not updated.
	cfg, err = s.srv.GetVulnerabilityExceptionConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.EqualValues(expected, cfg.GetConfig())
	// Verify other config was undisturbed.
	pCfg, err := s.srv.GetPrivateConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.Equal(initialCfg.GetPrivateConfig().GetImageRetentionDurationDays(), pCfg.GetImageRetentionDurationDays())

	// Update full config.
	updatedPrivateCfg := &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			ImageRetentionDurationDays: 7,
		},
	}
	_, err = s.srv.PutConfig(s.ctx, &v1.PutConfigRequest{Config: updatedPrivateCfg})
	s.Error(err)

	// Verify the config was not updated.
	pCfg, err = s.srv.GetPrivateConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.Equal(initialCfg.GetPrivateConfig(), pCfg)

	// Verify vulnerability exception configuration was not updated.
	exceptionCfg, err := s.srv.GetVulnerabilityExceptionConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.EqualValues(expected, exceptionCfg.GetConfig())

	// Valid Update.
	updatedExceptionCfg = &storage.VulnerabilityExceptionConfig{
		ExpiryOptions: &storage.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: []*storage.DayOption{
				{
					NumDays: 14,
					Enabled: true,
				},
				{
					NumDays: 0,
					Enabled: false,
				},
			},
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: true,
				AnyFixable: false,
			},
			CustomDate: true,
		},
	}
	req = &v1.UpdateVulnerabilityExceptionConfigRequest{
		Config: VulnerabilityExceptionConfigStorageToV1(updatedExceptionCfg),
	}
	_, err = s.srv.UpdateVulnerabilityExceptionConfig(s.ctx, req)
	s.NoError(err)

	// Verify vulnerability exception configuration was updated.
	cfg, err = s.srv.GetVulnerabilityExceptionConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.EqualValues(VulnerabilityExceptionConfigStorageToV1(updatedExceptionCfg), cfg.GetConfig())
	// Verify other config was undisturbed.
	pCfg, err = s.srv.GetPrivateConfig(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.Equal(initialCfg.GetPrivateConfig().GetImageRetentionDurationDays(), pCfg.GetImageRetentionDurationDays())
}
