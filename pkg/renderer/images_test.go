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
			ref: "stackrox.io/main:4.5.6",
			expected: map[string]string{
				"Tag": "4.5.6",
			},
		},
		{
			ref: "stackrox.io/main",
			expected: map[string]string{
				"Tag": "latest",
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
		configValues               CommonConfig
		expectedMainRegistry       string
		expectedMainOverrides      map[string]string
		expectedScannerOverrides   map[string]string
		expectedScannerDBOverrides map[string]string
	}{
		"Override Main Registry": {
			configValues: CommonConfig{
				MainImage:      "quay.io/rhacs/main",
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
			expectedMainRegistry: "quay.io/rhacs",
			expectedMainOverrides: map[string]string{
				"Tag": "latest",
			},
			expectedScannerOverrides: map[string]string{
				"Registry": "test.registry",
			},
			expectedScannerDBOverrides: map[string]string{
				"Registry": "test.registry",
			},
		},
		"Don't override main registry": {
			configValues: CommonConfig{
				MainImage:      testFlavor.MainImage(),
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
		},
		"Override Main sub-repo": {
			configValues: CommonConfig{
				MainImage:      "test.registry/sub-repo/main",
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
			expectedMainRegistry: "test.registry/sub-repo",
			expectedMainOverrides: map[string]string{
				"Tag": "latest",
			},
			expectedScannerOverrides: map[string]string{
				"Registry": "test.registry",
			},
			expectedScannerDBOverrides: map[string]string{
				"Registry": "test.registry",
			},
		},
		"Override Main sub-repo and name": {
			configValues: CommonConfig{
				MainImage:      "test.registry/sub-repo/my-main",
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
			expectedMainOverrides: map[string]string{
				"Name": "sub-repo/my-main",
				"Tag":  "latest",
			},
		},
		// Scanner
		"Override Scanner registry": {
			configValues: CommonConfig{
				MainImage:      testFlavor.MainImage(),
				ScannerImage:   "quay.io/rhacs/scanner",
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
			expectedScannerOverrides: map[string]string{
				"Registry": "quay.io/rhacs",
				"Tag":      "latest",
			},
		},
		// ScannerDB
		"Override ScannerDB registry": {
			configValues: CommonConfig{
				MainImage:      testFlavor.MainImage(),
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: "quay.io/rhacs/scanner-db",
			},
			expectedScannerDBOverrides: map[string]string{
				"Registry": "quay.io/rhacs",
				"Tag":      "latest",
			},
		},
		// Registries combinations
		"Override Main and Scanner registries": {
			configValues: CommonConfig{
				MainImage:      "quay.io/rhacs/main",
				ScannerImage:   "stackrox.io/scanner",
				ScannerDBImage: testFlavor.ScannerDBImage(),
			},
			expectedMainRegistry: "quay.io/rhacs",
			expectedMainOverrides: map[string]string{
				"Tag": "latest",
			},
			expectedScannerOverrides: map[string]string{
				"Registry": "stackrox.io",
				"Tag":      "latest",
			},
			expectedScannerDBOverrides: map[string]string{
				"Registry": "test.registry",
			},
		},
		"Override Main and ScannerDB registries": {
			configValues: CommonConfig{
				MainImage:      "quay.io/rhacs/main",
				ScannerImage:   testFlavor.ScannerImage(),
				ScannerDBImage: "stackrox.io/scanner-db",
			},
			expectedMainRegistry: "quay.io/rhacs",
			expectedMainOverrides: map[string]string{
				"Tag": "latest",
			},
			expectedScannerOverrides: map[string]string{
				"Registry": "test.registry",
			},
			expectedScannerDBOverrides: map[string]string{
				"Registry": "stackrox.io",
				"Tag":      "latest",
			},
		},
		"Override Main, Scanner and ScannerDB with the same registries": {
			configValues: CommonConfig{
				MainImage:      "quay.io/rhacs/main",
				ScannerImage:   "quay.io/rhacs/scanner",
				ScannerDBImage: "quay.io/rhacs/scanner-db",
			},
			expectedMainRegistry: "quay.io/rhacs",
			expectedMainOverrides: map[string]string{
				"Tag": "latest",
			},
			expectedScannerOverrides: map[string]string{
				"Tag": "latest",
			},
			expectedScannerDBOverrides: map[string]string{
				"Tag": "latest",
			},
		},
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
			if c.expectedMainOverrides != nil {
				assert.Equal(t, c.expectedMainOverrides, config.K8sConfig.ImageOverrides["Main"].(map[string]string))
			} else {
				assert.Len(t, config.K8sConfig.ImageOverrides["Main"], 0, "should have no keys in Main map")
			}
			if c.expectedScannerOverrides != nil {
				assert.Equal(t, c.expectedScannerOverrides, config.K8sConfig.ImageOverrides["Scanner"].(map[string]string))
			} else {
				assert.Len(t, config.K8sConfig.ImageOverrides["Scanner"], 0, "should have no keys in Scanner map")
			}
			if c.expectedScannerDBOverrides != nil {
				assert.Equal(t, c.expectedScannerDBOverrides, config.K8sConfig.ImageOverrides["ScannerDB"].(map[string]string))
			} else {
				assert.Len(t, config.K8sConfig.ImageOverrides["ScannerDB"], 0, "should have no keys in ScannerDB map")
			}
		})
	}
}
