package features

import (
	"os"
	"testing"
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

func testFlagEnabled(t *testing.T, flag FeatureFlag, feature Feature, test envTest) {
	t.Run(test.env, func(t *testing.T) {
		err := os.Setenv(flag.EnvVar(), test.env)
		if err != nil {
			t.Fatalf("Setting env failed for %s", flag.EnvVar())
		}

		got := feature.Enabled()
		if got != test.expected {
			t.Errorf("%s set to %s", flag.EnvVar(), test.env)
			t.Errorf("Expected %t; got %t", test.expected, got)
		}
	})
}
