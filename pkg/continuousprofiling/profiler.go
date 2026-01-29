package continuousprofiling

import (
	"net/url"
	"runtime"
	"strings"

	"github.com/grafana/pyroscope-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	mutexProfileFraction = 5
	blockProfileRate     = 5
)

// StartClientWrapper wraps the Start function to enable mocking in test
//
//go:generate mockgen-wrapper
type StartClientWrapper interface {
	Start(pyroscope.Config) (*pyroscope.Profiler, error)
}

type startClient struct {
}

// Start wrapper for pyroscope.Start
func (c *startClient) Start(cfg pyroscope.Config) (*pyroscope.Profiler, error) {
	return pyroscope.Start(cfg)
}

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

	startClientFuncWrapper StartClientWrapper = &startClient{}
)

// DefaultConfig creates a new configuration with default properties.
func DefaultConfig() *pyroscope.Config {
	labels, err := parseLabels(env.ContinuousProfilingLabels.Setting())
	if err != nil {
		log.Errorf("Unable to parse Labels in %q: %v", env.ContinuousProfilingLabels.EnvVar(), err)
	}
	return &pyroscope.Config{
		ApplicationName:   env.ContinuousProfilingAppName.Setting(),
		ServerAddress:     env.ContinuousProfilingServerAddress.Setting(),
		BasicAuthUser:     env.ContinuousProfilingBasicAuthUser.Setting(),
		BasicAuthPassword: env.ContinuousProfilingBasicAuthPassword.Setting(),
		ProfileTypes:      DefaultProfiles,
		Logger:            nil,
		Tags:              labels,
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
		return "", errox.InvalidArgs.Newf("unable to parse server address %q", address).CausedBy(err)
	}
	return sanitizedAddress, nil
}

func parseLabels(labels string) (map[string]string, error) {
	parsedLabels := make(map[string]string)
	entries := strings.Split(labels, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("invalid label format: %q (expected key=value)", entry)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, errors.Errorf("empty label key in %q", entry)
		}
		if value == "" {
			return nil, errors.Errorf("empty label value in %q", entry)
		}
		parsedLabels[key] = value
	}
	return parsedLabels, nil
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

	for _, o := range opts {
		o(cfg)
	}

	if err := validateConfig(cfg); err != nil {
		return err
	}

	if profileTypeEnabled(pyroscope.ProfileMutexCount, cfg.ProfileTypes...) {
		runtime.SetMutexProfileFraction(mutexProfileFraction)
	}

	if profileTypeEnabled(pyroscope.ProfileBlockCount, cfg.ProfileTypes...) {
		runtime.SetBlockProfileRate(blockProfileRate)
	}

	_, err := startClientFuncWrapper.Start(*cfg)
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
