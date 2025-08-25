package enricher

import (
	"context"
	"errors"
	"net/http"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/cache"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	scannertypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/signatures"
	"golang.org/x/sync/semaphore"
)

var (
	// emptyCtx used within all tests.
	emptyCtx = context.Background()

	// errBroken is a generic error.
	errBroken = errors.New("broken")
)

func emptySignatureIntegrationGetter(_ context.Context) ([]*storage.SignatureIntegration, error) {
	return nil, nil
}

func defaultRedHatSignatureIntegrationGetter(_ context.Context) ([]*storage.SignatureIntegration, error) {
	return []*storage.SignatureIntegration{signatures.DefaultRedHatSignatureIntegration}, nil
}

func twoSignaturesIntegrationGetter(_ context.Context) ([]*storage.SignatureIntegration, error) {
	return []*storage.SignatureIntegration{
		{
			Id:   "id-1",
			Name: "name-1",
		},
		{
			Id:   "id-2",
			Name: "name-2",
		},
	}, nil
}

type fakeSigFetcher struct {
	sigs      []*storage.Signature
	fail      bool
	retryable bool
}

func (f *fakeSigFetcher) FetchSignatures(_ context.Context, _ *storage.Image, _ string,
	_ types.Registry) ([]*storage.Signature, error) {
	if f.fail {
		err := errors.New("some error")
		if f.retryable {
			err = retry.MakeRetryable(err)
		}
		return nil, err
	}
	return f.sigs, nil
}

type fakeScanner struct {
	requestedScan bool
	notMatch      bool
}

func (*fakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
	f.requestedScan = true
	return &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve: "CVE-2020-1234",
					},
				},
			},
		},
	}, nil
}

func (f *fakeScanner) Match(*storage.ImageName) bool {
	return !f.notMatch
}

func (*fakeScanner) Test() error {
	return nil
}

func (*fakeScanner) Type() string {
	return "type"
}

func (*fakeScanner) Name() string {
	return "name"
}

func (*fakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

type fakeRegistryScanner struct {
	scanner           *fakeScanner
	requestedMetadata bool
	notMatch          bool
}

type opts struct {
	requestedScan     bool
	requestedMetadata bool
	notMatch          bool
}

func newFakeRegistryScanner(opts opts) *fakeRegistryScanner {
	return &fakeRegistryScanner{
		scanner: &fakeScanner{
			requestedScan: opts.requestedScan,
			notMatch:      opts.notMatch,
		},
		requestedMetadata: opts.requestedMetadata,
		notMatch:          opts.notMatch,
	}
}

func (f *fakeRegistryScanner) Metadata(*storage.Image) (*storage.ImageMetadata, error) {
	f.requestedMetadata = true
	return &storage.ImageMetadata{}, nil
}

func (f *fakeRegistryScanner) Config(_ context.Context) *types.Config {
	return nil
}

func (f *fakeRegistryScanner) Match(*storage.ImageName) bool {
	return !f.notMatch
}

func (*fakeRegistryScanner) Test() error {
	return nil
}

func (*fakeRegistryScanner) Type() string {
	return "type"
}

func (*fakeRegistryScanner) Name() string {
	return "name"
}

func (*fakeRegistryScanner) HTTPClient() *http.Client {
	return nil
}

func (f *fakeRegistryScanner) GetScanner() scannertypes.Scanner {
	return f.scanner
}

func (f *fakeRegistryScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{
		Id:   "id",
		Name: f.Name(),
	}
}

func (f *fakeRegistryScanner) Source() *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:   "id",
		Name: f.Name(),
	}
}

type fakeCVESuppressor struct{}

func (f *fakeCVESuppressor) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.Cve == "CVE-2020-1234" {
				v.Suppressed = true
			}
		}
	}
}

func (f *fakeCVESuppressor) EnrichImageV2WithSuppressedCVEs(image *storage.ImageV2) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.Cve == "CVE-2020-1234" {
				v.Suppressed = true
			}
		}
	}
}

type fakeCVESuppressorV2 struct{}

func (f *fakeCVESuppressorV2) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.Cve == "CVE-2020-1234" {
				v.State = storage.VulnerabilityState_DEFERRED
			}
		}
	}
}

func (f *fakeCVESuppressorV2) EnrichImageV2WithSuppressedCVEs(image *storage.ImageV2) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.Cve == "CVE-2020-1234" {
				v.State = storage.VulnerabilityState_DEFERRED
			}
		}
	}
}

func newCache() cache.ImageMetadata {
	return cache.ImageMetadata(expiringcache.NewExpiringCache[string, *storage.ImageMetadata](1 * time.Minute))
}

func createSignature(sig, payload string) *storage.Signature {
	return &storage.Signature{Signature: &storage.Signature_Cosign{
		Cosign: &storage.CosignSignature{
			RawSignature:     []byte(sig),
			SignaturePayload: []byte(payload),
		},
	}}
}

func createSignatureVerificationResult(verifier string, status storage.ImageSignatureVerificationResult_Status,
	verifiedImageNames ...string) *storage.ImageSignatureVerificationResult {
	return &storage.ImageSignatureVerificationResult{
		VerifierId:              verifier,
		Status:                  status,
		VerifiedImageReferences: verifiedImageNames,
	}
}

func fakeSignatureIntegrationGetter(id string, fail bool) SignatureIntegrationGetter {
	return func(ctx context.Context) ([]*storage.SignatureIntegration, error) {
		if fail {
			return nil, errors.New("fake error")
		}
		return []*storage.SignatureIntegration{
			{
				Id:   id,
				Name: id,
			},
		}, nil
	}
}
