package continuousprofiling

import (
	"net/url"
	"os"
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
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
		ApplicationName:   utils.IfThenElse[string](env.ContinuousProfilingAppName.Setting() != "", env.ContinuousProfilingAppName.Setting(), os.Getenv("POD_NAME")),
		ServerAddress:     env.ContinuousProfilingServerAddress.Setting(),
		BasicAuthUser:     env.ContinuousProfilingBasicAuthUser.Setting(),
		BasicAuthPassword: env.ContinuousProfilingBasicAuthPassword.Setting(),
		ProfileTypes:      DefaultProfiles,
		Logger:            nil,
	}
}

type OptionFunc func(*pyroscope.Config)

// WithDefaultAppName sets the ApplicationName
// Default: AppName
func WithDefaultAppName(appName string) OptionFunc {
	return func(cfg *pyroscope.Config) {
		// Never override with the default AppName
		if cfg.ApplicationName == "" {
			cfg.ApplicationName = appName
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

func validateServerAddress(address string) (string, error) {
	// We default to https unless http is specified
	sanitizedAddress := urlfmt.FormatURL(address, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if _, err := url.Parse(sanitizedAddress); err != nil {
		return "", errors.Wrapf(err, "unable to parse the server address %q", address)
	}
	return sanitizedAddress, nil
}

func validateConfig(cfg *pyroscope.Config) error {
	if cfg.ApplicationName == "" {
		return ErrApplicationName
	}
	if cfg.ServerAddress == "" {
		return ErrServerAddress
	}
	var err error
	cfg.ServerAddress, err = validateServerAddress(cfg.ServerAddress)
	if err != nil {
		return err
	}
	if len(cfg.ProfileTypes) == 0 {
		return ErrAtLeastOneProfileIsNeeded
	}
	return nil
}

// SetupClient setups the pyroscope client to start the continuous profiling.
func SetupClient(cfg *pyroscope.Config, opts ...OptionFunc) error {
	if !env.ContinuousProfiling.BooleanSetting() {
		return nil
	}

	if profileTypeEnabled(pyroscope.ProfileMutexCount, cfg.ProfileTypes...) {
		runtime.SetMutexProfileFraction(5)
	}

	if profileTypeEnabled(pyroscope.ProfileBlockCount, cfg.ProfileTypes...) {
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

func profileTypeEnabled(profile pyroscope.ProfileType, profiles ...pyroscope.ProfileType) bool {
	for _, p := range profiles {
		if p == profile {
			return true
		}
	}
	return false
}
