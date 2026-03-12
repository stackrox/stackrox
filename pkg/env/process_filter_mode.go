package env

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ProcessFilterMode allows users to easily configure process filtering behavior
// using predefined presets instead of individual settings.
//
// Available modes (case-insensitive):
//   - "default": Standard filtering with balanced resource usage
//   - "aggressive": Maximum filtering to minimize resource usage and data volume
//   - "minimal": Minimal filtering to capture more process details
//
// When set, this overrides individual settings (ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES,
// ROX_PROCESS_FILTER_FAN_OUT_LEVELS, ROX_PROCESS_FILTER_MAX_PROCESS_PATHS).
// Individual settings can still be used to override specific values within a mode.
var ProcessFilterMode = RegisterSetting("ROX_PROCESS_FILTER_MODE")

// ProcessFilterModeConfig holds the configuration values for a specific filter mode
type ProcessFilterModeConfig struct {
	MaxExactPathMatches int
	FanOutLevels        []int
	MaxProcessPaths     int
}

var (
	processFilterModePresets = map[string]*ProcessFilterModeConfig{
		"aggressive": {
			MaxExactPathMatches: 1,
			FanOutLevels:        []int{},
			MaxProcessPaths:     1000,
		},
		"default": {
			MaxExactPathMatches: ProcessFilterMaxExactPathMatches.DefaultValue(),
			FanOutLevels:        ProcessFilterFanOutLevels.DefaultValue(),
			MaxProcessPaths:     ProcessFilterMaxProcessPaths.DefaultValue(),
		},
		"minimal": {
			MaxExactPathMatches: 100,
			FanOutLevels:        []int{20, 15, 10, 5},
			MaxProcessPaths:     20000,
		},
	}
)

// getProcessFilterModeConfig returns the configuration for the current filter mode.
// Returns nil if the mode is not set, and the default if the mode is invalid.
func getProcessFilterModeConfig() (*ProcessFilterModeConfig, string, error) {
	rawMode, ok := os.LookupEnv(ProcessFilterMode.EnvVar())
	if !ok {
		return nil, "", nil
	}

	mode := strings.ToLower(rawMode)

	// Check if mode exists in presets
	if preset, found := processFilterModePresets[mode]; found {
		return preset, mode, nil
	}

	// Invalid mode - return default configuration with error
	return processFilterModePresets["default"], "default", fmt.Errorf("invalid mode for environment variable %s=%q. Will use the default.", ProcessFilterMode.EnvVar(), mode)
}

// GetEffectiveProcessFilterConfig returns the effective process filter configuration,
// respecting both the mode preset and any explicitly set individual environment variables.
// Individual settings override mode presets when explicitly set.
func GetEffectiveProcessFilterConfig() (config ProcessFilterModeConfig, mode string, err error) {
	config = ProcessFilterModeConfig{
		MaxExactPathMatches: ProcessFilterMaxExactPathMatches.IntegerSetting(),
		MaxProcessPaths:     ProcessFilterMaxProcessPaths.IntegerSetting(),
	}

	var fanOutErr error
	config.FanOutLevels, fanOutErr = ProcessFilterFanOutLevels.IntegerArraySetting()

	modeConfig, mode, modeErr := getProcessFilterModeConfig()

	if modeConfig == nil {
		// No mode set, return current individual settings
		return config, "", fanOutErr
	}

	// Apply mode preset, but only for values that aren't explicitly overridden
	if _, ok := os.LookupEnv(ProcessFilterMaxExactPathMatches.EnvVar()); !ok {
		config.MaxExactPathMatches = modeConfig.MaxExactPathMatches
	}
	if _, ok := os.LookupEnv(ProcessFilterFanOutLevels.EnvVar()); !ok {
		config.FanOutLevels = modeConfig.FanOutLevels
	}
	if _, ok := os.LookupEnv(ProcessFilterMaxProcessPaths.EnvVar()); !ok {
		config.MaxProcessPaths = modeConfig.MaxProcessPaths
	}

	return config, mode, errors.Join(fanOutErr, modeErr)
}
