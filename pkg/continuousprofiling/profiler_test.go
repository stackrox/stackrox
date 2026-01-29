package continuousprofiling

import (
	"runtime"
	"testing"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/continuousprofiling/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type continuousProfilingSuite struct {
	suite.Suite
	ctrl                   *gomock.Controller
	startClientFuncWrapper *mocks.MockStartClientWrapper
}

func (s *continuousProfilingSuite) TearDownTest() {
	resetRuntimeProfiles(s.T())
}

func (s *continuousProfilingSuite) SetupTest() {
	s.T().Setenv(env.ContinuousProfiling.EnvVar(), "true")
	resetRuntimeProfiles(s.T())
	s.ctrl = gomock.NewController(s.T())
	s.startClientFuncWrapper = mocks.NewMockStartClientWrapper(s.ctrl)
	startClientFuncWrapper = s.startClientFuncWrapper
}

var _ suite.SetupTestSuite = (*continuousProfilingSuite)(nil)
var _ suite.TearDownTestSuite = (*continuousProfilingSuite)(nil)

func TestContinuousProfiling(t *testing.T) {
	suite.Run(t, new(continuousProfilingSuite))
}

func (s *continuousProfilingSuite) TestDefaultValues() {
	s.Run("all defaults success", func() {
		s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
		s.T().Setenv(env.ContinuousProfilingLabels.EnvVar(), "app=stackrox,env=production")
		cfg := DefaultConfig()
		s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, nil)
		s.Assert().NoError(SetupClient(cfg))
		s.Assert().Equal(*cfg, *DefaultConfig())
		s.Assert().Equal(mutexProfileFraction, runtime.SetMutexProfileFraction(-1))
		s.Assert().Equal(map[string]string{
			"app": "stackrox",
			"env": "production",
		}, cfg.Tags)
	})
	s.Run("fail labels parsing", func() {
		s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
		s.T().Setenv(env.ContinuousProfilingLabels.EnvVar(), "invalid-labels")
		cfg := DefaultConfig()
		s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, nil)
		s.Assert().NoError(SetupClient(cfg))
		s.Assert().Equal(*cfg, *DefaultConfig())
		s.Assert().Equal(mutexProfileFraction, runtime.SetMutexProfileFraction(-1))
		s.Assert().Nil(cfg.Tags)
	})
}

func (s *continuousProfilingSuite) TestProfileValidation() {
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
			expectedError: errox.InvalidArgs,
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
			defer resetRuntimeProfiles(s.T())
			s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).AnyTimes().Return(nil, nil)
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

func (s *continuousProfilingSuite) TestOptions() {
	s.Run("with default name", func() {
		defer resetRuntimeProfiles(s.T())
		defaultName := "default-name"
		cfg := DefaultConfig()
		s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, nil)
		s.Assert().NoError(SetupClient(cfg, WithDefaultAppName(defaultName)))
		s.Assert().Equal(defaultName, cfg.ApplicationName)
	})
	s.Run("with profiles", func() {
		defer resetRuntimeProfiles(s.T())
		s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
		profiles := []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
		}
		cfg := DefaultConfig()
		s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, nil)
		s.Assert().NoError(SetupClient(cfg, WithProfiles(profiles...)))
		s.Assert().Equal(profiles, cfg.ProfileTypes)
	})
	s.Run("with logging", func() {
		defer resetRuntimeProfiles(s.T())
		s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
		cfg := DefaultConfig()
		s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, nil)
		s.Assert().NoError(SetupClient(cfg, WithLogging()))
		s.Assert().Equal(log, cfg.Logger)
	})
}

func (s *continuousProfilingSuite) TestClientStartError() {
	s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
	cfg := DefaultConfig()
	s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, errors.New("some error"))
	s.Assert().Error(SetupClient(cfg))
}

func (s *continuousProfilingSuite) TestParseLabels() {
	cases := map[string]struct {
		input          string
		expectedLabels map[string]string
		expectedError  string
	}{
		"empty string": {
			input:          "",
			expectedLabels: map[string]string{},
		},
		"single label": {
			input: "key=value",
			expectedLabels: map[string]string{
				"key": "value",
			},
		},
		"multiple labels": {
			input: "app=stackrox,env=production,team=security",
			expectedLabels: map[string]string{
				"app":  "stackrox",
				"env":  "production",
				"team": "security",
			},
		},
		"labels with whitespace": {
			input: " key1 = value1 , key2 = value2 ",
			expectedLabels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		"empty entries ignored": {
			input: "key1=value1,,key2=value2,,,",
			expectedLabels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		"value with equals sign": {
			input: "url=https://example.com,token=abc=123",
			expectedLabels: map[string]string{
				"url":   "https://example.com",
				"token": "abc=123",
			},
		},
		"invalid format no equals": {
			input:         "invalid",
			expectedError: "invalid label format",
		},
		"empty key": {
			input:         "=value",
			expectedError: "empty label key",
		},
		"empty key with whitespace": {
			input:         "  =value",
			expectedError: "empty label key",
		},
		"empty value": {
			input:         "key=",
			expectedError: "empty label value",
		},
		"empty value with whitespace": {
			input:         "key=  ",
			expectedError: "empty label value",
		},
		"multiple entries with one invalid": {
			input:         "valid=yes,invalid",
			expectedError: "invalid label format",
		},
	}

	for tName, tCase := range cases {
		s.Run(tName, func() {
			labels, err := parseLabels(tCase.input)
			if tCase.expectedError != "" {
				s.Assert().Error(err)
				s.Assert().Contains(err.Error(), tCase.expectedError)
				s.Assert().Nil(labels)
			} else {
				s.Assert().NoError(err)
				s.Assert().Equal(tCase.expectedLabels, labels)
			}
		})
	}
}

func resetRuntimeProfiles(t *testing.T) {
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)
	require.Eventually(t, func() bool {
		return runtime.SetMutexProfileFraction(-1) == 0
	}, 10*time.Second, 100*time.Millisecond)
}
