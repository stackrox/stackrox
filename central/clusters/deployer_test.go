package clusters

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

var (
	imageRegistryKey        = "ImageRegistry"
	collectorRegistryKey    = "CollectorRegistry"
	collectorImageRemoteKey = "CollectorImageRemote"
	collectorImageTagKey    = "CollectorImageTag"
)

func assertCollectorImageFullPath(t *testing.T, expected string, fields map[string]interface{}) {
	fullRef := fmt.Sprintf("%s/%s:%s", fields[collectorRegistryKey], fields[collectorImageRemoteKey], fields[collectorImageTagKey])
	assert.Equal(t, expected, fullRef)
}

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
			assert.Equal(t, outputImg.GetName(), defaultimages.GenerateNamedImageFromMainImage(inputImg.GetName(), c.collectorTag, defaultimages.Collector))
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
func getBaseConfig() *storage.Cluster {
	return &storage.Cluster{
		Id:                  "testID",
		Name:                "Test Cluster",
		Type:                storage.ClusterType_KUBERNETES_CLUSTER,
		MainImage:           "stackrox.io/main",
		CentralApiEndpoint:  "central.stackrox:443",
		CollectionMethod:    storage.CollectionMethod_KERNEL_MODULE,
		AdmissionController: false,
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: false,
		},
	}
}

func TestImagePaths(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetVersion(t, version.Versions{
		CollectorVersion: "1.2.3",
		MainVersion:      "3.2.1",
	})
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
			expectedMainRegistry:      "stackrox.io",
			expectedCollectorImage:    fmt.Sprintf("collector.stackrox.io/collector:%s-latest", collectorVersion),
			expectedCollectorRegistry: "collector.stackrox.io",
		},
		{
			name:                      "airgap with generated collector image",
			mainImage:                 "some.other.registry/main",
			expectedMainRegistry:      "some.other.registry",
			expectedCollectorImage:    fmt.Sprintf("some.other.registry/collector:%s-latest", collectorVersion),
			expectedCollectorRegistry: "some.other.registry",
		},
		{
			name:                      "airgap with specified collector image",
			mainImage:                 "some.other.registry/main",
			expectedMainRegistry:      "some.other.registry",
			collectorImage:            "some.other.registry/collector",
			expectedCollectorImage:    fmt.Sprintf("some.other.registry/collector:%s-latest", collectorVersion),
			expectedCollectorRegistry: "some.other.registry",
		},
		{
			name:                      "main and collector in different registries (rhel)",
			mainImage:                 "some.rhel.registry.stackrox/main",
			expectedMainRegistry:      "some.rhel.registry.stackrox",
			collectorImage:            "collector.stackrox.io/collector",
			expectedCollectorImage:    fmt.Sprintf("collector.stackrox.io/collector:%s-latest", collectorVersion),
			expectedCollectorRegistry: "collector.stackrox.io",
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

			fields, err := FieldsFromClusterAndRenderOpts(config, RenderOptions{})
			assert.NoError(t, err)
			assert.Contains(t, fields, imageRegistryKey)
			assert.Equal(t, c.expectedMainRegistry, fields[imageRegistryKey])
			assert.Contains(t, fields, collectorRegistryKey)
			assert.Equal(t, c.expectedCollectorRegistry, fields[collectorRegistryKey])

			assertCollectorImageFullPath(t, c.expectedCollectorImage, fields)
		})
	}
}
