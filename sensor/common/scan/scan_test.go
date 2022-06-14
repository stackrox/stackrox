package scan

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/images/types"
	"github.com/stackrox/stackrox/pkg/images/utils"
	registryTypes "github.com/stackrox/stackrox/pkg/registries/types"
	"github.com/stackrox/stackrox/pkg/signatures"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stackrox/stackrox/sensor/common/scannerclient"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type fakeImageServiceClient struct {
	v1.ImageServiceClient
	fail bool
	img  *storage.Image
	// Used to check if enrichment on central's side was triggered or not.
	enrichTriggered bool
}

func (i *fakeImageServiceClient) EnrichLocalImageInternal(ctx context.Context,
	req *v1.EnrichLocalImageInternalRequest, _ ...grpc.CallOption) (*v1.ScanImageInternalResponse, error) {
	i.enrichTriggered = true
	if i.fail {
		return nil, errors.New("failed enrichment")
	}
	return &v1.ScanImageInternalResponse{Image: i.img}, nil
}

type scanTestSuite struct {
	suite.Suite
	env                      *envisolator.EnvIsolator
	fetchSignaturesWithRetry func(ctx context.Context, fetcher signatures.SignatureFetcher, image *storage.Image,
		registry registryTypes.Registry) ([]*storage.Signature, error)
	getMatchingRegistry    func(image *storage.ImageName) (registryTypes.Registry, error)
	scannerClientSingleton func() *scannerclient.Client
	scanImg                func(ctx context.Context, image *storage.Image, registry registryTypes.Registry,
		scannerClient *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error)
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

func (suite *scanTestSuite) SetupSuite() {
	suite.env = envisolator.NewEnvIsolator(suite.T())
	suite.env.Setenv("ROX_VERIFY_IMAGE_SIGNATURE", "true")
	suite.fetchSignaturesWithRetry = fetchSignaturesWithRetry
	suite.getMatchingRegistry = getMatchingRegistry
	suite.scannerClientSingleton = scannerClientSingleton
	suite.scanImg = scanImg
}

func (suite *scanTestSuite) TearDownSuite() {
	suite.env.RestoreAll()
}

func (suite *scanTestSuite) AfterTest(_, _ string) {
	scanImg = suite.scanImg
	fetchSignaturesWithRetry = suite.fetchSignaturesWithRetry
	getMatchingRegistry = suite.getMatchingRegistry
	scannerClientSingleton = suite.scannerClientSingleton
}

func (suite *scanTestSuite) TestLocalEnrichment() {
	// Use mock functions to avoid having to provide a full registry / scanner.
	scanImg = successfulScan
	fetchSignaturesWithRetry = successfulFetchSignatures
	getMatchingRegistry = func(image *storage.ImageName) (registryTypes.Registry, error) {
		return &fakeRegistry{fail: false}, nil
	}
	scannerClientSingleton = emptyScannerClientSingleton

	// Original values will be restored within the teardown function. This will be done after each test.

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)

	imageServiceClient := suite.createMockImageServiceClient(img, false)

	resultImg, err := EnrichLocalImage(context.Background(), imageServiceClient, containerImg)

	suite.Require().NoError(err, "unexpected error when enriching image")

	suite.Assert().Equal(img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")
}

func (suite *scanTestSuite) TestEnrichImageFailures() {
	type testCase struct {
		scanImg func(ctx context.Context, image *storage.Image,
			registry registryTypes.Registry, _ *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error)
		fetchSignaturesWithRetry func(ctx context.Context, fetcher signatures.SignatureFetcher, image *storage.Image,
			registry registryTypes.Registry) ([]*storage.Signature, error)
		getMatchingRegistry    func(image *storage.ImageName) (registryTypes.Registry, error)
		fakeImageServiceClient *fakeImageServiceClient
		enrichmentTriggered    bool
	}

	cases := map[string]testCase{
		"fail getting a matching registry": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return nil, errors.New("image doesn't match any registry")
			},
		},
		"fail retrieving image metadata": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return &fakeRegistry{fail: true}, nil
			},
		},
		"fail scanning the image locally": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg: failingScan,
		},
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	// This will allow us to test this only if the env variable is set. For release builds, this will skip this case
	// until we have deprecated this feature flag.
	if features.ImageSignatureVerification.Enabled() {
		cases["fail enrich image via central"] = testCase{
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, true),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:                  successfulScan,
			fetchSignaturesWithRetry: successfulFetchSignatures,
			enrichmentTriggered:      true,
		}
		cases["fail fetching signatures"] = testCase{
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:                  successfulScan,
			fetchSignaturesWithRetry: failingFetchSignatures,
		}
	}

	for name, c := range cases {
		suite.Run(name, func() {
			scanImg = c.scanImg
			fetchSignaturesWithRetry = c.fetchSignaturesWithRetry
			getMatchingRegistry = c.getMatchingRegistry
			scannerClientSingleton = emptyScannerClientSingleton
			// Need to manually trigger after test here, otherwise it would only be called at the end of table tests.
			defer suite.AfterTest("", "")
			img, err := EnrichLocalImage(context.Background(), c.fakeImageServiceClient, containerImg)
			suite.Assert().Error(err, "expected an error")
			suite.Assert().Nil(img, "required an empty image")
			suite.Assert().Equal(c.enrichmentTriggered, c.fakeImageServiceClient.enrichTriggered,
				"expected enrichment trigger status to be as expected")
		})
	}
}

func successfulScan(_ context.Context, _ *storage.Image,
	_ registryTypes.Registry, _ *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error) {
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
	}, nil
}

func successfulFetchSignatures(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image,
	_ registryTypes.Registry) ([]*storage.Signature, error) {
	return []*storage.Signature{{
		Signature: &storage.Signature_Cosign{Cosign: &storage.CosignSignature{
			RawSignature:     []byte("some-signature"),
			SignaturePayload: []byte("some-payload"),
		}},
	}}, nil
}

func failingScan(_ context.Context, _ *storage.Image,
	_ registryTypes.Registry, _ *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error) {
	return nil, errors.New("failed scanning image")
}

func failingFetchSignatures(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image,
	_ registryTypes.Registry) ([]*storage.Signature, error) {
	return nil, errors.New("failed fetching signatures")
}

func emptyScannerClientSingleton() *scannerclient.Client {
	return &scannerclient.Client{}
}

type fakeRegistry struct {
	registryTypes.Registry
	fail bool
}

func (f *fakeRegistry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	if f.fail {
		return nil, errors.New("failed fetching metadata")
	}
	return nil, nil
}

func (f *fakeRegistry) Name() string {
	return "testing registry"
}
