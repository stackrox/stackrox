package clusters

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

const (
	mainRegistryKey          charts.MetaValuesKey = "MainRegistry"
	imageRemoteKey           charts.MetaValuesKey = "ImageRemote"
	collectorRegistryKey     charts.MetaValuesKey = "CollectorRegistry"
	collectorImageRemoteKey  charts.MetaValuesKey = "CollectorImageRemote"
	collectorFullImageTagKey charts.MetaValuesKey = "CollectorFullImageTag"
	collectorSlimImageTagKey charts.MetaValuesKey = "CollectorSlimImageTag"
	versionsKey              charts.MetaValuesKey = "Versions"
	chartRepoKey             charts.MetaValuesKey = "ChartRepo"
)

func getCollectorFull(fields charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields[collectorRegistryKey], fields[collectorImageRemoteKey], fields[collectorFullImageTagKey])
}

func getCollectorSlim(fields charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields[collectorRegistryKey], fields[collectorImageRemoteKey], fields[collectorSlimImageTagKey])
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
	testutils.SetExampleVersion(t)
	collectorVersion := version.GetCollectorVersion()
	var cases = []struct {
		name                      string
		mainImage                 string
		expectedMainRegistry      string
		collectorImage            string
		expectedCollectorFullRef  string
		expectedCollectorSlimRef  string
		expectedCollectorRegistry string
	}{
		{
			name:                      "defaults",
			mainImage:                 "stackrox.io/main",
			expectedMainRegistry:      "stackrox.io",
			expectedCollectorFullRef:  fmt.Sprintf("collector.stackrox.io/collector:%s-latest", collectorVersion),
			expectedCollectorSlimRef:  fmt.Sprintf("collector.stackrox.io/collector:%s-slim", collectorVersion),
			expectedCollectorRegistry: "collector.stackrox.io",
		},
		{
			name:                      "airgap with generated collector image",
			mainImage:                 "some.other.registry/main",
			expectedMainRegistry:      "some.other.registry",
			expectedCollectorFullRef:  fmt.Sprintf("some.other.registry/collector:%s-latest", collectorVersion),
			expectedCollectorSlimRef:  fmt.Sprintf("some.other.registry/collector:%s-slim", collectorVersion),
			expectedCollectorRegistry: "some.other.registry",
		},
		{
			name:                      "airgap with specified collector image",
			mainImage:                 "some.other.registry/main",
			expectedMainRegistry:      "some.other.registry",
			collectorImage:            "some.other.registry/collector",
			expectedCollectorFullRef:  fmt.Sprintf("some.other.registry/collector:%s-latest", collectorVersion),
			expectedCollectorSlimRef:  fmt.Sprintf("some.other.registry/collector:%s-slim", collectorVersion),
			expectedCollectorRegistry: "some.other.registry",
		},
		{
			name:                      "main and collector in different registries (rhel)",
			mainImage:                 "some.rhel.registry.stackrox/main",
			expectedMainRegistry:      "some.rhel.registry.stackrox",
			collectorImage:            "collector.stackrox.io/collector",
			expectedCollectorFullRef:  fmt.Sprintf("collector.stackrox.io/collector:%s-latest", collectorVersion),
			expectedCollectorSlimRef:  fmt.Sprintf("collector.stackrox.io/collector:%s-slim", collectorVersion),
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

			assert.Equal(t, c.expectedMainRegistry, fields[mainRegistryKey])
			assert.Equal(t, c.expectedCollectorRegistry, fields[collectorRegistryKey])
			assert.Equal(t, c.expectedCollectorFullRef, getCollectorFull(fields))
			assert.Equal(t, c.expectedCollectorSlimRef, getCollectorSlim(fields))
		})
	}
}

func TestRequiredFieldsArePresent(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetExampleVersion(t)

	fields, err := FieldsFromClusterAndRenderOpts(getBaseConfig(), RenderOptions{})
	assert.NoError(t, err)

	assert.NotEmpty(t, fields[mainRegistryKey])
	assert.NotEmpty(t, fields[imageRemoteKey])
	assert.NotEmpty(t, fields[collectorRegistryKey])
	assert.NotEmpty(t, fields[collectorImageRemoteKey])
	assert.NotEmpty(t, fields[collectorSlimImageTagKey])
	assert.NotEmpty(t, fields[collectorFullImageTagKey])

	versions := fields[versionsKey].(version.Versions)
	assert.NotEmpty(t, versions.ChartVersion)
	assert.NotEmpty(t, versions.MainVersion)
	assert.NotEmpty(t, versions.CollectorVersion)
	assert.NotEmpty(t, versions.ScannerVersion)

	chartRepo := fields[chartRepoKey].(images.ChartRepo)
	assert.NotEmpty(t, chartRepo.URL)
}
