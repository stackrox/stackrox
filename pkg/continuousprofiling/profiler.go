package continuousprofiling

import (
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	ProfileCPU           ProfileType = "cpu"
	ProfileAllocObjects  ProfileType = "alloc-objects"
	ProfileAllocSpace    ProfileType = "alloc-space"
	ProfileInuseObjects  ProfileType = "inuse-objects"
	ProfileInuseSpace    ProfileType = "inuse-space"
	ProfileGoroutines    ProfileType = "goroutines"
	ProfileMutexCount    ProfileType = "mutex-count"
	ProfileMutexDuration ProfileType = "mutex-duration"
	ProfileBlockCount    ProfileType = "block-count"
	ProfileBlockDuration ProfileType = "block-duration"
)

var (
	ErrApplicationName           = errors.New("the ApplicationName must be defined")
	ErrServerAddress             = errors.New("the ServerAddress must be defined")
	ErrAtLeastOneProfileIsNeeded = errors.New("at least one profile is needed")
	ErrUnknownProfileType        = errors.New("unknown profile type")

	log = logging.LoggerForModule()

	DefaultProfiles = []ProfileType{
		ProfileCPU,
		ProfileAllocObjects,
		ProfileAllocSpace,
		ProfileInuseObjects,
		ProfileInuseSpace,
		ProfileGoroutines,
		ProfileMutexCount,
		ProfileMutexDuration,
		ProfileBlockCount,
		ProfileBlockDuration,
	}
)

type ProfileType string

type ProfilerConfiguration struct {
	ApplicationName   string
	ServerAddress     string
	BasicAuthUser     string
	BasicAuthPassword string
	ProfilerTypes     []ProfileType
	WithLogs          bool
}

// DefaultConfig creates a new configuration with default properties.
func DefaultConfig() *ProfilerConfiguration {
	return &ProfilerConfiguration{
		ApplicationName:   "AppName",
		ServerAddress:     env.ContinuousProfilingServerAddress.Setting(),
		BasicAuthUser:     env.ContinuousProfilingBasicAuthUser.Setting(),
		BasicAuthPassword: env.ContinuousProfilingBasicAuthPassword.Setting(),
		ProfilerTypes:     DefaultProfiles,
		WithLogs:          false,
	}
}

// WithAppName sets the ApplicationName
// Default: AppName
func (cfg *ProfilerConfiguration) WithAppName(appName string) *ProfilerConfiguration {
	cfg.ApplicationName = appName
	if env.ContinuousProfilingAppName.Setting() != "" {
		// If ROX_CONTINUOUS_PROFILING_APP_NAME is set, we override the appName
		cfg.ApplicationName = env.ContinuousProfilingAppName.Setting()
	}
	return cfg
}

// WithProfiles sets the ProfilerTypes
// Default: ProfileCPU, ProfileAllocObjects, ProfileAllocSpace, ProfileInuseObjects, ProfileInuseSpace
func (cfg *ProfilerConfiguration) WithProfiles(profiles ...ProfileType) *ProfilerConfiguration {
	cfg.ProfilerTypes = profiles
	return cfg
}

// SetupContinuousProfilingClient setups the pyroscope client to start the continuous profiling.
func SetupContinuousProfilingClient(cfg *ProfilerConfiguration) error {
	if !env.ContinuousProfiling.BooleanSetting() {
		return nil
	}

	if isInProfiles(ProfileMutexCount, cfg.ProfilerTypes...) {
		runtime.SetMutexProfileFraction(5)
	}

	if isInProfiles(ProfileBlockCount, cfg.ProfilerTypes...) {
		runtime.SetBlockProfileRate(5)
	}

	pyroscopeCfg, err := convertToPyroscopeConfig(cfg)
	if err != nil {
		return err
	}
	_, err = pyroscope.Start(*pyroscopeCfg)
	if err != nil {
		return err
	}
	return err
}

func isInProfiles(profile ProfileType, profiles ...ProfileType) bool {
	for _, p := range profiles {
		if p == profile {
			return true
		}
	}
	return false
}
