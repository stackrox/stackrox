package clusters

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func getCollectorImage(fields *charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields.CollectorRegistry, fields.CollectorImageRemote, fields.CollectorImageTag)
}

func getScannerSlim(fields *charts.MetaValues) string {
	return fmt.Sprintf("%s/%s:%s", fields.MainRegistry, fields.ScannerSlimImageRemote, fields.ScannerImageTag)
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
		CollectionMethod:    storage.CollectionMethod_CORE_BPF,
		AdmissionController: false,
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: false,
		},
	}
}

func testMetaValueGenerationWithImageFlavor(s *deployerTestSuite, flavor defaults.ImageFlavor) {
	defaultMainImageNoTag := flavor.MainImageNoTag()
	defaultMainImage := flavor.MainImage()
	defaultCollectorImageNoTag := flavor.CollectorImageNoTag()
	defaultCollectorImage := flavor.CollectorImage()
	defaultScannerSlimImage := flavor.ScannerSlimImage()

	var cases = map[string]struct {
		cluster                *storage.Cluster
		expectedErrorMessage   string
		expectedMain           string
		expectedCollectorRef   string
		expectedScannerSlimRef string
	}{
		// we're testing possible main & collector combinations, grouped by main image setting

		// default main image
		"default main / no collector": {
			cluster:                makeTestCluster(defaultMainImageNoTag, ""),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   defaultCollectorImage,
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main / default collector": {
			cluster:                makeTestCluster(defaultMainImageNoTag, defaultCollectorImageNoTag),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   defaultCollectorImage,
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main / default collector: custom tag": {
			cluster:                makeTestCluster(defaultMainImageNoTag, fmt.Sprintf("%s:custom", defaultCollectorImageNoTag)),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   flavor.CollectorImage(),
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main / custom collector: with namespace": {
			cluster:                makeTestCluster(defaultMainImage, "quay.io/rhacs/collector"),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main / custom collector: with namespace & custom tag": {
			cluster:                makeTestCluster(defaultMainImage, "quay.io/rhacs/collector:custom"),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main / custom collector: without namespace": {
			cluster:                makeTestCluster(defaultMainImage, "example.io/collector"),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   fmt.Sprintf("example.io/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main / custom collector: without namespace & custom tag": {
			cluster:                makeTestCluster(defaultMainImage, "example.io/collector:custom"),
			expectedMain:           defaultMainImage,
			expectedCollectorRef:   fmt.Sprintf("example.io/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main: custom tag / no collector": {
			cluster:                makeTestCluster(fmt.Sprintf("%s:custom", defaultMainImageNoTag), ""),
			expectedMain:           fmt.Sprintf("%s:custom", defaultMainImageNoTag),
			expectedCollectorRef:   defaultCollectorImage,
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main: custom tag / default collector": {
			cluster:                makeTestCluster(fmt.Sprintf("%s:custom", defaultMainImageNoTag), ""),
			expectedMain:           fmt.Sprintf("%s:custom", defaultMainImageNoTag),
			expectedCollectorRef:   defaultCollectorImage,
			expectedScannerSlimRef: defaultScannerSlimImage,
		},
		"default main: custom registry / no collector": {
			cluster:                makeTestCluster("quay.io/rhacs/"+flavor.MainImageName, ""),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.MainImageName, flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},

		// custom main image
		"custom main: with namespace / no collector": {
			cluster:                makeTestCluster("quay.io/rhacs/main", ""),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main: without namespace / no collector": {
			cluster:                makeTestCluster("example.io/main", ""),
			expectedMain:           fmt.Sprintf("example.io/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("example.io/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("example.io/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main / default collector": {
			cluster:                makeTestCluster("quay.io/rhacs/main", defaultCollectorImageNoTag),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   defaultCollectorImage,
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main / custom collector: with namespace": {
			cluster:                makeTestCluster("quay.io/rhacs/main", "quay.io/rhacs/collector"),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main / custom collector: with namespace & custom tag": {
			cluster:                makeTestCluster("quay.io/rhacs/main", "quay.io/rhacs/collector:custom"),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main / custom collector: without namespace": {
			cluster:                makeTestCluster("quay.io/rhacs/main", "example.io/collector"),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("example.io/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main / custom collector: without namespace & custom tag": {
			cluster:                makeTestCluster("quay.io/rhacs/main", "example.io/collector:custom"),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("example.io/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		/*
			// These tests are failing right now but should succeed after ROX-15691 has been implemented
			"custom main: custom tag / no collector": {
				cluster:                  makeTestCluster("quay.io/rhacs/main:custom", ""),
				expectedMain:             fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
				expectedCollectorRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			},
			"custom main: custom tag / default collector": {
				cluster:                  makeTestCluster("quay.io/rhacs/main:custom", defaultCollectorImageNoTag),
				expectedMain:             fmt.Sprintf("quay.io/rhacs/main:%s", flavor.MainImageTag),
				expectedCollectorRef: defaultCollectorImage,
			},
		*/
		"custom main / custom collector: same registry, different namespaces": {
			cluster:                makeTestCluster("quay.io/namespace-a/main", "quay.io/namespace-b/collector"),
			expectedMain:           fmt.Sprintf("quay.io/namespace-a/main:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("quay.io/namespace-b/collector:%s", flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/namespace-a/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		"custom main: non-default name / no collector": {
			cluster:                makeTestCluster("quay.io/rhacs/customname", ""),
			expectedMain:           fmt.Sprintf("quay.io/rhacs/customname:%s", flavor.MainImageTag),
			expectedCollectorRef:   fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.CollectorImageName, flavor.CollectorImageTag),
			expectedScannerSlimRef: fmt.Sprintf("quay.io/rhacs/%s:%s", flavor.ScannerSlimImageName, flavor.ScannerImageTag),
		},
		// Expected fail cases
		"expectedError: empty main image": {
			cluster:              makeTestCluster("", ""),
			expectedErrorMessage: fmt.Sprintf("generating main image from cluster value (%s)", ""),
		},
		"expectedError: invalid main image": {
			cluster:              makeTestCluster("this is not an image #@!", ""),
			expectedErrorMessage: fmt.Sprintf("generating main image from cluster value (%s)", "this is not an image #@!"),
		},
		"expectedError: invalid collector image": {
			cluster:              makeTestCluster("stackrox.io/main", "this is not an image #@!"),
			expectedErrorMessage: fmt.Sprintf("generating collector image from cluster value (%s)", "this is not an image #@!"),
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			fields, err := FieldsFromClusterAndRenderOpts(c.cluster, &flavor, RenderOptions{})
			if len(c.expectedErrorMessage) > 0 {
				s.Error(err)
				s.Contains(err.Error(), c.expectedErrorMessage)
			} else {
				s.NoError(err)
				s.Equal(c.expectedMain, getMain(fields), "Main image does not match")
				s.Equal(c.expectedCollectorRef, getCollectorImage(fields), "Collector image does not match")
				s.Equal(c.expectedScannerSlimRef, getScannerSlim(fields), "Scanner slim image does not match")
			}
		})
	}
}

func testImageFlavorChartRepoSettings(s *deployerTestSuite, flavor defaults.ImageFlavor) {
	cluster := makeTestCluster(flavor.MainImage(), flavor.CollectorImage())

	fields, err := FieldsFromClusterAndRenderOpts(cluster, &flavor, RenderOptions{})
	s.NoError(err)
	s.NotEmpty(fields.ChartRepo.URL, "Chart Repo URL must not be empty")
	s.NotEmpty(fields.ChartRepo.IconURL, "Chart Repo IconURL must not be empty")
	s.Equal(flavor.ChartRepo.URL, fields.ChartRepo.URL, "ChartRepo URL does not match")
	s.Equal(flavor.ChartRepo.IconURL, fields.ChartRepo.IconURL, "ChartRepo IconURL does not match")
}

func (s *deployerTestSuite) TestFieldsFromClusterAndRenderOpts() {
	flavorCases := map[string]defaults.ImageFlavor{
		"development": defaults.DevelopmentBuildImageFlavor(),
		"rhacs":       defaults.RHACSReleaseImageFlavor(),
		"opensource":  defaults.OpenSourceImageFlavor(),
	}

	for name, flavor := range flavorCases {
		s.Run(name, func() {
			testMetaValueGenerationWithImageFlavor(s, flavor)
			testImageFlavorChartRepoSettings(s, flavor)
		})
	}

}

func TestRequiredFieldsArePresent(t *testing.T) {
	testutils.SetExampleVersion(t)

	testFlavor := flavorUtils.MakeImageFlavorForTest(t)
	fields, err := FieldsFromClusterAndRenderOpts(makeTestCluster("docker.io/stackrox/main", ""), &testFlavor, RenderOptions{})
	assert.NoError(t, err)

	assert.NotEmpty(t, fields.MainRegistry)
	assert.NotEmpty(t, fields.ImageRemote)
	assert.NotEmpty(t, fields.CollectorRegistry)
	assert.NotEmpty(t, fields.CollectorImageRemote)
	assert.NotEmpty(t, fields.CollectorImageTag)

	assert.NotEmpty(t, fields.Versions.ChartVersion)
	assert.NotEmpty(t, fields.Versions.MainVersion)
	assert.NotEmpty(t, fields.Versions.CollectorVersion)
	assert.NotEmpty(t, fields.Versions.ScannerVersion)

	assert.NotEmpty(t, fields.ChartRepo.URL)
}
