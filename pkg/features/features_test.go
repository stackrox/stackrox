package features

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stretchr/testify/assert"
)

type envTest struct {
	env      string
	expected bool
}

var (
	defaultTrueCases = []envTest{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"", true},
		{"blargle!", true},
	}

	defaultFalseCases = []envTest{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"", false},
		{"blargle!", false},
	}
)

func testFlagEnabled(t *testing.T, feature FeatureFlag, envSetting string, expected bool) {
	t.Run(fmt.Sprintf("%s/%s", feature.Name(), envSetting), func(t *testing.T) {
		oldValue, exists := os.LookupEnv(feature.EnvVar())

		err := os.Setenv(feature.EnvVar(), envSetting)
		if err != nil {
			t.Fatalf("Setting env failed for %s", feature.EnvVar())
		}

		// Make sure the env var is cleaned up or reset after the test finishes
		if !exists {
			defer func() {
				assert.NoError(t, os.Unsetenv(feature.EnvVar()))
			}()
		} else {
			defer func() {
				assert.NoError(t, os.Setenv(feature.EnvVar(), oldValue))
			}()
		}

		assert.Equal(t, feature.Enabled(), expected)
	})
}

func TestFeatureEnvVarStartsWithRox(t *testing.T) {
	// Use two blocks because it should fail if either of them doesn't panic
	assert.Panics(t, func() {
		registerFeature("blah", "NOT_ROX_WHATEVER")
	})
	assert.Panics(t, func() {
		registerFeature("blah", "NOT_ROX_WHATEVER", unchangeableInProd)
	})
}

func TestFeatureFlags(t *testing.T) {
	defaultTrueFeature := registerFeature("default_true", "ROX_DEFAULT_TRUE", enabled)
	for _, test := range defaultTrueCases {
		testFlagEnabled(t, defaultTrueFeature, test.env, test.expected)
	}
	defaultFalseFeature := registerFeature("default_false", "ROX_DEFAULT_FALSE")
	for _, test := range defaultFalseCases {
		testFlagEnabled(t, defaultFalseFeature, test.env, test.expected)
	}
}

// Test that the feature override works as expected given an appropriate overridable setting
func TestFeatureOverrideSetting(t *testing.T) {
	overridableFeature := registerFeature("test_feat", "ROX_TEST_FEAT", enabled)
	nonoverridableFeature := registerFeature("test_feat", "ROX_TEST_FEAT", enabled, withUnchangeable(true))

	// overridable features can be changed from the default value (true)
	testFlagEnabled(t, overridableFeature, "false", false)

	// unchangeable features cannot be changed from the default value (true)
	testFlagEnabled(t, nonoverridableFeature, "false", true)
}

// This is a similar test as `TestFeatureOverrideSetting` but the difference is that this tests the fact that
// registerUnchangeableFeature sets the correct overridable setting on a release build
func TestOverridesOnReleaseBuilds(t *testing.T) {
	overridableFeature := registerFeature("test_feat", "ROX_TEST_FEAT", enabled)
	unchangeableFeature := registerFeature("test_feat", "ROX_TEST_FEAT", enabled, unchangeableInProd)

	// overridable features can be changed from the default value (true) regardless of the type of build
	testFlagEnabled(t, overridableFeature, "false", false)

	// unchangeable features can only be changed from the default value (true) on non-release builds
	if buildinfo.ReleaseBuild {
		testFlagEnabled(t, unchangeableFeature, "false", true)
	} else {
		testFlagEnabled(t, unchangeableFeature, "false", false)
	}
}
