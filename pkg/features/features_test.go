package features

import (
	"fmt"
	"os"
	"testing"

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

func testFlagEnabled(t *testing.T, feature Feature, test envTest) {
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
		if got != test.expected {
			t.Errorf("%s set to %s", feature.EnvVar(), test.env)
			t.Errorf("Expected %t; got %t", test.expected, got)
		}
	})
}

func TestFlags(t *testing.T) {
	defaultTrueFeature := registerFeature("default_true", "DEFAULT_TRUE", true)
	for _, test := range defaultTrueCases {
		testFlagEnabled(t, defaultTrueFeature, test)
	}
	defaultFalseFeature := registerFeature("default_false", "DEFAULT_FALSE", false)
	for _, test := range defaultFalseCases {
		testFlagEnabled(t, defaultFalseFeature, test)
	}
}
