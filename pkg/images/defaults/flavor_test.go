package defaults

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type imageFlavorTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestImageFlavor(t *testing.T) {
	suite.Run(t, new(imageFlavorTestSuite))
}

func (s *imageFlavorTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	testbuildinfo.SetForTest(s.T())
	testutils.SetExampleVersion(s.T())
}

func (s *imageFlavorTestSuite) getEnvShouldPanic() {
	s.Panics(func() {
		GetImageFlavorFromEnv()
	})
}

func (s *imageFlavorTestSuite) TestGetImageFlavorFromEnv() {
	testCases := map[string]struct {
		expectedFlavor       ImageFlavor
		shouldPanicOnRelease bool
		shouldPanicAlways    bool
	}{
		"development_build": {
			expectedFlavor:       DevelopmentBuildImageFlavor(),
			shouldPanicOnRelease: true,
		},
		"stackrox.io": {
			expectedFlavor: StackRoxIOReleaseImageFlavor(),
		},
		"rhacs": {
			expectedFlavor: RHACSReleaseImageFlavor(),
		},
		"wrong_value": {
			shouldPanicAlways: true,
		},
		"": {
			shouldPanicAlways: true,
		},
	}

	for envValue, testCase := range testCases {
		s.Run(envValue, func() {
			s.envIsolator.Setenv(imageFlavorEnvName, envValue)
			if testCase.shouldPanicAlways {
				s.getEnvShouldPanic()
				return
			}

			if testCase.shouldPanicOnRelease && buildinfo.ReleaseBuild {
				s.getEnvShouldPanic()
				return
			}

			flavor := GetImageFlavorFromEnv()
			s.Equal(testCase.expectedFlavor, flavor)
		})
	}
}

func (s *imageFlavorTestSuite) TestGetImageFlavorByName() {
	testCases := map[string]struct {
		expectedFlavor          ImageFlavor
		expectedErrorNonRelease string
		expectedErrorRelease    string
	}{
		"development_build": {
			expectedFlavor:       DevelopmentBuildImageFlavor(),
			expectedErrorRelease: "unexpected value 'development_build'",
		},
		"stackrox.io": {
			expectedFlavor: StackRoxIOReleaseImageFlavor(),
		},
		"rhacs": {
			expectedFlavor: RHACSReleaseImageFlavor(),
		},
		"wrong_value": {
			expectedErrorRelease:    "unexpected value 'wrong_value'",
			expectedErrorNonRelease: "unexpected value 'wrong_value'",
		},
		"": {
			expectedErrorRelease:    "unexpected value ''",
			expectedErrorNonRelease: "unexpected value ''",
		},
	}

	for flavorName, testCase := range testCases {
		for _, isRelease := range []bool{false, true} {
			expectedError := testCase.expectedErrorNonRelease
			if isRelease {
				expectedError = testCase.expectedErrorRelease
			}
			s.Run(fmt.Sprintf("'%s'@isRelease=%t", flavorName, isRelease), func() {
				flavor, err := GetImageFlavorByName(flavorName, isRelease)
				if expectedError != "" {
					s.Require().Error(err)
					s.Contains(err.Error(), expectedError)
				} else {
					s.Equal(testCase.expectedFlavor, flavor)
				}
			})
		}
	}
}

func TestGetAllowedImageFlavorNames(t *testing.T) {
	tests := []struct {
		name      string
		isRelease bool
		want      []string
	}{
		{"development", false, []string{"development_build", "stackrox.io", "rhacs"}},
		{"release", true, []string{"stackrox.io", "rhacs"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAllowedImageFlavorNames(tt.isRelease)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
