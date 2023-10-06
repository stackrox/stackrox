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

func testFlagEnabled(t *testing.T, feature FeatureFlag, test envTest, defaultValue, unchangeableFeature bool) {
	t.Run(fmt.Sprintf("%s/%s", feature.Name(), test.env), func(t *testing.T) {
		oldValue, exists := os.LookupEnv(feature.EnvVar())

		err := os.Setenv(feature.EnvVar(), test.env)
		if err != nil {
			t.Fatalf("Setting env failed for %s", feature.EnvVar())
		}
		if !exists {
			defer func() {
				assert.NoError(t, os.Unsetenv(feature.EnvVar()))
			}()
		} else {
			defer func() {
				assert.NoError(t, os.Setenv(feature.EnvVar(), oldValue))
			}()
		}

		got := feature.Enabled()
		if buildinfo.ReleaseBuild && unchangeableFeature {
			assert.Equal(t, got, defaultValue)
		} else {
			assert.Equal(t, got, test.expected)
		}
	})
}

func TestFeatureFlags(t *testing.T) {
	assert.Panics(t, func() {
		registerFeature("blah", "NOT_ROX_WHATEVER", false)
	})
	defaultTrueFeature := registerFeature("default_true", "ROX_DEFAULT_TRUE", true)
	for _, test := range defaultTrueCases {
		testFlagEnabled(t, defaultTrueFeature, test, true, false)
	}
	defaultFalseFeature := registerFeature("default_false", "ROX_DEFAULT_FALSE", false)
	for _, test := range defaultFalseCases {
		testFlagEnabled(t, defaultFalseFeature, test, false, false)
	}
}

func TestUnchangeableFeatureFlags(t *testing.T) {
	assert.Panics(t, func() {
		registerUnchangeableFeature("blah", "NOT_ROX_WHATEVER", false)
	})
	defaultTrueFeature := registerUnchangeableFeature("default_true", "ROX_DEFAULT_TRUE", true)
	for _, test := range defaultTrueCases {
		testFlagEnabled(t, defaultTrueFeature, test, true, true)
	}
	defaultFalseFeature := registerUnchangeableFeature("default_false", "ROX_DEFAULT_FALSE", false)
	for _, test := range defaultFalseCases {
		testFlagEnabled(t, defaultFalseFeature, test, false, true)
	}
}
