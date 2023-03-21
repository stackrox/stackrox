package scan

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/sensor/common/scannerclient"
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
	// Use mock functions to avoid having to provide a full registry / scanner.
	scan := LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
			return &fakeRegistry{fail: false}, nil
		},
		scannerClientSingleton: emptyScannerClientSingleton,
	}

	// Original values will be restored within the teardown function. This will be done after each test.

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)

	imageServiceClient := suite.createMockImageServiceClient(img, false)

	resultImg, err := scan.EnrichLocalImage(context.Background(), imageServiceClient, containerImg)

	suite.Require().NoError(err, "unexpected error when enriching image")

	suite.Assert().Equal(img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")
}

func (suite *scanTestSuite) TestEnrichImageFailures() {
	type testCase struct {
		scanImg func(ctx context.Context, image *storage.Image,
			registry registryTypes.Registry, _ *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error)
		fetchSignaturesWithRetry func(ctx context.Context, fetcher signatures.SignatureFetcher, image *storage.Image,
			fullImageName string, registry registryTypes.Registry) ([]*storage.Signature, error)
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
			enrichmentTriggered: true,
		},
		"fail scanning the image locally": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:             failingScan,
			enrichmentTriggered: true,
		},
		"fail enrich image via central": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, true),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
				return &fakeRegistry{fail: false}, nil
			},
			scanImg:                  successfulScan,
			fetchSignaturesWithRetry: successfulFetchSignatures,
			enrichmentTriggered:      true,
		},
		"fail fetching signatures": {
			fakeImageServiceClient: suite.createMockImageServiceClient(nil, false),
			getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
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
		suite.Run(name, func() {
			scan := LocalScan{
				scanImg:                  c.scanImg,
				fetchSignaturesWithRetry: c.fetchSignaturesWithRetry,
				getMatchingRegistry:      c.getMatchingRegistry,
				scannerClientSingleton:   emptyScannerClientSingleton,
			}
			img, err := scan.EnrichLocalImage(context.Background(), c.fakeImageServiceClient, containerImg)
			suite.Assert().Error(err, "expected an error")
			suite.Assert().Nil(img, "required an empty image")
			suite.Assert().Equal(c.enrichmentTriggered, c.fakeImageServiceClient.enrichTriggered,
				"expected enrichment trigger status to be as expected")
		})
	}
}

func (suite *scanTestSuite) TestMetadataBeingSet() {
	scan := LocalScan{
		scanImg: successfulScan,
		fetchSignaturesWithRetry: func(_ context.Context, _ signatures.SignatureFetcher, img *storage.Image, _ string,
			_ registryTypes.Registry) ([]*storage.Signature, error) {
			if img.GetMetadata().GetV2() == nil {
				return nil, errors.New("image metadata missing, not attempting fetch of signatures")
			}
			return nil, nil
		},
		getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
			return &fakeRegistry{fail: false}, nil
		},
		scannerClientSingleton: emptyScannerClientSingleton,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)
	resultImg, err := scan.EnrichLocalImage(context.Background(), imageServiceClient, containerImg)

	suite.Require().NoError(err, "unexpected error when enriching image")

	suite.Assert().Equal(img, resultImg, "resulting image is not equal to expected one")

	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")
}

func (suite *scanTestSuite) TestEnrichLocalImageInNamespace() {
	fakeRegStore := &fakeRegistryStore{}

	scan := LocalScan{
		scanImg:                  successfulScan,
		fetchSignaturesWithRetry: successfulFetchSignatures,
		getMatchingRegistry: func(image *storage.ImageName) (registryTypes.Registry, error) {
			return &fakeRegistry{fail: false}, nil
		},
		scannerClientSingleton: emptyScannerClientSingleton,
		upsertNoAuthRegistry:   fakeRegStore.UpsertNoAuthRegistry,
	}

	containerImg, err := utils.GenerateImageFromString("docker.io/nginx")
	suite.Require().NoError(err, "failed creating test image")

	img := types.ToImage(containerImg)
	imageServiceClient := suite.createMockImageServiceClient(img, false)

	namespace := "fake-namespace"

	// a no-auth upsert should occur on empty regstore
	scan.getRegistryForImageInNamespace = func(image *storage.ImageName, namespace string) (registryTypes.Registry, error) {
		return nil, errors.New("no reg found")
	}
	resultImg, err := scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, containerImg, namespace)
	suite.Require().NoError(err, "unexpected error when enriching image")
	suite.Assert().Equal(img, resultImg, "resulting image is not equal to expected one")
	suite.Assert().True(imageServiceClient.enrichTriggered, "enrichment on central was not triggered")
	suite.Assert().True(fakeRegStore.upsertNoAuthRegistryInvoked, "entry should have been upserted for image")

	// enrichment with matching regstore entry should NOT create a new noauth registry
	fakeRegStore = &fakeRegistryStore{}
	scan.getRegistryForImageInNamespace = func(image *storage.ImageName, namespace string) (registryTypes.Registry, error) {
		return &fakeRegistry{}, nil
	}
	scan.upsertNoAuthRegistry = fakeRegStore.UpsertNoAuthRegistry

	_, err = scan.EnrichLocalImageInNamespace(context.Background(), imageServiceClient, containerImg, namespace)
	suite.Require().NoError(err)
	suite.Assert().False(fakeRegStore.upsertNoAuthRegistryInvoked, "entry should have been upserted for image")
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
	_ registryTypes.Registry, _ *scannerclient.Client) (*scannerV1.GetImageComponentsResponse, error) {
	return nil, errors.New("failed scanning image")
}

func failingFetchSignatures(_ context.Context, _ signatures.SignatureFetcher, _ *storage.Image, _ string,
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

func (f *fakeRegistry) Metadata(_ *storage.Image) (*storage.ImageMetadata, error) {
	if f.fail {
		return nil, errors.New("failed fetching metadata")
	}
	return &storage.ImageMetadata{V2: &storage.V2Metadata{Digest: "sha256:XYZ"}}, nil
}

func (f *fakeRegistry) Name() string {
	return "testing registry"
}

type fakeRegistryStore struct {
	upsertNoAuthRegistryInvoked bool
}

func (f *fakeRegistryStore) UpsertNoAuthRegistry(ctx context.Context, namespace string, imgName *storage.ImageName) (registryTypes.Registry, error) {
	f.upsertNoAuthRegistryInvoked = true
	return &fakeRegistry{}, nil
}
