package renderer

import (
	"testing"

	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stretchr/testify/assert"
)

func TestComputeOverrides(t *testing.T) {
	cases := []struct {
		ref      string
		expected map[string]string
	}{
		{
			ref: "stackrox.io/main:1.2.3",
		},
		{
			ref: "stackrox.io/main",
		},
		{
			ref: "stackrox.io/main:4.5.6",
			expected: map[string]string{
				"Tag": "4.5.6",
			},
		},
		{
			ref: "stackrox.io/main@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			expected: map[string]string{
				"Tag": "@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			},
		},
		{
			// As of December 2020, this is not a valid image reference (because of the digest
			// length), but we want to tolerantly accept it.
			ref: "stackrox.io/main@sha256:8badf00d",
			expected: map[string]string{
				"Tag": "@sha256:8badf00d",
			},
		},
		{
			ref: "stackrox.io/main:1.2.3@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			expected: map[string]string{
				"Tag": "1.2.3@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			},
		},
		{
			ref: "stackrox.io/sub-repo/main:1.2.3",
			expected: map[string]string{
				"Registry": "stackrox.io/sub-repo",
			},
		},
		{
			ref: "stackrox.io/sub-repo/main:4.5.6",
			expected: map[string]string{
				"Registry": "stackrox.io/sub-repo",
				"Tag":      "4.5.6",
			},
		},
		{
			ref: "stackrox.io/mymain:1.2.3",
			expected: map[string]string{
				"Name": "mymain",
			},
		},
		{
			ref: "stackrox.io/mymain:4.5.6",
			expected: map[string]string{
				"Name": "mymain",
				"Tag":  "4.5.6",
			},
		},
		{
			ref: "stackrox.io/sub-repo/mymain:4.5.6",
			expected: map[string]string{
				"Name": "sub-repo/mymain",
				"Tag":  "4.5.6",
			},
		},
		{
			ref: "docker.io/stackrox/main:1.2.3",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
			},
		},
		{
			ref: "docker.io/stackrox/main:4.5.6",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
				"Tag":      "4.5.6",
			},
		},
		{
			ref: "docker.io/stackrox/mymain:1.2.3",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
				"Name":     "mymain",
			},
		},
		{
			ref: "docker.io/stackrox/mymain:4.5.6",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
				"Name":     "mymain",
				"Tag":      "4.5.6",
			},
		},
		{
			ref: "10.0.0.1:5000/stackrox/main:4.5.6",
			expected: map[string]string{
				"Registry": "10.0.0.1:5000/stackrox",
				"Tag":      "4.5.6",
			},
		},
		{
			ref: "10.0.0.1:5000/stackrox/mymain@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			expected: map[string]string{
				"Registry": "10.0.0.1:5000/stackrox",
				"Name":     "mymain",
				"Tag":      "@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			},
		},
		{
			// As of December 2020, this is not a valid image reference (because of the digest
			// length), but we want to tolerantly accept it.
			ref: "10.0.0.1:5000/stackrox/mymain@sha256:8badf00d",
			expected: map[string]string{
				"Registry": "10.0.0.1:5000/stackrox",
				"Name":     "mymain",
				"Tag":      "@sha256:8badf00d",
			},
		},
		{
			ref: "10.0.0.1:5000/stackrox/mymain:3.0.52.x-5-gdeadbeef@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			expected: map[string]string{
				"Registry": "10.0.0.1:5000/stackrox",
				"Name":     "mymain",
				"Tag":      "3.0.52.x-5-gdeadbeef@sha256:8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d8badf00d",
			},
		},
		{
			// As of December 2020, this is not a valid image reference (because of the digest
			// length), but we want to tolerantly accept it.
			ref: "10.0.0.1:5000/stackrox/mymain:3.0.52.x-5-gdeadbeef@sha256:8badf00d",
			expected: map[string]string{
				"Registry": "10.0.0.1:5000/stackrox",
				"Name":     "mymain",
				"Tag":      "3.0.52.x-5-gdeadbeef@sha256:8badf00d",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			overrides := ComputeImageOverrides(c.ref, "stackrox.io", "main", "1.2.3")
			assert.Equal(t, c.expected, overrides)
		})
	}
}

func TestConfigureImageOverrides(t *testing.T) {
	testFlavor := flavorUtils.MakeImageFlavorForTest(t)
	cases := map[string]struct {
		configValues              CommonConfig
		expectedMainRegistry      string
		expectedMainOverride      map[string]string
		expectedScannerOverride   map[string]string
		expectedScannerDBOverride map[string]string
	}{
		"Override main registry": {
			configValues: CommonConfig{
				MainImage:      "quay.io/rhacs/main:1.2.3",
				ScannerImage:   "quay.io/rhacs/scanner:2.2.2",
				ScannerDBImage: "quay.io/rhacs/scanner-db:2.2.2",
			},
			expectedMainRegistry: "quay.io/rhacs",
		},
		"Don't override main registry": {
			configValues: CommonConfig{
				MainImage:      testFlavor.MainImage(),
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
		},
		"Override main tag if provided": {
			configValues: CommonConfig{
				MainImage:      "test.registry/main:99.9.9",
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
			expectedMainOverride: map[string]string{
				"Tag": "99.9.9",
			},
		},
		"Don't override main tag if no tag provided": {
			configValues: CommonConfig{
				MainImage:      "test.registry/main",
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
		},
		// TODO(RS-397): Cover other overrides in this test cases (e.g. scanner and scanner-db overrides)
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			// (Arrange) configuration parameters
			config := Config{
				K8sConfig: &K8sConfig{
					CommonConfig: c.configValues,
				},
			}

			// (Act) compute overrides
			configureImageOverrides(&config, testFlavor)

			// (Assert) overrides are mapped (if any)
			assert.NotNil(t, config.K8sConfig.ImageOverrides)
			if c.expectedMainRegistry != "" {
				assert.Equal(t, c.expectedMainRegistry, config.K8sConfig.ImageOverrides["MainRegistry"])
			}

			assertOverride(t, config.K8sConfig, c.expectedMainOverride, "Main")
			assertOverride(t, config.K8sConfig, c.expectedScannerOverride, "Scanner")
			assertOverride(t, config.K8sConfig, c.expectedScannerDBOverride, "ScannerDB")
		})
	}
}

func assertOverride(t *testing.T, k8sConfig *K8sConfig, expected map[string]string, overrideKey string) {
	if expected == nil {
		assert.Len(t, k8sConfig.ImageOverrides[overrideKey], 0, "should have no keys in %s map", overrideKey)
	} else {
		assert.EqualValues(t, k8sConfig.ImageOverrides[overrideKey], expected)
	}
}
