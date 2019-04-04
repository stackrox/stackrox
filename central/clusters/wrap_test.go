package clusters

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestGenerateCollectorImage(t *testing.T) {
	var cases = []struct {
		mainImage     string
		collectorTag  string
		expectedImage string
	}{
		{
			mainImage:     "stackrox/main:latest",
			collectorTag:  "latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			mainImage:     "docker.io/stackrox/main:latest",
			collectorTag:  "latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			mainImage:     "stackrox.io/main:latest",
			collectorTag:  "latest",
			expectedImage: "collector.stackrox.io/collector:latest",
		},
		{
			mainImage:     "stackrox.io/main:latest",
			collectorTag:  "loooool",
			expectedImage: "collector.stackrox.io/collector:loooool",
		},
	}

	for _, c := range cases {
		t.Run(c.mainImage, func(t *testing.T) {
			inputImg, err := utils.GenerateImageFromString(c.mainImage)
			assert.NoError(t, err)
			outputImg, err := utils.GenerateImageFromString(c.expectedImage)
			assert.NoError(t, err, "You wrote a bad test and your expected image string didn't parse")
			assert.Equal(t, outputImg.GetName(), generateCollectorImageFromMainImage(inputImg.GetName(), c.collectorTag))
		})
	}
}

func TestGenerateCollectorImageFromString(t *testing.T) {
	var cases = []struct {
		collectorTag   string
		collectorImage string
		expectedImage  string
	}{
		{
			collectorTag:   "latest",
			expectedImage:  "collector.stackrox.io/collector:latest",
			collectorImage: "collector.stackrox.io/collector",
		},
		{
			collectorTag:   "loooool",
			expectedImage:  "collector.stackrox.io/collector:loooool",
			collectorImage: "collector.stackrox.io/collector",
		},
		{
			collectorTag:   "latest",
			expectedImage:  "some.other.path/someothercollectorname:latest",
			collectorImage: "some.other.path/someothercollectorname",
		},
	}

	for _, c := range cases {
		t.Run(c.collectorImage, func(t *testing.T) {
			outputImg, err := utils.GenerateImageFromString(c.expectedImage)
			assert.NoError(t, err, "You wrote a bad test and your expected image string didn't parse")
			collectorName, err := generateCollectorImageNameFromString(c.collectorImage, c.collectorTag)
			assert.NoError(t, err)
			assert.Equal(t, outputImg.GetName(), collectorName)
		})
	}
}

// This should represent the defaults from the UI
func getBaseConfig() Wrap {
	test := Wrap{
		Id:                  "testID",
		Name:                "Test Cluster",
		Type:                storage.ClusterType_KUBERNETES_CLUSTER,
		MainImage:           "stackrox.io/main",
		CentralApiEndpoint:  "central.stackrox:443",
		CollectionMethod:    storage.CollectionMethod_KERNEL_MODULE,
		AdmissionController: false,
	}
	return test
}

func TestImagePaths(t *testing.T) {
	image := "Image"
	imageRegistry := "ImageRegistry"
	collectorImage := "CollectorImage"
	collectorRegistry := "CollectorRegistry"
	collectorVersion := version.GetCollectorVersion()
	var cases = []struct {
		name                      string
		mainImage                 string
		expectedMainRegistry      string
		collectorImage            string
		expectedCollectorImage    string
		expectedCollectorRegistry string
	}{
		{
			name:                      "defaults",
			mainImage:                 "stackrox.io/main",
			expectedMainRegistry:      "https://stackrox.io",
			expectedCollectorImage:    fmt.Sprintf("collector.stackrox.io/collector:%s", collectorVersion),
			expectedCollectorRegistry: "https://collector.stackrox.io",
		},
		{
			name:                      "airgap with generated collector image",
			mainImage:                 "some.other.registry/main",
			expectedMainRegistry:      "https://some.other.registry",
			expectedCollectorImage:    fmt.Sprintf("some.other.registry/collector:%s", collectorVersion),
			expectedCollectorRegistry: "https://some.other.registry",
		},
		{
			name:                      "airgap with specified collector image",
			mainImage:                 "some.other.registry/main",
			expectedMainRegistry:      "https://some.other.registry",
			collectorImage:            "some.other.registry/collector",
			expectedCollectorImage:    fmt.Sprintf("some.other.registry/collector:%s", collectorVersion),
			expectedCollectorRegistry: "https://some.other.registry",
		},
		{
			name:                      "main and collector in different registries (rhel)",
			mainImage:                 "some.rhel.registry.stackrox/main",
			expectedMainRegistry:      "https://some.rhel.registry.stackrox",
			collectorImage:            "collector.stackrox.io/collector",
			expectedCollectorImage:    fmt.Sprintf("collector.stackrox.io/collector:%s", collectorVersion),
			expectedCollectorRegistry: "https://collector.stackrox.io",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config := getBaseConfig()
			if c.mainImage != "" {
				config.MainImage = c.mainImage
			}
			if c.collectorImage != "" {
				config.CollectorImage = c.collectorImage
			}

			// Local tests have no tag, tests run by CircleCI have a tag
			expectedMainImage := c.mainImage
			if version.GetMainVersion() != "" {
				expectedMainImage = fmt.Sprintf("%s:%s", expectedMainImage, version.GetMainVersion())
			}
			fields, err := fieldsFromWrap(config)
			assert.NoError(t, err)
			assert.Contains(t, fields, image)
			assert.Equal(t, expectedMainImage, fields[image])
			assert.Contains(t, fields, imageRegistry)
			assert.Equal(t, c.expectedMainRegistry, fields[imageRegistry])
			assert.Contains(t, fields, collectorImage)
			assert.Equal(t, c.expectedCollectorImage, fields[collectorImage])
			assert.Contains(t, fields, collectorRegistry)
			assert.Equal(t, c.expectedCollectorRegistry, fields[collectorRegistry])
		})
	}
}
