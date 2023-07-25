package scan

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	scannerV4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/registries"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/sensor/common/scannerclient"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
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
	var getRegistryForImageInNamespaceTriggered bool
	fakeRegStore := &fakeRegistryStore{}

	// Use mock functions to avoid having to provide a full registry / scanner.
	scan := LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		getRegistryForImageInNamespace: func(*storage.ImageName, string) (registryTypes.ImageRegistry, error) {
			getRegistryForImageInNamespaceTriggered = true
			return &fakeRegistry{fail: false}, nil
		},
		getGlobalRegistryForImage: func(*storage.ImageName) (registryTypes.ImageRegistry, error) {
			return nil, nil
		},
		scannerClientSingleton:            emptyScannerClientSingleton,
		scanSemaphore:                     semaphore.NewWeighted(10),
		getMatchingCentralRegIntegrations: fakeRegStore.GetMatchingCentralRegistryIntegrations,
	}

	// Original values will be restored within the teardown function. This will be done after each test.

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)

	imageServiceClient := suite.createMockImageServiceClient(img, false)

	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, containerImg, "fake-namespace", "", false)

	suite.Require().NoError(err, "unexpected error when enriching image")

	suite.Assert().Equal(img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")

	suite.Assert().True(getRegistryForImageInNamespaceTriggered, "get registry was not triggered")
}

func (suite *scanTestSuite) TestEnrichImageFailures() {
	type testCase struct {
		scanImg func(ctx context.Context, image *storage.Image,
			registry registryTypes.ImageRegistry, _ scannerclient.Client) (*scannerV1.GetImageComponentsResponse, *scannerV4.IndexReport, error)
		fetchSignaturesWithRetry func(ctx context.Context, fetcher signatures.SignatureFetcher, image *storage.Image,
			fullImageName string, registry registryTypes.Registry) ([]*storage.Signature, error)
		getRegistryForImageInNamespace func(image *storage.ImageName, ns string) (registryTypes.ImageRegistry, error)
		fakeImageServiceClient         *fakeImageServiceClient
		enrichmentTriggered            bool
	}

	cases := map[string]testCase{
		"fail retrieving image metadata": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getRegistryForImageInNamespace: func(*storage.ImageName, string) (registryTypes.ImageRegistry, error) {
				return &fakeRegistry{fail: true}, nil
			},
			enrichmentTriggered: true,
		},
		"fail scanning the image locally": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getRegistryForImageInNamespace: func(*storage.ImageName, string) (registryTypes.ImageRegistry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:             failingScan,
			enrichmentTriggered: true,
		},
		"fail enrich image via central": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, true),
			getRegistryForImageInNamespace: func(*storage.ImageName, string) (registryTypes.ImageRegistry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:                  successfulScan,
			fetchSignaturesWithRetry: successfulFetchSignatures,
			enrichmentTriggered:      true,
		},
		"fail fetching signatures": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getRegistryForImageInNamespace: func(*storage.ImageName, string) (registryTypes.ImageRegistry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:                  successfulScan,
			fetchSignaturesWithRetry: failingFetchSignatures,
			enrichmentTriggered:      true,
		},
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	for name, c := range cases {
		fakeRegStore := &fakeRegistryStore{centralNoRegs: true}
		suite.Run(name, func() {
			scan := LocalScan{
				scanImg:                           c.scanImg,
				fetchSignaturesWithRetry:          c.fetchSignaturesWithRetry,
				getRegistryForImageInNamespace:    c.getRegistryForImageInNamespace,
				getGlobalRegistryForImage:         emptyGetGlobalRegistryForImage,
				scannerClientSingleton:            emptyScannerClientSingleton,
				scanSemaphore:                     semaphore.NewWeighted(10),
				getMatchingCentralRegIntegrations: fakeRegStore.GetMatchingCentralRegistryIntegrations,
			}
			img, err := scan.EnrichLocalImageInNamespace(context.Background(), c.fakeImageServiceClient, containerImg, "fake-namespace", "", false)
			suite.Assert().Error(err, "expected an error")
			suite.Assert().Nil(img, "required an empty image")
			suite.Assert().Equal(c.enrichmentTriggered, c.fakeImageServiceClient.enrichTriggered,
				"expected enrichment trigger status to be as expected")
		})
	}
}

func (suite *scanTestSuite) TestMetadataBeingSet() {
	fakeRegStore := &fakeRegistryStore{}

	scan := LocalScan{
		scanImg: successfulScan,
		fetchSignaturesWithRetry: func(_ context.Context, _ signatures.SignatureFetcher, img *storage.Image, _ string,
			_ registryTypes.Registry) ([]*storage.Signature, error) {
			if img.GetMetadata().GetV2() == nil {
				return nil, errors.New("image metadata missing, not attempting fetch of signatures")
			}
			return nil, nil
		},
		getGlobalRegistryForImage: emptyGetGlobalRegistryForImage,
		getRegistryForImageInNamespace: func(image *storage.ImageName, ns string) (registryTypes.ImageRegistry, error) {
			return &fakeRegistry{fail: false}, nil
		},
		scannerClientSingleton:            emptyScannerClientSingleton,
		scanSemaphore:                     semaphore.NewWeighted(10),
		getMatchingCentralRegIntegrations: fakeRegStore.GetMatchingCentralRegistryIntegrations,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)
	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, containerImg, "fake-namespace", "", false)

	suite.Require().NoError(err, "unexpected error when enriching image")

	suite.Assert().Equal(img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")
}

func (suite *scanTestSuite) TestEnrichLocalImageInNamespace() {
	fakeRegStore := &fakeRegistryStore{}

	scan := LocalScan{
		scanImg:                           successfulScan,
		fetchSignaturesWithRetry:          successfulFetchSignatures,
		getRegistryForImageInNamespace:    fakeRegStore.GetRegistryForImageInNamespace,
		getGlobalRegistryForImage:         fakeRegStore.GetGlobalRegistryForImage,
		scannerClientSingleton:            emptyScannerClientSingleton,
		scanSemaphore:                     semaphore.NewWeighted(10),
		createNoAuthImageRegistry:         successCreateNoAuthImageRegistry,
		getMatchingCentralRegIntegrations: fakeRegStore.GetMatchingCentralRegistryIntegrations,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	// an empty namespace should not trigger namespace specific regStore methods
	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, containerImg, "", "", false)
	suite.Require().NoError(err)
	suite.Assert().Equal(img, resultImg)
	suite.Assert().True(imageServiceClient.enrichTriggered)
	suite.Assert().True(fakeRegStore.getMatchingCentralRegistryIntegrationsInvoked)
	suite.Assert().False(fakeRegStore.getRegistryForImageInNamespaceInvoked)
	suite.Assert().True(fakeRegStore.getGlobalRegistryForImageInvoked)

	// non-openshift namespaces should not invoke getGlobalRegistryForImage
	namespace := "fake-namespace"
	imageServiceClient.enrichTriggered = false
	fakeRegStore.getRegistryForImageInNamespaceInvoked = false
	fakeRegStore.getGlobalRegistryForImageInvoked = false
	resultImg, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, containerImg, namespace, "", false)
	suite.Require().NoError(err)
	suite.Assert().Equal(img, resultImg)
	suite.Assert().True(imageServiceClient.enrichTriggered)
	suite.Assert().True(fakeRegStore.getMatchingCentralRegistryIntegrationsInvoked)
	suite.Assert().True(fakeRegStore.getRegistryForImageInNamespaceInvoked)
	suite.Assert().True(fakeRegStore.getGlobalRegistryForImageInvoked)
}

func (suite *scanTestSuite) TestEnrichErrorNoScanner() {
	scan := LocalScan{
		scannerClientSingleton: func() scannerclient.Client { return nil },
	}

	_, err := scan.enrichLocalImageFromRegistry(context.Background(), nil, &storage.ContainerImage{}, nil, "", false)
	suite.Require().ErrorIs(err, ErrNoLocalScanner)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestEnrichErrorNoImage() {
	scan := LocalScan{
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(10),
	}

	_, err := scan.enrichLocalImageFromRegistry(context.Background(), nil, nil, nil, "", false)
	suite.Require().Error(err)
	suite.Require().NotErrorIs(err, ErrNoLocalScanner)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestEnrichThrottle() {
	scan := LocalScan{
		scannerClientSingleton: emptyScannerClientSingleton,
		scanSemaphore:          semaphore.NewWeighted(0),
	}

	_, err := scan.enrichLocalImageFromRegistry(context.Background(), nil, &storage.ContainerImage{}, nil, "", false)
	suite.Require().ErrorIs(err, ErrTooManyParallelScans)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
}

func (suite *scanTestSuite) TestEnrichMultipleRegistries() {
	reg1 := &fakeRegistry{fail: true}
	reg2 := &fakeRegistry{}
	reg3 := &fakeRegistry{}
	regs := []registryTypes.ImageRegistry{reg1, reg2, reg3}

	scan := &LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		scannerClientSingleton:   emptyScannerClientSingleton,
		scanSemaphore:            semaphore.NewWeighted(10),
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")
	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	// reg1 metadata should fail and not be used for scanning
	// reg2 metadata should succeed and be used for scanning
	// reg3 metadata should have never been invoked because reg2 succeeded
	_, err = scan.enrichLocalImageFromRegistry(context.Background(), imageServiceClient, containerImg, regs, "", false)
	suite.Require().NoError(err)
	suite.Require().True(reg1.metadataInvoked)
	suite.Require().False(reg1.configInvoked)

	suite.Require().True(reg2.metadataInvoked)
	suite.Require().True(reg2.configInvoked)

	suite.Require().False(reg3.metadataInvoked)
	suite.Require().False(reg3.configInvoked)
}

func (suite *scanTestSuite) TestEnrichNoRegistries() {
	fakeRegStore := &fakeRegistryStore{}

	scan := LocalScan{
		scanImg:                        successfulScan,
		fetchSignaturesWithRetry:       successfulFetchSignatures,
		getRegistryForImageInNamespace: fakeRegStore.GetRegistryForImageInNamespace,
		getGlobalRegistryForImage:      fakeRegStore.GetGlobalRegistryForImage,
		scannerClientSingleton:         emptyScannerClientSingleton,
		scanSemaphore:                  semaphore.NewWeighted(10),
		createNoAuthImageRegistry:      successCreateNoAuthImageRegistry,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	_, err = scan.enrichLocalImageFromRegistry(context.Background(), imageServiceClient, containerImg, nil, "", false)
	suite.Require().NoError(err, "unexpected error enriching image")
	suite.Require().False(fakeRegStore.getGlobalRegistryForImageInvoked)
	suite.Require().False(fakeRegStore.getRegistryForImageInNamespaceInvoked)
	suite.Require().False(fakeRegStore.getMatchingCentralRegistryIntegrationsInvoked)
}

func (suite *scanTestSuite) TestEnrichNoRegistriesFailure() {
	scan := LocalScan{
		scannerClientSingleton:    emptyScannerClientSingleton,
		scanSemaphore:             semaphore.NewWeighted(10),
		createNoAuthImageRegistry: failCreateNoAuthImageRegistry,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	_, err = scan.enrichLocalImageFromRegistry(context.Background(), imageServiceClient, containerImg, nil, "", false)
	suite.Require().ErrorIs(err, ErrEnrichNotStarted)
	suite.Require().ErrorContains(err, "unable to create no auth registry")
}

func (suite *scanTestSuite) TestGetRegistries() {
	scan := &LocalScan{}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	setup := func(regStore *fakeRegistryStore) {
		scan.getGlobalRegistryForImage = regStore.GetGlobalRegistryForImage
		scan.getMatchingCentralRegIntegrations = regStore.GetMatchingCentralRegistryIntegrations
		scan.getRegistryForImageInNamespace = regStore.GetRegistryForImageInNamespace
	}

	suite.Run("no regs", func() {
		regStore := &fakeRegistryStore{
			namespaceNoRegs: true,
			globalNoRegs:    true,
			centralNoRegs:   true,
		}
		setup(regStore)

		regs := scan.getRegistries("fake", containerImg.GetName())
		suite.Len(regs, 0)
	})

	suite.Run("namespaces regs", func() {
		regStore := &fakeRegistryStore{
			namespaceNoRegs: false,
			globalNoRegs:    true,
			centralNoRegs:   true,
		}
		setup(regStore)

		regs := scan.getRegistries("", containerImg.GetName())
		suite.Len(regs, 0)
		suite.False(regStore.getRegistryForImageInNamespaceInvoked)

		regs = scan.getRegistries("fake", containerImg.GetName())
		suite.Len(regs, 1)
		suite.True(regStore.getRegistryForImageInNamespaceInvoked)
	})

	suite.Run("regs from all", func() {
		regStore := &fakeRegistryStore{
			globalReg:    &fakeRegistry{},
			namespaceReg: &fakeRegistry{},
			centralRegs:  []registryTypes.ImageRegistry{&fakeRegistry{}, &fakeRegistry{}},
		}

		setup(regStore)
		regs := scan.getRegistries("fake", containerImg.GetName())
		suite.Len(regs, 4)
		suite.True(regStore.getRegistryForImageInNamespaceInvoked)
		suite.True(regStore.getGlobalRegistryForImageInvoked)
		suite.True(regStore.getMatchingCentralRegistryIntegrationsInvoked)
	})
}

func successfulScan(_ context.Context, _ *storage.Image,
	reg registryTypes.ImageRegistry, _ scannerclient.Client) (*scannerV1.GetImageComponentsResponse, *scannerV4.IndexReport, error) {

	if reg != nil {
		reg.Config()
	}
	return &scannerV1.GetImageComponentsResponse{
		ScannerVersion: "1",
		Status:         scannerV1.ScanStatus_SUCCEEDED,
		Components: &scannerV1.Components{
			Namespace: "default",
			LanguageComponents: []*scannerV1.LanguageComponent{{
				Type:    scannerV1.SourceType_JAVA,
				Name:    "log4j",
				Version: "1.0",
			}},
		},
	}, nil, nil
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
	_ registryTypes.ImageRegistry, _ scannerclient.Client) (*scannerV1.GetImageComponentsResponse, *scannerV4.IndexReport, error) {
	return nil, nil, errors.New("failed scanning image")
}

func failingFetchSignatures(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image, _ string,
	_ registryTypes.Registry) ([]*storage.Signature, error) {
	return nil, errors.New("failed fetching signatures")
}

func emptyScannerClientSingleton() scannerclient.Client {
	return &scannerclient.GrpcClient{}
}

func emptyGetGlobalRegistryForImage(*storage.ImageName) (registryTypes.ImageRegistry, error) {
	return nil, errors.New("no registry found")
}

func successCreateNoAuthImageRegistry(context.Context, *storage.ImageName, registries.Factory) (registryTypes.ImageRegistry, error) {
	return &fakeRegistry{}, nil
}

func failCreateNoAuthImageRegistry(context.Context, *storage.ImageName, registries.Factory) (registryTypes.ImageRegistry, error) {
	return nil, errors.New("broken")
}

type fakeRegistry struct {
	metadataInvoked bool
	configInvoked   bool
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

func (f *fakeRegistry) Config() *registryTypes.Config {
	f.configInvoked = true
	return nil
}

func (f *fakeRegistry) Name() string {
	return "testing registry"
}

func (f *fakeRegistry) DataSource() *storage.DataSource {
	return nil
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

func (f *fakeRegistryStore) GetRegistryForImageInNamespace(_ *storage.ImageName, _ string) (registryTypes.ImageRegistry, error) {
	f.getRegistryForImageInNamespaceInvoked = true
	if f.namespaceReg != nil {
		return f.namespaceReg, nil
	}
	if f.namespaceNoRegs {
		return nil, errors.New("no regs")
	}
	return &fakeRegistry{}, nil
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
