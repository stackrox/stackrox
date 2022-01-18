package clusters

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func getCollectorFull(fields *charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields.CollectorRegistry, fields.CollectorFullImageRemote, fields.CollectorFullImageTag)
}

func getCollectorSlim(fields *charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields.CollectorRegistry, fields.CollectorSlimImageRemote, fields.CollectorSlimImageTag)
}

func getMain(fields *charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields.MainRegistry, fields.ImageRemote, fields.ImageTag)
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

var (
	NoNamespaceImage = &storage.ImageName{
		Registry: "my.registry.io",
		Remote:   "main",
		Tag:      "latest",
	}
	ImageWithSingleNamespace = &storage.ImageName{
		Registry: "my.registry.io",
		Remote:   "stackrox/main",
		Tag:      "latest",
	}
)

func (s *deployerTestSuite) Test_deriveImageWithNewName() {
	var cases = map[string]struct {
		baseImage                            *storage.ImageName
		targetImageName                      string
		expectedRegistry, expectedRepository string
	}{
		"my.registry.io/imageA": {
			baseImage:          NoNamespaceImage,
			targetImageName:    "imageA",
			expectedRegistry:   "my.registry.io",
			expectedRepository: "imageA",
		},
		"my.registry.io/imageB": {
			baseImage:          NoNamespaceImage,
			targetImageName:    "imageB",
			expectedRegistry:   "my.registry.io",
			expectedRepository: "imageB",
		},
		"my.registry.io/stackrox/imageA": {
			baseImage:          ImageWithSingleNamespace,
			targetImageName:    "imageA",
			expectedRegistry:   "my.registry.io",
			expectedRepository: "stackrox/imageA",
		},
		"my.registry.io/stackrox/imageB": {
			baseImage:          ImageWithSingleNamespace,
			targetImageName:    "company/imageB",
			expectedRegistry:   "my.registry.io",
			expectedRepository: "stackrox/imageB",
		},
	}

	for name, testCase := range cases {
		s.Run(name, func() {
			img := deriveImageWithNewName(testCase.baseImage, testCase.targetImageName)
			s.Equal(testCase.expectedRegistry, img.Registry)
			s.Equal(testCase.expectedRepository, img.Remote)
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

func testMetaValueGenerationWithImageFlavor(s *deployerTestSuite, flavor defaults.ImageFlavor) {
	defaultMainImageNoTag := flavor.MainImageNoTag()
	defaultMainImage := flavor.MainImage()
	defaultCollectorFullImageNoTag := flavor.CollectorFullImageNoTag()
	defaultCollectorFullImage := flavor.CollectorFullImage()
	defaultCollectorSlimImage := flavor.CollectorSlimImage()

	var cases = map[string]struct {
		cluster                  *storage.Cluster
		expectedError            bool
		expectedMain             string
		expectedCollectorFullRef string
		expectedCollectorSlimRef string
	}{
		"default main image / no collector": {
			cluster:                  makeTestCluster(defaultMainImageNoTag, ""),
			expectedMain:             defaultMainImage,
			expectedCollectorFullRef: defaultCollectorFullImage,
			expectedCollectorSlimRef: defaultCollectorSlimImage,
		},
		"custom main image (with namespace) / no collector": {
			cluster:                  makeTestCluster("quay.io/rhacs/main", ""),
			expectedMain:             fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.MainImageName, flavor.MainImageTag),
			expectedCollectorFullRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedCollectorSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorSlimImageName, flavor.CollectorSlimImageTag),
		},
		"custom main image (without namespace) / no collector": {
			cluster:                  makeTestCluster("example.io/main", ""),
			expectedMain:             fmt.Sprintf("example.io/%s:%s", flavor.MainImageName, flavor.MainImageTag),
			expectedCollectorFullRef: fmt.Sprintf("example.io/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedCollectorSlimRef: fmt.Sprintf("example.io/%s:%s", flavor.CollectorSlimImageName, flavor.CollectorSlimImageTag),
		},
		"custom main and collector images": {
			cluster:                  makeTestCluster("quay.io/rhacs/main", "quay.io/rhacs/collector"),
			expectedMain:             fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorFullRef: fmt.Sprintf("quay.io/rhacs/collector:%s", flavor.CollectorImageTag),
			expectedCollectorSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorSlimImageName, flavor.CollectorSlimImageTag),
		},
		"custom main image / default collector image": {
			cluster:                  makeTestCluster("quay.io/rhacs/main", defaultCollectorFullImageNoTag),
			expectedMain:             fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorFullRef: defaultCollectorFullImage,
			expectedCollectorSlimRef: defaultCollectorSlimImage,
		},
		"default main image / custom collector image": {
			cluster:                  makeTestCluster(defaultMainImage, "quay.io/rhacs/collector"),
			expectedMain:             defaultMainImage,
			expectedCollectorFullRef: fmt.Sprintf("quay.io/rhacs/collector:%s", flavor.CollectorImageTag),
			expectedCollectorSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorSlimImageName, flavor.CollectorSlimImageTag),
		},
		"default main image with custom tag / no collector": {
			cluster:                  makeTestCluster(fmt.Sprintf("%s:custom", defaultMainImageNoTag), ""),
			expectedMain:             fmt.Sprintf("%s:custom", defaultMainImageNoTag),
			expectedCollectorFullRef: defaultCollectorFullImage,
			expectedCollectorSlimRef: defaultCollectorSlimImage,
		},
		"custom main image with custom tag / default collector image": {
			cluster:                  makeTestCluster("quay.io/rhacs/main:custom", defaultCollectorFullImageNoTag),
			expectedMain:             "quay.io/rhacs/main:custom",
			expectedCollectorFullRef: defaultCollectorFullImage,
			expectedCollectorSlimRef: defaultCollectorSlimImage,
		},
		"custom main image / custom collector image: same registry with different namespaces": {
			cluster:                  makeTestCluster("quay.io/namespace-a/main", "quay.io/namespace-b/collector"),
			expectedMain:             fmt.Sprintf("quay.io/namespace-a/main:%s", flavor.MainImageTag),
			expectedCollectorFullRef: fmt.Sprintf("quay.io/namespace-b/collector:%s", flavor.CollectorImageTag),
			expectedCollectorSlimRef: fmt.Sprintf("quay.io/namespace-b/%s:%s", flavor.CollectorSlimImageName, flavor.CollectorSlimImageTag),
		},
		"custom main image with non-default name": {
			cluster:                  makeTestCluster("quay.io/rhacs/customname", ""),
			expectedMain:             fmt.Sprintf("quay.io/rhacs/customname:%s", flavor.MainImageTag),
			expectedCollectorFullRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedCollectorSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorSlimImageName, flavor.CollectorSlimImageTag),
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
			fields, err := FieldsFromClusterAndRenderOpts(c.cluster, &flavor, RenderOptions{})
			if c.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(c.expectedMain, getMain(fields), "Main image does not match")
				s.Equal(c.expectedCollectorFullRef, getCollectorFull(fields), "Collector full image does not match")
				s.Equal(c.expectedCollectorSlimRef, getCollectorSlim(fields), "Collector slim image does not match")
			}
		})
	}
}

func (s *deployerTestSuite) TestFieldsFromClusterAndRenderOpts() {
	flavorCases := map[string]defaults.ImageFlavor{
		"development": defaults.DevelopmentBuildImageFlavor(),
		"stackrox":    defaults.StackRoxIOReleaseImageFlavor(),
	}

	for name, flavor := range flavorCases {
		s.Run(name, func() {
			testMetaValueGenerationWithImageFlavor(s, flavor)
		})
	}

}

func TestRequiredFieldsArePresent(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetExampleVersion(t)

	testFlavor := flavorUtils.MakeImageFlavorForTest(t)
	fields, err := FieldsFromClusterAndRenderOpts(makeTestCluster("docker.io/stackrox/main", ""), &testFlavor, RenderOptions{})
	assert.NoError(t, err)

	assert.NotEmpty(t, fields.MainRegistry)
	assert.NotEmpty(t, fields.ImageRemote)
	assert.NotEmpty(t, fields.CollectorRegistry)
	assert.NotEmpty(t, fields.CollectorFullImageRemote)
	assert.NotEmpty(t, fields.CollectorSlimImageTag)
	assert.NotEmpty(t, fields.CollectorFullImageTag)

	assert.NotEmpty(t, fields.Versions.ChartVersion)
	assert.NotEmpty(t, fields.Versions.MainVersion)
	assert.NotEmpty(t, fields.Versions.CollectorVersion)
	assert.NotEmpty(t, fields.Versions.ScannerVersion)

	assert.NotEmpty(t, fields.ChartRepo.URL)
}
