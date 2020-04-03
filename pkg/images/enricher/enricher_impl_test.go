package enricher

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/integration/mocks"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	mocks2 "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	mocks3 "github.com/stackrox/rox/pkg/scanners/mocks"
	types2 "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

type fakeRegistryScanner struct {
	requestedMetadata bool
	requestedScan     bool
}

func (f *fakeRegistryScanner) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	f.requestedMetadata = true
	return &storage.ImageMetadata{}, nil
}

func (f *fakeRegistryScanner) Global() bool {
	return true
}

func (f *fakeRegistryScanner) Config() *types.Config {
	return nil
}

func (f *fakeRegistryScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeRegistryScanner) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	f.requestedScan = true
	return &storage.ImageScan{}, nil
}

func (f *fakeRegistryScanner) Match(image *storage.ImageName) bool {
	return true
}

func (f *fakeRegistryScanner) Test() error {
	return nil
}

func (f *fakeRegistryScanner) Type() string {
	return "type"
}

func (f *fakeRegistryScanner) Name() string {
	return "name"
}

func TestEnricherFlow(t *testing.T) {
	cases := []struct {
		name            string
		ctx             EnrichmentContext
		inMetadataCache bool
		inScanCache     bool
		image           *storage.Image

		fsr    *fakeRegistryScanner
		result EnrichmentResult
	}{
		{
			name: "nothing in the cache",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache: false,
			inScanCache:     false,
			image:           &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: true,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "scan and metadata in both caches",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache: true,
			inScanCache:     true,
			image:           &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in both caches, but force refetch",
			ctx: EnrichmentContext{
				FetchOpt: ForceRefetch,
			},
			inMetadataCache: true,
			inScanCache:     true,
			image:           &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: true,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in both caches but force refetch scans only",
			ctx: EnrichmentContext{
				FetchOpt: ForceRefetchScansOnly,
			},
			inMetadataCache: true,
			inScanCache:     true,
			image:           &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data not in caches, and no external metadata",
			ctx: EnrichmentContext{
				FetchOpt: NoExternalMetadata,
			},
			inMetadataCache: false,
			inScanCache:     false,
			image:           &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: false,
				ScanResult:   ScanNotDone,
			},
		},
		{
			name: "data not in cache, but iamge already has metadata and scan",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache: false,
			inScanCache:     false,
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
			},
			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: false,
				ScanResult:   ScanNotDone,
			},
		},
		{
			name: "data not in cache and ignore existing images",
			ctx: EnrichmentContext{
				FetchOpt: IgnoreExistingImages,
			},
			inMetadataCache: false,
			inScanCache:     false,
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
			},
			fsr: &fakeRegistryScanner{
				requestedMetadata: true,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in cache and ignore existing images",
			ctx: EnrichmentContext{
				FetchOpt: IgnoreExistingImages,
			},
			inMetadataCache: true,
			inScanCache:     true,
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
			},
			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fsr := &fakeRegistryScanner{}
			registrySet := mocks2.NewMockSet(ctrl)
			registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr})

			scannerSet := mocks3.NewMockSet(ctrl)
			scannerSet.EXPECT().GetAll().Return([]types2.ImageScanner{fsr})

			set := mocks.NewMockSet(ctrl)
			set.EXPECT().RegistrySet().Return(registrySet)
			set.EXPECT().ScannerSet().Return(scannerSet)

			enricherImpl := &enricherImpl{
				integrations:    set,
				metadataLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
				metadataCache:   expiringcache.NewExpiringCache(1 * time.Minute),
				scanCache:       expiringcache.NewExpiringCache(1 * time.Minute),
				metrics:         newMetrics(pkgMetrics.CentralSubsystem),
			}

			if c.inMetadataCache {
				enricherImpl.metadataCache.Add(c.image.GetId(), c.image.GetMetadata())
			}
			if c.inScanCache {
				enricherImpl.scanCache.Add(c.image.GetId(), c.image.GetScan())
			}
			result, err := enricherImpl.EnrichImage(c.ctx, c.image)
			require.NoError(t, err)
			assert.Equal(t, c.result, result)

			assert.Equal(t, c.fsr, fsr)
		})
	}
}

func TestFillScanStats(t *testing.T) {
	cases := []struct {
		image                *storage.Image
		expectedVulns        int32
		expectedFixableVulns int32
	}{
		{
			image: &storage.Image{
				Id: "image-1",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
								},
							},
						},
					},
				},
			},
			expectedVulns:        1,
			expectedFixableVulns: 1,
		},
		{
			image: &storage.Image{
				Id: "image-1",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
					},
				},
			},
			expectedVulns:        2,
			expectedFixableVulns: 2,
		},
		{
			image: &storage.Image{
				Id: "image-1",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-3",
								},
							},
						},
					},
				},
			},
			expectedVulns:        3,
			expectedFixableVulns: 0,
		},
	}

	for _, c := range cases {
		t.Run(t.Name(), func(t *testing.T) {
			FillScanStats(c.image)
			assert.Equal(t, c.expectedVulns, c.image.GetCves())
			assert.Equal(t, c.expectedFixableVulns, c.image.GetFixableCves())
		})
	}
}
