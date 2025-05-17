package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionSetting(t *testing.T) {
	const versionSettingEnvVar = "TEST_VERSION_SETTING"
	const defaultVersion = "1.1.1"

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

	versionSetting := RegisterVersionSetting(versionSettingEnvVar, defaultVersion)
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
