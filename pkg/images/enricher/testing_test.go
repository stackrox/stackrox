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
	"google.golang.org/protobuf/proto"
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
	si := &storage.SignatureIntegration{}
	si.SetId("id-1")
	si.SetName("name-1")
	si2 := &storage.SignatureIntegration{}
	si2.SetId("id-2")
	si2.SetName("name-2")
	return []*storage.SignatureIntegration{
		si,
		si2,
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
	return storage.ImageScan_builder{
		Components: []*storage.EmbeddedImageScanComponent{
			storage.EmbeddedImageScanComponent_builder{
				Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2020-1234",
					}.Build(),
				},
			}.Build(),
		},
	}.Build(), nil
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
	ds := &storage.DataSource{}
	ds.SetId("id")
	ds.SetName(f.Name())
	return ds
}

func (f *fakeRegistryScanner) Source() *storage.ImageIntegration {
	ii := &storage.ImageIntegration{}
	ii.SetId("id")
	ii.SetName(f.Name())
	return ii
}

type fakeCVESuppressor struct{}

func (f *fakeCVESuppressor) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.GetCve() == "CVE-2020-1234" {
				v.SetSuppressed(true)
			}
		}
	}
}

func (f *fakeCVESuppressor) EnrichImageV2WithSuppressedCVEs(image *storage.ImageV2) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.GetCve() == "CVE-2020-1234" {
				v.SetSuppressed(true)
			}
		}
	}
}

type fakeCVESuppressorV2 struct{}

func (f *fakeCVESuppressorV2) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.GetCve() == "CVE-2020-1234" {
				v.SetState(storage.VulnerabilityState_DEFERRED)
			}
		}
	}
}

func (f *fakeCVESuppressorV2) EnrichImageV2WithSuppressedCVEs(image *storage.ImageV2) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.GetCve() == "CVE-2020-1234" {
				v.SetState(storage.VulnerabilityState_DEFERRED)
			}
		}
	}
}

func newCache() cache.ImageMetadata {
	return cache.ImageMetadata(expiringcache.NewExpiringCache[string, *storage.ImageMetadata](1 * time.Minute))
}

func createSignature(sig, payload string) *storage.Signature {
	cs := &storage.CosignSignature{}
	cs.SetRawSignature([]byte(sig))
	cs.SetSignaturePayload([]byte(payload))
	signature := &storage.Signature{}
	signature.SetCosign(proto.ValueOrDefault(cs))
	return signature
}

func createSignatureVerificationResult(verifier string, status storage.ImageSignatureVerificationResult_Status,
	verifiedImageNames ...string) *storage.ImageSignatureVerificationResult {
	isvr := &storage.ImageSignatureVerificationResult{}
	isvr.SetVerifierId(verifier)
	isvr.SetStatus(status)
	isvr.SetVerifiedImageReferences(verifiedImageNames)
	return isvr
}

func fakeSignatureIntegrationGetter(id string, fail bool) SignatureIntegrationGetter {
	return func(ctx context.Context) ([]*storage.SignatureIntegration, error) {
		if fail {
			return nil, errors.New("fake error")
		}
		si := &storage.SignatureIntegration{}
		si.SetId(id)
		si.SetName(id)
		return []*storage.SignatureIntegration{
			si,
		}, nil
	}
}
