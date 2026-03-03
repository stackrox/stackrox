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

// getProcessFilterModeConfig returns the configuration for the current filter mode.
// Returns nil if the mode is not set, and the default if the mode is invalid.
func getProcessFilterModeConfig() (*ProcessFilterModeConfig, string, error) {
	rawMode, ok := os.LookupEnv(ProcessFilterMode.EnvVar())
	if !ok {
		return nil, "", nil
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

	mode := strings.ToLower(rawMode)

	if mode == "aggressive" {
		return aggressiveConfig, "aggressive", nil
	} else if mode == "default" {
		return defaultConfig, "default", nil
	} else if mode == "minimal" {
		return minimalConfig, "minimal", nil
	}

	return defaultConfig, "default", fmt.Errorf("Invalid mode for environment variable %s=%q. Will use the default.", ProcessFilterMode.EnvVar(), mode)
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
