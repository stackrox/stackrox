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
var ProcessFilterMode = RegisterSetting("ROX_PROCESS_FILTER_MODE", WithDefault("default"))

// ProcessFilterModeConfig holds the configuration values for a specific filter mode
type ProcessFilterModeConfig struct {
	MaxExactPathMatches int
	FanOutLevels        []int
	MaxProcessPaths     int
}

// GetProcessFilterModeConfig returns the configuration for the current filter mode.
// Returns nil if the mode is not set, and the default if the mode is invalid.
func GetProcessFilterModeConfig() (*ProcessFilterModeConfig, error) {
	_, ok := os.LookupEnv(ProcessFilterMode.EnvVar())
	if !ok {
		return nil, nil
	}

	defaultConfig := &ProcessFilterModeConfig{
		MaxExactPathMatches: ProcessFilterMaxExactPathMatches.DefaultValue(),
		FanOutLevels:        ProcessFilterFanOutLevels.DefaultValue(),
		MaxProcessPaths:     ProcessFilterMaxProcessPaths.DefaultValue(),
	}

	aggressiveConfig := &ProcessFilterModeConfig{
		MaxExactPathMatches: 1,
		FanOutLevels:        []int{},
		MaxProcessPaths:     1000,
	}

	minimalConfig := &ProcessFilterModeConfig{
		MaxExactPathMatches: 100,
		FanOutLevels:        []int{20, 15, 10, 5},
		MaxProcessPaths:     20000,
	}

	mode := strings.ToLower(ProcessFilterMode.Setting())

	if mode == "aggressive" {
		return aggressiveConfig, nil
	} else if mode == "default" {
		return defaultConfig, nil
	} else if mode == "minimal" {
		return minimalConfig, nil
	}

	if mode == "" {
		return defaultConfig, fmt.Errorf("%s set to empty string. Will use the default.", ProcessFilterMode.EnvVar())
	}

	return defaultConfig, fmt.Errorf("Unrecognized mode for environment variable %s: %s. Will use the default.", ProcessFilterMode.EnvVar(), mode)
}

// GetEffectiveProcessFilterConfig returns the effective process filter configuration,
// respecting both the mode preset and any explicitly set individual environment variables.
// Individual settings override mode presets when explicitly set.
func GetEffectiveProcessFilterConfig() (ProcessFilterModeConfig, error) {
	config := ProcessFilterModeConfig{
		MaxExactPathMatches: ProcessFilterMaxExactPathMatches.IntegerSetting(),
		MaxProcessPaths:     ProcessFilterMaxProcessPaths.IntegerSetting(),
	}

	var fanOutErr error
	config.FanOutLevels, fanOutErr = ProcessFilterFanOutLevels.IntegerArraySetting()
	modeConfig, modeErr := GetProcessFilterModeConfig()

	if modeConfig == nil {
		// No valid mode set, return current settings
		return config, errors.Join(fanOutErr, modeErr)
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

	return config, errors.Join(fanOutErr, modeErr)
}
