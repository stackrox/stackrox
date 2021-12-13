package clusters

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	mainRegistryKey          charts.MetaValuesKey = "MainRegistry"
	imageRemoteKey           charts.MetaValuesKey = "ImageRemote"
	imageTagKey              charts.MetaValuesKey = "ImageTag"
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

func getMain(fields charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields[mainRegistryKey], fields[imageRemoteKey], fields[imageTagKey])
}

type deployerTestSuite struct {
	suite.Suite
}

func TestDeployerTestSuite(t *testing.T) {
	suite.Run(t, new(deployerTestSuite))
}

func (s *deployerTestSuite) SetupTest() {
	testbuildinfo.SetForTest(s.T())
	testutils.SetExampleVersion(s.T())
}

func (s *deployerTestSuite) TestGenerateCollectorImage() {
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
			collectorTag:  "custom",
			expectedImage: "collector.stackrox.io/collector:custom",
		},
	}

	for _, c := range cases {
		s.Run(c.mainImage, func() {
			inputImg, err := utils.GenerateImageFromString(c.mainImage)
			s.NoError(err)
			outputImg, err := utils.GenerateImageFromString(c.expectedImage)
			s.NoError(err, "You wrote a bad test and your expected image string didn't parse")
			s.Equal(outputImg.GetName(), defaultimages.GenerateNamedImageFromMainImage(inputImg.GetName(), c.collectorTag, defaultimages.Collector))
		})
	}
}

func (s *deployerTestSuite) TestGenerateCollectorImageFromString() {
	var cases = []struct {
		collectorTag   string
		collectorImage string
		expectedImage  string
	}{
		{
			collectorImage: "collector.stackrox.io/collector",
			collectorTag:   "latest",
			expectedImage:  "collector.stackrox.io/collector:latest",
		},
		{
			collectorImage: "collector.stackrox.io/collector",
			collectorTag:   "custom",
			expectedImage:  "collector.stackrox.io/collector:custom",
		},
		{
			collectorImage: "some.other.path/someothercollectorname",
			collectorTag:   "latest",
			expectedImage:  "some.other.path/someothercollectorname:latest",
		},
	}

	for _, c := range cases {
		s.Run(c.collectorImage, func() {
			outputImg, err := utils.GenerateImageFromString(c.expectedImage)
			s.NoError(err, "You wrote a bad test and your expected image string didn't parse")
			collectorName, err := generateCollectorImageNameFromString(c.collectorImage, c.collectorTag)
			s.NoError(err)
			s.Equal(outputImg.GetName(), collectorName)
		})
	}
}

// Create a cluster object for test purposes.
func makeTestCluster(mainImage, collectorImage string) *storage.Cluster {
	return &storage.Cluster{
		Id:                  "testID",
		Name:                "Test Cluster",
		Type:                storage.ClusterType_KUBERNETES_CLUSTER,
		MainImage:           mainImage,
		CollectorImage:      collectorImage,
		CentralApiEndpoint:  "central.stackrox:443",
		CollectionMethod:    storage.CollectionMethod_KERNEL_MODULE,
		AdmissionController: false,
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: false,
		},
	}
}

func (s *deployerTestSuite) TestImagePaths() {

	var cases = map[string]struct {
		cluster                  *storage.Cluster
		expectedError            bool
		expectedMain             string
		expectedCollectorFullRef string
		expectedCollectorSlimRef string
	}{
		"default main image / no collector": {
			cluster:                  makeTestCluster("stackrox.io/main", ""),
			expectedMain:             "stackrox.io/main:3.0.99.0",
			expectedCollectorFullRef: "collector.stackrox.io/collector:99.9.9-latest",
			expectedCollectorSlimRef: "collector.stackrox.io/collector:99.9.9-slim",
		},
		"custom main image / no collector": {
			cluster:                  makeTestCluster("quay.io/rhacs/main", ""),
			expectedMain:             "quay.io/rhacs/main:3.0.99.0",
			expectedCollectorFullRef: "quay.io/rhacs/collector:99.9.9-latest",
			expectedCollectorSlimRef: "quay.io/rhacs/collector:99.9.9-slim",
		},
		"custom main and collector images": {
			cluster:                  makeTestCluster("quay.io/rhacs/main", "quay.io/rhacs/collector"),
			expectedMain:             "quay.io/rhacs/main:3.0.99.0",
			expectedCollectorFullRef: "quay.io/rhacs/collector:99.9.9-latest",
			expectedCollectorSlimRef: "quay.io/rhacs/collector:99.9.9-slim",
		},
		"custom main image / default collector image": {
			cluster:                  makeTestCluster("quay.io/rhacs/main", "collector.stackrox.io/collector"),
			expectedMain:             "quay.io/rhacs/main:3.0.99.0",
			expectedCollectorFullRef: "collector.stackrox.io/collector:99.9.9-latest",
			expectedCollectorSlimRef: "collector.stackrox.io/collector:99.9.9-slim",
		},
		"default main image / custom collector image": {
			cluster:                  makeTestCluster("stackrox.io/main", "quay.io/rhacs/collector"),
			expectedMain:             "stackrox.io/main:3.0.99.0",
			expectedCollectorFullRef: "quay.io/rhacs/collector:99.9.9-latest",
			expectedCollectorSlimRef: "quay.io/rhacs/collector:99.9.9-slim",
		},
		"default main image with custom tag / no collector": {
			cluster:                  makeTestCluster("stackrox.io/main:custom", ""),
			expectedMain:             "stackrox.io/main:custom",
			expectedCollectorFullRef: "collector.stackrox.io/collector:99.9.9-latest",
			expectedCollectorSlimRef: "collector.stackrox.io/collector:99.9.9-slim",
		},
		"custom main image with custom tag / default collector image": {
			cluster:                  makeTestCluster("quay.io/rhacs/main:custom", "collector.stackrox.io/collector"),
			expectedMain:             "quay.io/rhacs/main:custom",
			expectedCollectorFullRef: "collector.stackrox.io/collector:99.9.9-latest",
			expectedCollectorSlimRef: "collector.stackrox.io/collector:99.9.9-slim",
		},
		"custom main image / custom collector image: same registry with different namespaces": {
			cluster:                  makeTestCluster("quay.io/namespace-a/main", "quay.io/namespace-b/collector"),
			expectedMain:             "quay.io/namespace-a/main:3.0.99.0",
			expectedCollectorFullRef: "quay.io/namespace-b/collector:99.9.9-latest",
			expectedCollectorSlimRef: "quay.io/namespace-b/collector:99.9.9-slim",
		},
		"custom main image with non-default name": {
			cluster:                  makeTestCluster("quay.io/rhacs/customname", ""),
			expectedMain:             "quay.io/rhacs/customname:3.0.99.0",
			expectedCollectorFullRef: "quay.io/rhacs/collector:99.9.9-latest",
			expectedCollectorSlimRef: "quay.io/rhacs/collector:99.9.9-slim",
		},
		"expectedError: invalid main image": {
			cluster:       makeTestCluster("this is not an image #@!", ""),
			expectedError: true,
		},
		"expectedError: invalid collector image": {
			cluster:       makeTestCluster("stackrox.io/main", "this is not an image #@!"),
			expectedError: true,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			fields, err := FieldsFromClusterAndRenderOpts(c.cluster, RenderOptions{})
			if c.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Contains(fields, mainRegistryKey)
				s.Contains(fields, collectorRegistryKey)
				s.Equal(c.expectedMain, getMain(fields))
				s.Equal(c.expectedCollectorFullRef, getCollectorFull(fields))
				s.Equal(c.expectedCollectorSlimRef, getCollectorSlim(fields))
			}
		})
	}
}

func TestRequiredFieldsArePresent(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetExampleVersion(t)

	fields, err := FieldsFromClusterAndRenderOpts(makeTestCluster("docker.io/stackrox/main", ""), RenderOptions{})
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

	chartRepo := fields[chartRepoKey].(charts.ChartRepo)
	assert.NotEmpty(t, chartRepo.URL)
}
