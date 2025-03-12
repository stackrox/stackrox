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
	s.T().Setenv(env.ContinuousProfilingAppName.EnvVar(), "test")
	cfg := DefaultConfig()
	s.startClientFuncWrapper.EXPECT().Start(gomock.Any()).Times(1).Return(nil, nil)
	s.Assert().NoError(SetupClient(cfg))
	s.Assert().Equal(*cfg, *DefaultConfig())
	s.Assert().Equal(mutexProfileFraction, runtime.SetMutexProfileFraction(-1))
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

func resetRuntimeProfiles(t *testing.T) {
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)
	require.Eventually(t, func() bool {
		return runtime.SetMutexProfileFraction(-1) == 0
	}, 10*time.Second, 100*time.Millisecond)
}
