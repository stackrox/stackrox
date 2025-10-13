package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionSetting(t *testing.T) {
	const versionSettingEnvVar = "TEST_VERSION_SETTING"
	const defaultVersion = "1.1.1"
	const minimalVersion = "1.0.1"

	testCases := map[string]struct {
		envValue      string
		expectVersion string
	}{
		"unset": {
			envValue:      "",
			expectVersion: defaultVersion,
		},
		"invalid semver": {
			envValue:      "invalid-semver",
			expectVersion: defaultVersion,
		},
		"valid semver with v prefix": {
			envValue:      "v1.2.3",
			expectVersion: "1.2.3",
		},
		"valid semver without v prefix": {
			envValue:      "1.2.3",
			expectVersion: "1.2.3",
		},
		"valid only minor semver": {
			envValue:      "1.2",
			expectVersion: "1.2.0",
		},
		"valid only major semver": {
			envValue:      "2",
			expectVersion: "2.0.0",
		},
	}

	versionSetting := RegisterVersionSetting(versionSettingEnvVar, defaultVersion, minimalVersion)
	for name, tc := range testCases {
		t.Run(name, func(tt *testing.T) {
			if tc.envValue != "" {
				assert.NoError(tt, os.Setenv(versionSettingEnvVar, tc.envValue))
			} else {
				assert.NoError(tt, os.Unsetenv(versionSettingEnvVar))
			}
			defer func() { _ = os.Unsetenv(versionSettingEnvVar) }()

			result := versionSetting.VersionSetting()
			assert.Equal(tt, tc.expectVersion, result.String())
		})
	}
}

func TestVersionSettingMinimalVersion(t *testing.T) {
	const versionSettingEnvVar = "TEST_VERSION_SETTING_MINIMAL_VERSION"
	versionSetting := RegisterVersionSetting(versionSettingEnvVar, "2.2.2", "1.1.1")
	assert.Equal(t, "2.2.2", versionSetting.VersionSetting().String())

	assert.NoError(t, os.Setenv(versionSettingEnvVar, "3.3.3"))
	assert.Equal(t, "3.3.3", versionSetting.VersionSetting().String())

	assert.NoError(t, os.Setenv(versionSettingEnvVar, "2.0.0"))
	assert.Equal(t, "2.0.0", versionSetting.VersionSetting().String())

	assert.NoError(t, os.Setenv(versionSettingEnvVar, "1.0.0"))
	assert.Equal(t, "2.2.2", versionSetting.VersionSetting().String())
}

func TestVersionSettingPanics(t *testing.T) {
	const versionSettingEnvVar = "TEST_VERSION_SETTING_PANICS"
	assert.Panics(t, func() {
		_ = RegisterVersionSetting(versionSettingEnvVar, "broken-default", "1.1.1")
	})

	assert.Panics(t, func() {
		_ = RegisterVersionSetting(versionSettingEnvVar, "1.1.1", "broken-minimal")
	})
}
