package env

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/stackrox/rox/pkg/utils/panic"
)

// VersionSetting represents an environment variable which should be parsed into a semver version
type VersionSetting struct {
	envVar       string
	defaultValue *semver.Version
	minimalValue *semver.Version
}

// EnvVar returns the string name of the environment variable
func (s *VersionSetting) EnvVar() string {
	return s.envVar
}

// DefaultValue returns the default value for the setting
func (s *VersionSetting) DefaultValue() *semver.Version {
	return s.defaultValue
}

// Setting returns the string form of the version environment variable
func (s *VersionSetting) Setting() string {
	return s.VersionSetting().String()
}

// VersionSetting returns the semver.Version object represented by the environment variable
func (s *VersionSetting) VersionSetting() *semver.Version {
	val := os.Getenv(s.envVar)
	if val == "" {
		return s.defaultValue
	}

	version, err := semver.NewVersion(val)
	if err != nil {
		return s.defaultValue
	}

	if version.LessThan(s.minimalValue) {
		return s.defaultValue
	}

	return version
}

// RegisterVersionSetting globally registers and returns a new version setting.
func RegisterVersionSetting(envVar string, defaultValue string, minimalValue string) *VersionSetting {
	defaultVersion, err := semver.NewVersion(defaultValue)
	if err != nil {
		panic.HardPanic(fmt.Sprintf("Incorrect default value of %s: %v", envVar, err))
	}

	minimalVersion, err := semver.NewVersion(minimalValue)
	if err != nil {
		panic.HardPanic(fmt.Sprintf("Incorrect minimal value of %s: %v", envVar, err))
	}

	s := &VersionSetting{
		envVar:       envVar,
		defaultValue: defaultVersion,
		minimalValue: minimalVersion,
	}

	Settings[s.EnvVar()] = s

	return s
}
