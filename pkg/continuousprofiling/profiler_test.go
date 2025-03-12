package continuousprofiling

import (
	"runtime"
	"testing"

	"github.com/grafana/pyroscope-go"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

type ContinuousProfilingSuite struct {
	suite.Suite
}

func (s *ContinuousProfilingSuite) TearDownTest() {
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)
}

func (s *ContinuousProfilingSuite) SetupTest() {
	s.T().Setenv(env.ContinuousProfiling.EnvVar(), "true")
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)
}

var _ suite.SetupTestSuite = (*ContinuousProfilingSuite)(nil)
var _ suite.TearDownTestSuite = (*ContinuousProfilingSuite)(nil)

func TestContinuousProfiling(t *testing.T) {
	suite.Run(t, new(ContinuousProfilingSuite))
}

func (s *ContinuousProfilingSuite) TestDefaultValues() {
	s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
	cfg := DefaultConfig()
	s.Assert().NoError(SetupClient(cfg))
	s.Assert().Equal(*cfg, *DefaultConfig())
	s.Assert().Equal(mutexProfileFraction, runtime.SetMutexProfileFraction(-1))
}

func (s *ContinuousProfilingSuite) TestProfileValidation() {
	s.T().Setenv(env.ContinuousProfilingServerAddress.EnvVar(), "")
	cases := map[string]struct {
		config                pyroscope.Config
		expectedConfig        pyroscope.Config
		expectedMutexFraction int
		expectedError         error
	}{
		"no app name": {
			config:        pyroscope.Config{},
			expectedError: ErrApplicationName,
		},
		"no server address": {
			config: pyroscope.Config{
				ApplicationName: "test",
			},
			expectedError: ErrServerAddress,
		},
		"invalid server address": {
			config: pyroscope.Config{
				ApplicationName: "test",
				ServerAddress:   "https:// invalid",
			},
			expectedError: ErrUnableToParseServerAddress,
		},
		"no profiles defined": {
			config: pyroscope.Config{
				ApplicationName: "test",
				ServerAddress:   "https://valid",
			},
			expectedError: ErrAtLeastOneProfileIsNeeded,
		},
		"no mutex profile": {
			config: pyroscope.Config{
				ApplicationName: "test",
				ServerAddress:   "https://valid",
				ProfileTypes: []pyroscope.ProfileType{
					pyroscope.ProfileCPU,
				},
			},
			expectedConfig: pyroscope.Config{
				ApplicationName: "test",
				ServerAddress:   "https://valid",
				ProfileTypes: []pyroscope.ProfileType{
					pyroscope.ProfileCPU,
				},
			},
			expectedMutexFraction: 0,
		},
		"with default profiles": {
			config: pyroscope.Config{
				ApplicationName: "test",
				ServerAddress:   "https://valid",
				ProfileTypes:    DefaultProfiles,
			},
			expectedConfig: pyroscope.Config{
				ApplicationName: "test",
				ServerAddress:   "https://valid",
				ProfileTypes:    DefaultProfiles,
			},
			expectedMutexFraction: mutexProfileFraction,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			err := SetupClient(&tCase.config)
			if tCase.expectedError == nil {
				s.Assert().NoError(err)
				s.Assert().Equal(tCase.expectedMutexFraction, runtime.SetMutexProfileFraction(-1))
				s.Assert().Equal(tCase.config, tCase.expectedConfig)
			} else {
				s.Assert().ErrorIs(err, tCase.expectedError)
			}
		})
	}
}

func (s *ContinuousProfilingSuite) TestOptions() {
	s.Run("with default name", func() {
		defaultName := "default-name"
		cfg := DefaultConfig()
		s.Assert().NoError(SetupClient(cfg, WithDefaultAppName(defaultName)))
		s.Assert().Equal(defaultName, cfg.ApplicationName)
	})
	s.Run("with profiles", func() {
		s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
		profiles := []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
		}
		cfg := DefaultConfig()
		s.Assert().NoError(SetupClient(cfg, WithProfiles(profiles...)))
		s.Assert().Equal(profiles, cfg.ProfileTypes)
	})
	s.Run("with logging", func() {
		s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
		cfg := DefaultConfig()
		s.Assert().NoError(SetupClient(cfg, WithLogging()))
		s.Assert().Equal(log, cfg.Logger)
	})
}
