package continuousprofiling

import (
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	ErrApplicationName           = errors.New("the ApplicationName must be defined")
	ErrServerAddress             = errors.New("the ServerAddress must be defined")
	ErrAtLeastOneProfileIsNeeded = errors.New("at least one profile is needed")

	log = logging.LoggerForModule()

	DefaultProfiles = []pyroscope.ProfileType{
		pyroscope.ProfileCPU,
		pyroscope.ProfileAllocObjects,
		pyroscope.ProfileAllocSpace,
		pyroscope.ProfileInuseObjects,
		pyroscope.ProfileInuseSpace,
		pyroscope.ProfileGoroutines,
		pyroscope.ProfileMutexCount,
		pyroscope.ProfileMutexDuration,
		pyroscope.ProfileBlockCount,
		pyroscope.ProfileBlockDuration,
	}
)

// DefaultConfig creates a new configuration with default properties.
func DefaultConfig() *pyroscope.Config {
	return &pyroscope.Config{
		ApplicationName:   "AppName",
		ServerAddress:     env.ContinuousProfilingServerAddress.Setting(),
		BasicAuthUser:     env.ContinuousProfilingBasicAuthUser.Setting(),
		BasicAuthPassword: env.ContinuousProfilingBasicAuthPassword.Setting(),
		ProfileTypes:      DefaultProfiles,
		Logger:            nil,
	}
}

type OptionFunc func(*pyroscope.Config)

// WithAppName sets the ApplicationName
// Default: AppName
func WithAppName(appName string) OptionFunc {
	return func(cfg *pyroscope.Config) {
		cfg.ApplicationName = appName
		if env.ContinuousProfilingAppName.Setting() != "" {
			// If ROX_CONTINUOUS_PROFILING_APP_NAME is set, we override the appName
			cfg.ApplicationName = env.ContinuousProfilingAppName.Setting()
		}
	}
}

// WithProfiles sets the ProfilerTypes
// Default: ProfileCPU, ProfileAllocObjects, ProfileAllocSpace, ProfileInuseObjects, ProfileInuseSpace
func WithProfiles(profiles ...pyroscope.ProfileType) OptionFunc {
	return func(cfg *pyroscope.Config) {
		cfg.ProfileTypes = profiles
	}
}

// WithLogging enables logging
// Default: nil
func WithLogging() OptionFunc {
	return func(cfg *pyroscope.Config) {
		cfg.Logger = log
	}
}

func validateConfig(cfg *pyroscope.Config) error {
	if cfg.ApplicationName == "" {
		return ErrApplicationName
	}
	if cfg.ServerAddress == "" {
		return ErrServerAddress
	}
	if len(cfg.ProfileTypes) == 0 {
		return ErrAtLeastOneProfileIsNeeded
	}
	return nil
}

// SetupContinuousProfilingClient setups the pyroscope client to start the continuous profiling.
func SetupContinuousProfilingClient(cfg *pyroscope.Config, opts ...OptionFunc) error {
	if !env.ContinuousProfiling.BooleanSetting() {
		return nil
	}

	if isInProfiles(pyroscope.ProfileMutexCount, cfg.ProfileTypes...) {
		runtime.SetMutexProfileFraction(5)
	}

	if isInProfiles(pyroscope.ProfileBlockCount, cfg.ProfileTypes...) {
		runtime.SetBlockProfileRate(5)
	}

	for _, o := range opts {
		o(cfg)
	}

	if err := validateConfig(cfg); err != nil {
		return err
	}

	_, err := pyroscope.Start(*cfg)
	if err != nil {
		return err
	}
	log.Info("Continuous Profiling enabled")
	return nil
}

func isInProfiles(profile pyroscope.ProfileType, profiles ...pyroscope.ProfileType) bool {
	for _, p := range profiles {
		if p == profile {
			return true
		}
	}
	return false
}
