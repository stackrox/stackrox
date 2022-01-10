package defaults

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
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
		"stackrox_io_release": {
			expectedFlavor: StackRoxIOReleaseImageFlavor(),
		},
		// TODO(RS-380): Add test for RHACS flavor when available
		// "rhacs_release": {
		//	 expectedFlavor: RHACS
		// },
		"wrong_value": {
			expectedFlavor:    StackRoxIOReleaseImageFlavor(),
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
