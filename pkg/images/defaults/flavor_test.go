package defaults

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type imageFlavorTestSuite struct {
	suite.Suite
}

func TestImageFlavor(t *testing.T) {
	suite.Run(t, new(imageFlavorTestSuite))
}

func (s *imageFlavorTestSuite) SetupTest() {
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
			expectedFlavor: DevelopmentBuildImageFlavor(),
		},
		"stackrox.io": {
			expectedFlavor: StackRoxIOReleaseImageFlavor(),
		},
		"rhacs": {
			expectedFlavor: RHACSReleaseImageFlavor(),
		},
		"opensource": {
			expectedFlavor: OpenSourceImageFlavor(),
		},
		"wrong_value": {
			shouldPanicAlways: true,
		},
		"": {
			expectedFlavor:       DevelopmentBuildImageFlavor(),
			shouldPanicOnRelease: true,
		},
	}

	for envValue, testCase := range testCases {
		s.Run(envValue, func() {
			s.T().Setenv(ImageFlavorEnvName, envValue)
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

func (s *imageFlavorTestSuite) TestChartRepoAndIcon() {
	ossRepoURL := "https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/"
	ossRepoIcon := "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/StackRox_icon.png"
	acsRepoURL := "https://mirror.openshift.com/pub/rhacs/charts"
	acsRepoIcon := "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/Red_Hat-Hat_icon.png"

	testCases := map[string]struct {
		isRelease        []bool
		expectedRepoURL  string
		expectedRepoIcon string
	}{
		"development_build": {
			isRelease:        []bool{false},
			expectedRepoURL:  acsRepoURL,
			expectedRepoIcon: acsRepoIcon,
		},
		"rhacs": {
			isRelease:        []bool{false},
			expectedRepoURL:  acsRepoURL,
			expectedRepoIcon: acsRepoIcon,
		},
		"opensource": {
			isRelease:        []bool{true, false},
			expectedRepoURL:  ossRepoURL,
			expectedRepoIcon: ossRepoIcon,
		},
	}

	for flavorName, testCase := range testCases {
		for _, releaseType := range testCase.isRelease {
			s.Run(flavorName, func() {
				flavor, err := GetImageFlavorByName(flavorName, releaseType)
				s.NoError(err)
				s.Equal(testCase.expectedRepoURL, flavor.ChartRepo.URL)
				s.Equal(testCase.expectedRepoIcon, flavor.ChartRepo.IconURL)
			})
		}
	}
}

func (s *imageFlavorTestSuite) TestOpenSourceImageFlavorDevReleaseTags() {
	f := OpenSourceImageFlavor()
	if buildinfo.ReleaseBuild {
		// All versions/tags should be unified
		s.Equal(f.MainImageTag, "3.0.99.0")
		s.Equal(f.CentralDBImageTag, "3.0.99.0")
		s.Equal(f.CollectorImageTag, "3.0.99.0")
		s.Equal(f.CollectorSlimImageTag, "3.0.99.0")
		s.Equal(f.ScannerImageTag, "3.0.99.0")

		s.Contains(f.CollectorSlimImageName, "-slim")
	} else {
		// Original tags are used
		s.Equal(f.MainImageTag, "3.0.99.0")
		s.Equal(f.CentralDBImageTag, "3.0.99.0")
		s.Equal(f.CollectorImageTag, "99.9.9-latest")
		s.Equal(f.CollectorSlimImageTag, "99.9.9-slim")
		s.Equal(f.ScannerImageTag, "99.9.9")

		s.NotContains(f.CollectorSlimImageName, "-slim")
	}
}

func (s *imageFlavorTestSuite) TestGetImageFlavorByName() {
	testCases := map[string]struct {
		expectedFlavor ImageFlavor
		expectedError  string
	}{
		"development_build": {
			expectedFlavor: DevelopmentBuildImageFlavor(),
		},
		"stackrox.io": {
			expectedFlavor: StackRoxIOReleaseImageFlavor(),
		},
		"rhacs": {
			expectedFlavor: RHACSReleaseImageFlavor(),
		},
		"opensource": {
			expectedFlavor: OpenSourceImageFlavor(),
		},
		"wrong_value": {
			expectedError: "unexpected value 'wrong_value'",
		},
		"": {
			expectedError: "unexpected value ''",
		},
	}

	for flavorName, testCase := range testCases {
		// It is intentional that the logic is tested for both release and non-release because previously there was a
		// different logic depending on that. We can get rid of testing release/non-release after some time in case we
		// see the logic remains the same.
		for _, isRelease := range []bool{false, true} {
			s.Run(fmt.Sprintf("'%s'@isRelease=%t", flavorName, isRelease), func() {
				flavor, err := GetImageFlavorByName(flavorName, isRelease)
				if testCase.expectedError != "" {
					s.Require().Error(err)
					s.Contains(err.Error(), testCase.expectedError)
				} else {
					s.Equal(testCase.expectedFlavor, flavor)
				}
			})
		}
	}
}

func TestGetVisibleImageFlavorNames(t *testing.T) {
	tests := []struct {
		name      string
		isRelease bool
		want      []string
	}{
		{"development", false, []string{"development_build", "stackrox.io", "rhacs", "opensource"}},
		{"release", true, []string{"rhacs", "opensource"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVisibleImageFlavorNames(tt.isRelease)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
