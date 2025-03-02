package scan

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/registries"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	mirrorStoreMocks "github.com/stackrox/rox/pkg/registrymirror/mocks"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/sensor/common/scannerclient"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
)

var (
	errBroken = errors.New("broken")
)

type fakeImageServiceClient struct {
	v1.ImageServiceClient
	fail bool
	img  *storage.Image
	// Used to check if enrichment on central's side was triggered or not.
	enrichTriggered bool
}

func (i *fakeImageServiceClient) EnrichLocalImageInternal(_ context.Context,
	_ *v1.EnrichLocalImageInternalRequest, _ ...grpc.CallOption) (*v1.ScanImageInternalResponse, error) {
	i.enrichTriggered = true
	if i.fail {
		return nil, errors.New("failed enrichment")
	}
	return &v1.ScanImageInternalResponse{Image: i.img}, nil
}

type echoImageServiceClient struct {
}

// EnrichLocalImageInternal returns an image with values taken from the request (echoes them back).
func (i *echoImageServiceClient) EnrichLocalImageInternal(_ context.Context, req *v1.EnrichLocalImageInternalRequest, _ ...grpc.CallOption) (*v1.ScanImageInternalResponse, error) {
	img := &storage.Image{
		Id:        req.GetImageId(),
		Name:      req.GetImageName(),
		Metadata:  req.GetMetadata(),
		Notes:     req.GetImageNotes(),
		Signature: req.GetImageSignature(),
	}
	return &v1.ScanImageInternalResponse{Image: img}, nil
}

type scanTestSuite struct {
	suite.Suite
}

func TestScanSuite(t *testing.T) {
	suite.Run(t, new(scanTestSuite))
}

func (suite *scanTestSuite) createMockImageServiceClient(img *storage.Image, fail bool) *fakeImageServiceClient {
	return &fakeImageServiceClient{
		fail: fail,
		img:  img,
	}
}

func (suite *scanTestSuite) TestLocalEnrichment() {
	var getRegistriesTriggered bool
	fakeRegStore := &fakeRegistryStore{}
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

	// Use mock functions to avoid having to provide a full registry / scanner.
	scan := LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		getPullSecretRegistries: func(*storage.ImageName, string, []string) ([]registryTypes.ImageRegistry, error) {
			getRegistriesTriggered = true
			return []registryTypes.ImageRegistry{&fakeRegistry{fail: false}}, nil
		},
		getGlobalRegistry: func(*storage.ImageName) (registryTypes.ImageRegistry, error) {
			return &fakeRegistry{fail: false}, nil
		},
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(10),
		getCentralRegistries:   fakeRegStore.GetMatchingCentralRegistryIntegrations,
		mirrorStore:            mirrorStore,
		maxSemaphoreWaitTime:   defaultMaxSemaphoreWaitTime,
	}

	// Original values will be restored within the teardown function. This will be done after each test.

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)

	imageServiceClient := suite.createMockImageServiceClient(img, false)

	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())

	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "fake-namespace", "", false))

	suite.Require().NoError(err, "unexpected error when enriching image")

	protoassert.Equal(suite.T(), img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")

	suite.Assert().True(getRegistriesTriggered, "get registry was not triggered")
}

func (suite *scanTestSuite) TestEnrichImageFailures() {
	type testCase struct {
		scanImg func(ctx context.Context, image *storage.Image,
			registry registryTypes.ImageRegistry, _ scannerclient.ScannerClient) (*scannerclient.ImageAnalysis, error)
		fetchSignaturesWithRetry func(ctx context.Context, fetcher signatures.SignatureFetcher, image *storage.Image,
			fullImageName string, registry registryTypes.Registry) ([]*storage.Signature, error)
		getRegistries          func(image *storage.ImageName, ns string, imagePullSecrets []string) ([]registryTypes.ImageRegistry, error)
		fakeImageServiceClient *fakeImageServiceClient
		enrichmentTriggered    bool
	}

	cases := map[string]testCase{
		"fail retrieving image metadata": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getRegistries: func(*storage.ImageName, string, []string) ([]registryTypes.ImageRegistry, error) {
				return []registryTypes.ImageRegistry{&fakeRegistry{fail: true}}, nil
			},
			enrichmentTriggered: true,
		},
		"fail scanning the image locally": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getRegistries: func(*storage.ImageName, string, []string) ([]registryTypes.ImageRegistry, error) {
				return []registryTypes.ImageRegistry{&fakeRegistry{fail: false}}, nil
			},
			scanImg:             failingScan,
			enrichmentTriggered: true,
		},
		"fail enrich image via central": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, true),
			getRegistries: func(*storage.ImageName, string, []string) ([]registryTypes.ImageRegistry, error) {
				return []registryTypes.ImageRegistry{&fakeRegistry{fail: false}}, nil
			},
			scanImg:                  successfulScan,
			fetchSignaturesWithRetry: successfulFetchSignatures,
			enrichmentTriggered:      true,
		},
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	for name, c := range cases {
		fakeRegStore := &fakeRegistryStore{centralNoRegs: true}
		mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

		suite.Run(name, func() {
			scan := LocalScan{
				scanImg:                  c.scanImg,
				fetchSignaturesWithRetry: c.fetchSignaturesWithRetry,
				getPullSecretRegistries:  c.getRegistries,
				getGlobalRegistry:        emptyGetGlobalRegistryForImage,
				scannerClientSingleton:   emptyScannerClientSingleton,
				scanSemaphore:            semaphore.NewWeighted(10),
				getCentralRegistries:     fakeRegStore.GetMatchingCentralRegistryIntegrations,
				mirrorStore:              mirrorStore,
				maxSemaphoreWaitTime:     defaultMaxSemaphoreWaitTime,
			}
			mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())
			img, err := scan.EnrichLocalImageInNamespace(context.Background(), c.fakeImageServiceClient, genScanReq(containerImg, "fake-namespace", "", false))
			suite.Assert().Error(err, "expected an error")
			suite.Assert().Nil(img, "required an empty image")
			suite.Assert().Equal(c.enrichmentTriggered, c.fakeImageServiceClient.enrichTriggered,
				"expected enrichment trigger status to be as expected")
		})
	}
}

func (suite *scanTestSuite) TestMetadataBeingSet() {
	fakeRegStore := &fakeRegistryStore{}
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

	scan := LocalScan{
		scanImg: successfulScan,
		fetchSignaturesWithRetry: func(_ context.Context, _ signatures.SignatureFetcher, img *storage.Image, _ string,
			_ registryTypes.Registry) ([]*storage.Signature, error) {
			if img.GetMetadata().GetV2() == nil {
				return nil, errors.New("image metadata missing, not attempting fetch of signatures")
			}
			return nil, nil
		},
		getGlobalRegistry: emptyGetGlobalRegistryForImage,
		getPullSecretRegistries: func(image *storage.ImageName, ns string, imagePullSecrets []string) ([]registryTypes.ImageRegistry, error) {
			return []registryTypes.ImageRegistry{&fakeRegistry{fail: false}}, nil
		},
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(10),
		getCentralRegistries:   fakeRegStore.GetMatchingCentralRegistryIntegrations,
		mirrorStore:            mirrorStore,
		maxSemaphoreWaitTime:   defaultMaxSemaphoreWaitTime,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())

	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "fake-namespace", "", false))

	suite.Require().NoError(err, "unexpected error when enriching image")

	protoassert.Equal(suite.T(), img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")
}

func (suite *scanTestSuite) TestEnrichLocalImageInNamespace() {
	fakeRegStore := &fakeRegistryStore{}
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

	scan := LocalScan{
		scanImg:                   successfulScan,
		fetchSignaturesWithRetry:  successfulFetchSignatures,
		getPullSecretRegistries:   fakeRegStore.GetRegistries,
		getGlobalRegistry:         fakeRegStore.GetGlobalRegistryForImage,
		scannerClientSingleton:    emptyScannerClientSingleton,
		scanSemaphore:             semaphore.NewWeighted(10),
		createNoAuthImageRegistry: successCreateNoAuthImageRegistry,
		getCentralRegistries:      fakeRegStore.GetMatchingCentralRegistryIntegrations,
		mirrorStore:               mirrorStore,
		maxSemaphoreWaitTime:      defaultMaxSemaphoreWaitTime,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	// an empty namespace should not trigger namespace specific regStore methods
	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())
	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
	suite.Require().NoError(err)
	protoassert.Equal(suite.T(), img, resultImg)
	suite.Assert().True(imageServiceClient.enrichTriggered)
	suite.Assert().True(fakeRegStore.getMatchingCentralRegistryIntegrationsInvoked)
	suite.Assert().False(fakeRegStore.getRegistryForImageInNamespaceInvoked)
	suite.Assert().True(fakeRegStore.getGlobalRegistryForImageInvoked)

	// non-openshift namespaces should not invoke getGlobalRegistryForImage
	namespace := "fake-namespace"
	imageServiceClient.enrichTriggered = false
	fakeRegStore.getRegistryForImageInNamespaceInvoked = false
	fakeRegStore.getGlobalRegistryForImageInvoked = false
	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())
	resultImg, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, namespace, "", false))
	suite.Require().NoError(err)
	protoassert.Equal(suite.T(), img, resultImg)
	suite.Assert().True(imageServiceClient.enrichTriggered)
	suite.Assert().True(fakeRegStore.getMatchingCentralRegistryIntegrationsInvoked)
	suite.Assert().True(fakeRegStore.getRegistryForImageInNamespaceInvoked)
	suite.Assert().True(fakeRegStore.getGlobalRegistryForImageInvoked)
}

func (suite *scanTestSuite) TestEnrichErrorNoScanner() {
	scan := LocalScan{
		scannerClientSingleton: func() scannerclient.ScannerClient { return nil },
	}

	img := &storage.ContainerImage{Name: &storage.ImageName{Registry: "fake"}}
	_, err := scan.EnrichLocalImageInNamespace(context.Background(), nil, genScanReq(img, "", "", false))
	suite.Require().ErrorIs(err, ErrNoLocalScanner)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestEnrichErrorNoImage() {
	scan := LocalScan{
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(10),
	}

	_, err := scan.EnrichLocalImageInNamespace(context.Background(), nil, genScanReq(nil, "", "", false))
	suite.Require().Error(err)
	suite.Require().NotErrorIs(err, ErrNoLocalScanner)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestEnrichErrorBadImage() {
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))
	imageServiceClient := &echoImageServiceClient{}
	scan := LocalScan{
		scannerClientSingleton:    emptyScannerClientSingleton,
		scanSemaphore:             semaphore.NewWeighted(10),
		getCentralRegistries:      emptyGetMatchingCentralIntegrations,
		mirrorStore:               mirrorStore,
		getGlobalRegistry:         emptyGetGlobalRegistryForImage,
		createNoAuthImageRegistry: failCreateNoAuthImageRegistry,
		scanImg:                   scanImage,
		maxSemaphoreWaitTime:      defaultMaxSemaphoreWaitTime,
	}

	imgNameStr := "   is an invalid image"

	suite.Run("enrich error on nil image name", func() {
		containerImg := &storage.ContainerImage{}
		resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().ErrorContains(err, "missing image name")
		suite.Require().ErrorIs(err, ErrEnrichNotStarted)
		suite.Require().Nil(resultImg)

	})

	suite.Run("enrich error missing image registry", func() {
		containerImg := &storage.ContainerImage{
			Name: &storage.ImageName{},
		}
		resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().ErrorContains(err, "missing image registry")
		suite.Require().ErrorIs(err, ErrEnrichNotStarted)
		suite.Require().Nil(resultImg)

	})

	suite.Run("enrich error on bad full image name", func() {
		containerImg := &storage.ContainerImage{
			Name: &storage.ImageName{
				Registry: "fake",
				FullName: imgNameStr,
			},
		}
		mirrorStore.EXPECT().PullSources(gomock.Any()).Return([]string{imgNameStr}, nil)
		resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().ErrorContains(err, "zero valid pull sources")
		suite.Require().ErrorIs(err, ErrEnrichNotStarted)
		suite.Require().Nil(resultImg)
	})
}

func (suite *scanTestSuite) TestEnrichThrottle() {
	scan := LocalScan{
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(0),
		maxSemaphoreWaitTime:   1 * time.Millisecond,
	}

	img := &storage.ContainerImage{Name: &storage.ImageName{Registry: "fake"}}
	_, err := scan.EnrichLocalImageInNamespace(context.Background(), nil, genScanReq(img, "", "", false))
	suite.Require().ErrorIs(err, ErrTooManyParallelScans)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestAdHocScanThrottle() {
	ls := LocalScan{
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(5),
		adHocScanSemaphore:     semaphore.NewWeighted(0),
		maxSemaphoreWaitTime:   1 * time.Millisecond,
	}

	img := &storage.ContainerImage{Name: &storage.ImageName{Registry: "fake"}}
	req := genScanReq(img, "", "some-id", false) // "makes it an ad-hoc delegated request

	_, err := ls.EnrichLocalImageInNamespace(context.Background(), nil, req)
	suite.Require().ErrorIs(err, ErrTooManyParallelScans)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestEnrichMultipleRegistries() {
	reg1 := &fakeRegistry{fail: true}
	reg2 := &fakeRegistry{}
	reg3 := &fakeRegistry{}
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

	scan := &LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		scannerClientSingleton:   emptyScannerClientSingleton,
		scanSemaphore:            semaphore.NewWeighted(10),
		getGlobalRegistry:        emptyGetGlobalRegistryForImage,
		mirrorStore:              mirrorStore,
		getCentralRegistries: func(in *storage.ImageName) []registryTypes.ImageRegistry {
			return []registryTypes.ImageRegistry{reg1, reg2}
		},
		maxSemaphoreWaitTime: defaultMaxSemaphoreWaitTime,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")
	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	// reg1 metadata should fail and not be used for scanning
	// reg2 metadata should succeed and be used for scanning
	// reg3 metadata should have never been invoked because reg2 succeeded
	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())
	_, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
	suite.Require().NoError(err)
	suite.Require().True(reg1.metadataInvoked)
	suite.Require().False(reg1.usedForScan)

	suite.Require().True(reg2.metadataInvoked)
	suite.Require().True(reg2.usedForScan)

	suite.Require().False(reg3.metadataInvoked)
	suite.Require().False(reg3.usedForScan)
}

func (suite *scanTestSuite) TestEnrichNoRegistries() {
	fakeRegStore := &fakeRegistryStore{}
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

	var createNoAuthRegistryInvoked bool
	scan := LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		getPullSecretRegistries:  fakeRegStore.GetRegistries,
		scannerClientSingleton:   emptyScannerClientSingleton,
		scanSemaphore:            semaphore.NewWeighted(10),
		mirrorStore:              mirrorStore,
		createNoAuthImageRegistry: func(ctx context.Context, in *storage.ImageName, f registries.Factory) (registryTypes.ImageRegistry, error) {
			createNoAuthRegistryInvoked = true
			return &fakeRegistry{}, nil
		},
		getCentralRegistries: emptyGetMatchingCentralIntegrations,
		getGlobalRegistry:    emptyGetGlobalRegistryForImage,
		maxSemaphoreWaitTime: defaultMaxSemaphoreWaitTime,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())
	_, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
	suite.Require().NoError(err, "unexpected error enriching image")
	suite.Require().False(fakeRegStore.getRegistryForImageInNamespaceInvoked)
	suite.Require().True(createNoAuthRegistryInvoked)
}

func (suite *scanTestSuite) TestEnrichNoRegistriesFailure() {
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))
	scan := LocalScan{
		scannerClientSingleton:    emptyScannerClientSingleton,
		scanSemaphore:             semaphore.NewWeighted(10),
		getCentralRegistries:      emptyGetMatchingCentralIntegrations,
		getGlobalRegistry:         emptyGetGlobalRegistryForImage,
		createNoAuthImageRegistry: failCreateNoAuthImageRegistry,
		mirrorStore:               mirrorStore,
		maxSemaphoreWaitTime:      defaultMaxSemaphoreWaitTime,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	mirrorStore.EXPECT().PullSources(containerImg.GetName().GetFullName())
	_, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
	suite.Require().ErrorContains(err, "unable to create no auth integration")
	suite.Require().ErrorContains(err, errBroken.Error())
}

func (suite *scanTestSuite) TestGetImageRegistries() {
	reg1 := &fakeRegistry{fail: true}
	reg2 := &fakeRegistry{}
	reg3 := &fakeRegistry{}
	reg4 := &fakeRegistry{}
	reg5 := &fakeRegistry{}
	expected := []registryTypes.ImageRegistry{reg1, reg2, reg3, reg4}

	scan := &LocalScan{
		getCentralRegistries: func(in *storage.ImageName) []registryTypes.ImageRegistry {
			return []registryTypes.ImageRegistry{reg1, reg2}
		},
		getPullSecretRegistries: func(in *storage.ImageName, s string, imagePullSecrets []string) ([]registryTypes.ImageRegistry, error) {
			return []registryTypes.ImageRegistry{reg3}, nil
		},
		getGlobalRegistry: func(in *storage.ImageName) (registryTypes.ImageRegistry, error) {
			return reg4, nil
		},
		createNoAuthImageRegistry: func(ctx context.Context, in *storage.ImageName, f registries.Factory) (registryTypes.ImageRegistry, error) {
			return reg5, nil
		},
	}

	regs, err := scan.getRegistries(context.Background(), "fake-namespace", nil, nil)
	suite.Require().NoError(err)
	suite.Assert().Len(regs, 4)
	suite.Assert().Equal(expected[:4], regs)

	// with no namespace, reg3 should not be returned
	regs, err = scan.getRegistries(context.Background(), "", nil, nil)
	suite.Require().NoError(err)
	suite.Assert().Len(regs, 3)
	suite.Assert().Equal([]registryTypes.ImageRegistry{reg1, reg2, reg4}, regs)
}

func (suite *scanTestSuite) TestMultiplePullSources() {
	mirror1 := "example.com/mirror1:latest"
	mirror2 := "example.com/mirror2:latest"
	source := "example.com/source:latest"
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))

	var mirror2ScanTriggered bool
	scan := LocalScan{
		scanImg: func(ctx context.Context, i *storage.Image, ir registryTypes.ImageRegistry, c scannerclient.ScannerClient) (*scannerclient.ImageAnalysis, error) {
			if i.GetName().GetFullName() == mirror2 {
				mirror2ScanTriggered = true
			}
			return successfulScan(ctx, i, ir, c)
		},
		fetchSignaturesWithRetry: successfulFetchSignatures,
		scannerClientSingleton:   emptyScannerClientSingleton,
		scanSemaphore:            semaphore.NewWeighted(10),
		mirrorStore:              mirrorStore,
		getCentralRegistries:     emptyGetMatchingCentralIntegrations,
		getGlobalRegistry: func(in *storage.ImageName) (registryTypes.ImageRegistry, error) {
			if in.GetFullName() == mirror1 {
				return &fakeRegistry{fail: true}, nil
			}

			return &fakeRegistry{}, nil
		},
		maxSemaphoreWaitTime: defaultMaxSemaphoreWaitTime,
	}

	containerImg, err := utils.GenerateImageFromString(source)
	suite.Require().NoError(err, "failed creating test image")

	imageServiceClient := &echoImageServiceClient{}

	mirrorStore.EXPECT().PullSources(gomock.Any()).Return([]string{mirror1, mirror2, source}, nil)

	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
	suite.Require().NoError(err)

	// image.Name should represent image from k8s podspec (not the mirror).
	suite.Assert().Equal(source, resultImg.GetName().GetFullName())

	// The datasource should represent the mirror.
	suite.Assert().Equal(mirror2, resultImg.GetMetadata().GetDataSource().GetMirror())

	// A scan should have occurred via the 2nd mirror.
	suite.Assert().True(mirror2ScanTriggered, "scan should have been triggered using mirror2")
}

// TestGetPullSourceIDKeep ensures that an image ID is not lost when pull sources / mirrors
// are processed. The ID is used when pulling image metadata and layers, if the ID is lost
// then incorrect metadata or layers may be pulled when image content changes in the
// registry and an image is referenced by tag.
func (suite *scanTestSuite) TestGetPullSourceIDKeep() {
	imgStr := "reg.invalid/repo:tag"
	imgID := "sha256:fake"

	srcImg, err := utils.GenerateImageFromString(imgStr)
	suite.Require().NoError(err)
	srcImg.Id = imgID // Simulate an image ID being set on an image after it has been observed running.

	suite.Run("Ensure image ID is not lost when pull sources not found", func() {
		mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))
		scan := LocalScan{mirrorStore: mirrorStore}

		mirrorStore.EXPECT().PullSources(gomock.Any()).Return(nil, fs.ErrNotExist)

		pullSrcs := scan.getPullSources(srcImg)
		suite.Require().Len(pullSrcs, 1)
		suite.Equal(imgID, pullSrcs[0].GetId())
	})

	suite.Run("Ensure image ID is not lost when mirroring", func() {
		mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))
		scan := LocalScan{mirrorStore: mirrorStore}

		srcImg, err := utils.GenerateImageFromString(imgStr)
		suite.Require().NoError(err)
		srcImg.Id = imgID

		fakeDigest := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		mirrorStore.EXPECT().PullSources(gomock.Any()).Return([]string{
			"reg.mirror1.invalid/repo:tag",
			// This next returned string should never happen, a pull source would
			// never be returned from the mirror store with a different digest
			// than the source image. Also technically the returned mirrors would
			// not contain a mix of tags and digests like what this test shows, for
			// testing purposes this is OK because we are testing how the response
			// from the mirror store is handled, not the mirror store itself.
			"reg.mirror2.invalid/repo@" + fakeDigest,
			imgStr,
		}, nil)

		pullSrcs := scan.getPullSources(srcImg)
		suite.Require().Len(pullSrcs, 3)

		suite.Equal("reg.mirror1.invalid", pullSrcs[0].GetName().GetRegistry())
		suite.Equal(srcImg.GetName().GetTag(), pullSrcs[0].GetName().GetTag())
		suite.Equal(imgID, pullSrcs[0].GetId())

		suite.Equal("reg.mirror2.invalid", pullSrcs[1].GetName().GetRegistry())
		suite.Equal(fakeDigest, pullSrcs[1].GetId())

		protoassert.Equal(suite.T(), srcImg, pullSrcs[2], "last mirror is expected to be identical to src image")
	})
}

func (suite *scanTestSuite) TestNotes() {
	mirrorStore := mirrorStoreMocks.NewMockStore(gomock.NewController(suite.T()))
	mirrorStore.EXPECT().PullSources(gomock.Any()).AnyTimes()

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	imageServiceClient := &echoImageServiceClient{}

	scan := LocalScan{
		scannerClientSingleton:    emptyScannerClientSingleton,
		scanSemaphore:             semaphore.NewWeighted(10),
		getCentralRegistries:      emptyGetMatchingCentralIntegrations,
		mirrorStore:               mirrorStore,
		getGlobalRegistry:         emptyGetGlobalRegistryForImage,
		createNoAuthImageRegistry: failCreateNoAuthImageRegistry,
		maxSemaphoreWaitTime:      defaultMaxSemaphoreWaitTime,
	}

	suite.Run("missing metadata", func() {
		resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().Error(err)
		suite.Require().Contains(resultImg.GetNotes(), storage.Image_MISSING_METADATA)
	})

	scan.createNoAuthImageRegistry = successCreateNoAuthImageRegistry
	scan.scanImg = failingScan
	suite.Run("missing scan data", func() {
		resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().Error(err)
		suite.Require().Contains(resultImg.GetNotes(), storage.Image_MISSING_SCAN_DATA)
	})

	scan.scanImg = successfulScan
	suite.Run("missing sigs", func() {
		scan.fetchSignaturesWithRetry = failingFetchSignatures
		resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().NoError(err)
		suite.Require().Contains(resultImg.GetNotes(), storage.Image_MISSING_SIGNATURE)

		scan.fetchSignaturesWithRetry = failingFetchSignaturesUnauthorized
		resultImg, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, genScanReq(containerImg, "", "", false))
		suite.Require().NoError(err)
		suite.Require().Contains(resultImg.GetNotes(), storage.Image_MISSING_SIGNATURE)
	})
}

func successfulScan(_ context.Context, _ *storage.Image,
	reg registryTypes.ImageRegistry, _ scannerclient.ScannerClient) (*scannerclient.ImageAnalysis, error) {

	if reg != nil {
		if r, ok := reg.(*fakeRegistry); ok {
			r.usedForScan = true
		}
	}
	return &scannerclient.ImageAnalysis{
		ScanStatus: scannerV1.ScanStatus_SUCCEEDED,
		V1Components: &scannerV1.Components{
			Namespace: "default",
			LanguageComponents: []*scannerV1.LanguageComponent{{
				Type:    scannerV1.SourceType_JAVA,
				Name:    "log4j",
				Version: "1.0",
			}},
		},
	}, nil
}

func successfulFetchSignatures(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image, _ string,
	_ registryTypes.Registry) ([]*storage.Signature, error) {
	return []*storage.Signature{{
		Signature: &storage.Signature_Cosign{Cosign: &storage.CosignSignature{
			RawSignature:     []byte("some-signature"),
			SignaturePayload: []byte("some-payload"),
		}},
	}}, nil
}

func failingScan(_ context.Context, _ *storage.Image,
	_ registryTypes.ImageRegistry, _ scannerclient.ScannerClient) (*scannerclient.ImageAnalysis, error) {
	return nil, errors.New("failed scanning image")
}

func failingFetchSignatures(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image, _ string,
	_ registryTypes.Registry) ([]*storage.Signature, error) {
	return nil, errors.New("failed fetching signatures")
}

func failingFetchSignaturesUnauthorized(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image, _ string,
	_ registryTypes.Registry) ([]*storage.Signature, error) {
	return nil, errox.NotAuthorized
}

type emptyClient struct {
}

func (*emptyClient) Close() error {
	return nil
}

func (*emptyClient) GetImageAnalysis(_ context.Context, _ *storage.Image, _ *registryTypes.Config) (*scannerclient.ImageAnalysis, error) {
	return nil, nil
}

func emptyScannerClientSingleton() scannerclient.ScannerClient {
	return &emptyClient{}
}

func emptyGetGlobalRegistryForImage(*storage.ImageName) (registryTypes.ImageRegistry, error) {
	return nil, errors.New("no registry found")
}

func emptyGetMatchingCentralIntegrations(_ *storage.ImageName) []registryTypes.ImageRegistry {
	return nil
}

func successCreateNoAuthImageRegistry(context.Context, *storage.ImageName, registries.Factory) (registryTypes.ImageRegistry, error) {
	return &fakeRegistry{}, nil
}

func failCreateNoAuthImageRegistry(context.Context, *storage.ImageName, registries.Factory) (registryTypes.ImageRegistry, error) {
	return nil, errBroken
}

type fakeRegistry struct {
	metadataInvoked bool
	usedForScan     bool
	registryTypes.Registry
	fail bool
}

func (f *fakeRegistry) Metadata(_ *storage.Image) (*storage.ImageMetadata, error) {
	f.metadataInvoked = true
	if f.fail {
		return nil, errors.New("failed fetching metadata")
	}
	return &storage.ImageMetadata{V2: &storage.V2Metadata{Digest: "sha256:XYZ"}}, nil
}

func (f *fakeRegistry) Config(_ context.Context) *registryTypes.Config {
	return nil
}

func (f *fakeRegistry) Name() string {
	return "testing registry"
}

func (f *fakeRegistry) DataSource() *storage.DataSource {
	return &storage.DataSource{}
}

func (f *fakeRegistry) Source() *storage.ImageIntegration {
	return nil
}

type fakeRegistryStore struct {
	getGlobalRegistryForImageInvoked              bool
	getRegistryForImageInNamespaceInvoked         bool
	getMatchingCentralRegistryIntegrationsInvoked bool

	globalReg    registryTypes.ImageRegistry
	namespaceReg registryTypes.ImageRegistry
	centralRegs  []registryTypes.ImageRegistry

	globalNoRegs    bool
	centralNoRegs   bool
	namespaceNoRegs bool
}

func (f *fakeRegistryStore) GetRegistries(_ *storage.ImageName, _ string, _ []string) ([]registryTypes.ImageRegistry, error) {
	f.getRegistryForImageInNamespaceInvoked = true
	if f.namespaceReg != nil {
		return []registryTypes.ImageRegistry{f.namespaceReg}, nil
	}
	if f.namespaceNoRegs {
		return nil, errors.New("no regs")
	}
	return []registryTypes.ImageRegistry{&fakeRegistry{}}, nil
}

func (f *fakeRegistryStore) GetGlobalRegistryForImage(*storage.ImageName) (registryTypes.ImageRegistry, error) {
	f.getGlobalRegistryForImageInvoked = true
	if f.globalReg != nil {
		return f.globalReg, nil
	}
	if f.globalNoRegs {
		return nil, errors.New("no regs")
	}
	return &fakeRegistry{}, nil
}

func (f *fakeRegistryStore) GetMatchingCentralRegistryIntegrations(*storage.ImageName) []registryTypes.ImageRegistry {
	f.getMatchingCentralRegistryIntegrationsInvoked = true
	if f.centralRegs != nil {
		return f.centralRegs
	}
	if f.centralNoRegs {
		return nil
	}
	return []registryTypes.ImageRegistry{&fakeRegistry{}}
}

func genScanReq(img *storage.ContainerImage, namespace, reqID string, force bool) *LocalScanRequest {
	return &LocalScanRequest{
		ID:        reqID,
		Image:     img,
		Namespace: namespace,
		Force:     force,
	}
}
