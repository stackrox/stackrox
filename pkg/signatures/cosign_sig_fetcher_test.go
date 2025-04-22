package signatures

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	stdLog "log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sig1 = "MEUCIFcvJdlCG8a36z5FMfxBnT/B9k3iK7T7oc8S2FOxy6B0AiEAjWG0eBCzogfG8gXTwLm9DXWe4RgwYA8dPNZnOBA6LhQ="
	sig2 = "MEYCIQCNA5wSnIrvNBv5Irg8s4ptMzaSJWfEYALLm45iliHzfgIhAOrFxQpJ0FEuR9sanCIbKWE4Y7DSCTvUbTMZPpMpcvnI"
	// Signature signed with cert and cert chain.
	sig3 = "B0YxA+VVwKaDt8LHlJ9U+tjBl6MM+V5DG6wwgLRPeNybWC/cbHHJK0aN+y/e1RxQ5e4OJzszRi3LJgyz81B7s/" +
		"r72buMhuN+vVWb0EBu8PZbJwaMF8vgBHAslhQnTp6pTE9jgOjKTRl9jYjRaMG2SiuCMZmE" +
		"yA5A0K7BntXwPwFqDNI1otReqFZr6BrgDFpzhJ5l2od/u+6KrcvIFT/7R1TnnpF87peJXP" +
		"Nwxd6kyuhNCZw0YcP31X41YD+PDb3AntnydyXlI7Arzr66ib5dnZB/DefDNFibFIgwLIRU" +
		"CSVxi8WM1FUbpsnNFr1tMsy+UUGgkYzfjmqnMq4epvVGQA=="
	// Signature signed using Fulcio and Rekor. Proof of inclusion is in `bundle.json`.
	sig4 = "MEQCID1AIJK4aJYJpbBVaFmoQfC4wMTJnBW/hsaCQobGoiBqAiBcfLS81tDBMhyON0jZpizle8XIzYWE2wou4CCPilVlBA=="

	payload1 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2E1NDIzODZlLTFiMjItNDQ5" +
		"ZC1hYWZkLWIxMjMzZjFhOWEzYyJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmJiMTI5YTcxMmM" +
		"yNDMxZWNjZTRhZjhkZGU4MzFlOTgwMzczYjI2MzY4MjMzZWYwZjNiMmJhZTllOWVjNTE1ZWUifSwidHlwZSI6ImNvc2lnbiBjb2" +
		"50YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	payload2 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2M4M2MyMTIzLTQ5ZGItNGM2" +
		"ZC1hM2Q2LWI5Y2JlNGQ3YjE3NCJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmJiMTI5YTcxMmM" +
		"yNDMxZWNjZTRhZjhkZGU4MzFlOTgwMzczYjI2MzY4MjMzZWYwZjNiMmJhZTllOWVjNTE1ZWUifSwidHlwZSI6ImNvc2lnbiBjb2" +
		"50YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	payload3 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZX" +
		"ItcmVmZXJlbmNlIjoidGVzdC5pby90ZXN0aW5nIn0sImltYWdlI" +
		"jp7ImRvY2tlci1tYW5pZmVzdC1kaWdlc3QiOiJzaGEyNTY6NmFk" +
		"ODM5NGFkMzFiMjY5YjU2MzU2Njk5OGZkODBhOGYyNTllOGRlY2Y" +
		"xNmU4MDdmODMxMGVjYzEwYzY4NzM4NSJ9LCJ0eXBlIjoiY29zaW" +
		"duIGNvbnRhaW5lciBpbWFnZSBzaWduYXR1cmUifSwib3B0aW9uY" +
		"WwiOm51bGx9Cg=="
	payload4 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoLzRkM2VjYTdmLWQyNGUtNDI5N" +
		"C1hYzExLWZhNzE0Y2U4Nzc5YiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OjBlMWFjN2YxMmQ5M" +
		"DRhNWNlMDc3ZDFiNWM3NjNiNTc1MGM3OTg1ZTUyNGY2MDgzZTVlYWE3ZTczMTM4MzM0NDAifSwidHlwZSI6ImNvc2lnbiBjb250Y" +
		"WluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
)

type mockRegistry struct {
	registryTypes.ImageRegistry
	cfg *registryTypes.Config
}

func (m *mockRegistry) Config(_ context.Context) *registryTypes.Config {
	return m.cfg
}

func (m *mockRegistry) Name() string {
	return "mock registry"
}

// registryServerWithImage creates a local registry that can be accessed via a httptest.Server during tests with an
// image pushed.
func registryServerWithImage(imgName string) (*httptest.Server, string, error) {
	nopLog := stdLog.New(io.Discard, "", 0)
	reg := registry.New(registry.Logger(nopLog))
	srv := httptest.NewServer(reg)
	imgFullName := fmt.Sprintf("%s/%s", srv.Listener.Addr().String(), imgName)
	image, err := random.Image(1024, 1)
	if err != nil {
		return nil, "", err
	}
	err = crane.Push(image, imgFullName)
	if err != nil {
		return nil, "", err
	}

	digest, err := image.Digest()
	if err != nil {
		return nil, "", err
	}

	imgWithDigest := fmt.Sprintf("%s@%s:%s", imgFullName, digest.Algorithm, digest.Hex)

	return srv, imgWithDigest, nil
}

// uploadSignatureForImage will upload the given signature and payload for the specified image reference.
func uploadSignatureForImage(imgRef string, b64Sig string, sigPayload, certPEM, chainPEM, byteBundle []byte) error {
	sigOpts := []static.Option{static.WithCertChain(certPEM, chainPEM)}

	rekorBundle, err := unmarshalRekorBundle(byteBundle)
	if err != nil {
		return err
	}
	if rekorBundle != nil {
		sigOpts = append(sigOpts, static.WithBundle(rekorBundle))
	}

	sig, err := static.NewSignature(sigPayload, b64Sig, sigOpts...)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(imgRef)
	if err != nil {
		return err
	}
	d, ok := ref.(name.Digest)
	if !ok {
		return fmt.Errorf("could not cast reference %q to name.Digest", ref.String())
	}

	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		return err
	}

	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return err
	}

	err = ociremote.WriteSignatures(d.Repository, newSE)
	if err != nil {
		return err
	}
	return nil
}

func TestCosignSignatureFetcher_FetchSignature_Success(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	cimg, err := imgUtils.GenerateImageFromString(imgRef)
	require.NoError(t, err, "creating test image")
	img := types.ToImage(cimg)
	img.Metadata = &storage.ImageMetadata{V2: &storage.V2Metadata{Digest: "something"}}

	rawSig1, err := base64.StdEncoding.DecodeString(sig1)
	require.NoError(t, err, "decoding signature")
	rawSig2, err := base64.StdEncoding.DecodeString(sig2)
	require.NoError(t, err, "decoding signature")
	rawSig3, err := base64.StdEncoding.DecodeString(sig3)
	require.NoError(t, err, "decoding signature")
	rawSig4, err := base64.StdEncoding.DecodeString(sig4)
	require.NoError(t, err, "decoding signature")
	sigPayload1, err := base64.StdEncoding.DecodeString(payload1)
	require.NoError(t, err, "decoding signature")
	sigPayload2, err := base64.StdEncoding.DecodeString(payload2)
	require.NoError(t, err, "decoding signature")
	sigPayload3, err := base64.StdEncoding.DecodeString(payload3)
	require.NoError(t, err, "decoding signature")
	sigPayload4, err := base64.StdEncoding.DecodeString(payload4)
	require.NoError(t, err, "decoding signature")

	certPEM, err := os.ReadFile("testdata/cert.pem")
	require.NoError(t, err)
	chainPEM, err := os.ReadFile("testdata/chain.pem")
	require.NoError(t, err)

	certPEMWithTlog, err := os.ReadFile("testdata/cert_with_tlog.pem")
	require.NoError(t, err)
	chainPEMWithTlog, err := os.ReadFile("testdata/chain_with_tlog.pem")
	require.NoError(t, err)

	bundle, err := os.ReadFile("testdata/bundle.json")
	require.NoError(t, err)

	require.NoError(t, uploadSignatureForImage(imgRef, sig1, sigPayload1, nil, nil, nil),
		"uploading signature")
	require.NoError(t, uploadSignatureForImage(imgRef, sig2, sigPayload2, nil, nil, nil),
		"uploading signature")
	require.NoError(t, uploadSignatureForImage(imgRef, sig3, sigPayload3, certPEM, chainPEM, nil),
		"uploading signature")
	require.NoError(t, uploadSignatureForImage(imgRef, sig4, sigPayload4, certPEMWithTlog, chainPEMWithTlog, bundle),
		"uploading signature")

	expectedSignatures := []*storage.Signature{
		{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig1,
					SignaturePayload: sigPayload1,
				},
			},
		},
		{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig2,
					SignaturePayload: sigPayload2,
				},
			},
		},
		{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig3,
					SignaturePayload: sigPayload3,
					CertPem:          certPEM,
					CertChainPem:     chainPEM,
				},
			},
		},
		{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig4,
					SignaturePayload: sigPayload4,
					CertPem:          certPEMWithTlog,
					CertChainPem:     chainPEMWithTlog,
					RekorBundle:      bundle,
				},
			},
		},
	}

	f := newCosignSignatureFetcher()
	mockConfig := &registryTypes.Config{
		Username: "",
		Password: "",
		Insecure: false,
	}
	reg := &mockRegistry{cfg: mockConfig}

	res, err := f.FetchSignatures(context.Background(), img, img.GetName().GetFullName(), reg)
	assert.NoError(t, err)
	protoassert.SlicesEqual(t, expectedSignatures, res)
}

func TestCosignSignatureFetcher_FetchSignature_Failure(t *testing.T) {
	registryServer, _, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	f := newCosignSignatureFetcher()

	cimg, err := imgUtils.GenerateImageFromString("nginx")
	require.NoError(t, err, "creating test image")

	// Fail with a non-retryable error when an image is given with a wrong reference.
	cimg.Name.FullName = "fa@wrongreference"
	img := types.ToImage(cimg)
	img.Metadata = &storage.ImageMetadata{V2: &storage.V2Metadata{Digest: "something"}}
	res, err := f.FetchSignatures(context.Background(), img, img.GetName().GetFullName(), nil)
	assert.Nil(t, res)
	require.Error(t, err)
	assert.False(t, retry.IsRetryable(err))
}

func TestCosignSignatureFetcher_FetchSignature_NoSignature(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	cimg, err := imgUtils.GenerateImageFromString(imgRef)
	require.NoError(t, err, "creating test image")
	img := types.ToImage(cimg)

	f := newCosignSignatureFetcher()
	reg := &mockRegistry{cfg: &registryTypes.Config{}}

	require.NoError(t, err)
	result, err := f.FetchSignatures(context.Background(), img, img.GetName().GetFullName(), reg)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestIsMissingSignatureError(t *testing.T) {
	notFoundErr := dockerRegistry.HttpStatusError{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	unauthorizedErr := dockerRegistry.HttpStatusError{
		Response: &http.Response{
			StatusCode: http.StatusUnauthorized,
		},
	}

	emptyResponseErr := dockerRegistry.HttpStatusError{
		Response: &http.Response{},
	}

	cases := map[string]struct {
		err              error
		missingSignature bool
	}{
		"cosign error for missing signatures should indicate missing signature": {
			err:              errors.New("no signatures associated with test image"),
			missingSignature: true,
		},
		"registry error with status code not found should indicate missing signature": {
			err: errors.Wrap(&url.Error{
				Err: &notFoundErr,
			}, "something went wrong"),
			missingSignature: true,
		},
		"registry error without response should not indicate missing signature": {
			err: errors.Wrap(&url.Error{
				Err: &emptyResponseErr,
			}, "something went wrong"),
		},
		"registry error with status code unauthorized should not indicate missing signature": {
			err: errors.Wrap(&url.Error{
				Err: &unauthorizedErr,
			}, "something went wrong"),
		},
		"status error with status code not found should indicate missing signature": {
			err:              &notFoundErr,
			missingSignature: true,
		},
		"status error without response should not indicate missing signature": {
			err: &emptyResponseErr,
		},
		"status error with status code unauthorized should not indicate missing signature": {
			err: &unauthorizedErr,
		},
		"wrapped registry error with status code not found should indicate missing signature": {
			err:              fmt.Errorf("something went wrong %w", &url.Error{Err: &notFoundErr}),
			missingSignature: true,
		},
		"wrapped registry error with status code unauthorized should not indicate missing signature": {
			err: fmt.Errorf("something went wrong %w", &url.Error{Err: &unauthorizedErr}),
		},
		"transport error with status code unauthorized should not indicate missing signature": {
			err: &transport.Error{
				StatusCode: http.StatusUnauthorized,
			},
		},
		"transport error with status code not found should indicate missing signature": {
			err: &transport.Error{
				StatusCode: http.StatusNotFound,
			},
			missingSignature: true,
		},
		"neither registry nor cosign error": {
			err: errors.New("something error"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.missingSignature, isMissingSignatureError(c.err))
		})
	}
}

func TestIsUnauthorizedError(t *testing.T) {
	notFoundErr := dockerRegistry.HttpStatusError{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	unauthorizedErr := dockerRegistry.HttpStatusError{
		Response: &http.Response{
			StatusCode: http.StatusUnauthorized,
		},
	}
	forbiddenErr := dockerRegistry.HttpStatusError{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
		},
	}

	emptyResponseErr := dockerRegistry.HttpStatusError{}

	cases := map[string]struct {
		err               error
		unauthorizedError bool
	}{
		"registry error with status code not found should not indicate unauthorized error": {
			err: errors.Wrap(&url.Error{
				Err: &notFoundErr,
			}, "something went wrong"),
		},
		"registry error without response should not indicate unauthorized error": {
			err: errors.Wrap(&url.Error{
				Err: &emptyResponseErr,
			}, "something went wrong"),
		},
		"registry error with status code unauthorized should indicate unauthorized error": {
			err: errors.Wrap(&url.Error{
				Err: &unauthorizedErr,
			}, "something went wrong"),
			unauthorizedError: true,
		},
		"registry error with status code forbidden should indicate unauthorized error": {
			err: errors.Wrap(&url.Error{
				Err: &forbiddenErr,
			}, "something went wrong"),
			unauthorizedError: true,
		},
		"status error without response should not indicate unauthorized error": {
			err: &emptyResponseErr,
		},
		"status error with status code unauthorized should indicate unauthorized error": {
			err:               &unauthorizedErr,
			unauthorizedError: true,
		},
		"status error with status code forbidden should indicate unauthorized error": {
			err:               &forbiddenErr,
			unauthorizedError: true,
		},
		"transport error with status code unauthorized should indicate unauthorized error": {
			err: &transport.Error{
				StatusCode: http.StatusUnauthorized,
			},
			unauthorizedError: true,
		},
		"transport error with status code not found should not indicate unauthorized error": {
			err: &transport.Error{
				StatusCode: http.StatusNotFound,
			},
		},
		"transport error with status code forbidden should indicate unauthorized error": {
			err: &transport.Error{
				StatusCode: http.StatusForbidden,
			},
			unauthorizedError: true,
		},
		"neither transport nor registry error should not indicate unauthorized error": {
			err: errors.New("some random error"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.unauthorizedError, isUnauthorizedError(c.err))
		})
	}
}

func TestOptionsFromRegistry(t *testing.T) {
	cases := map[string]struct {
		registry             registryTypes.Registry
		expectedNumOfOptions int
	}{
		"empty config settings should not lead to options": {
			registry: &mockRegistry{cfg: &registryTypes.Config{}},
		},
		"only username set should not create options": {
			registry: &mockRegistry{cfg: &registryTypes.Config{Username: "test"}},
		},
		"username + password set should create options": {
			registry:             &mockRegistry{cfg: &registryTypes.Config{Username: "test", Password: "test"}},
			expectedNumOfOptions: 1,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Len(t, optionsFromRegistry(context.Background(), c.registry), c.expectedNumOfOptions)
		})
	}
}

func TestIsUnknownMimeTypeError(t *testing.T) {
	cases := map[string]struct {
		err         error
		expectedRes bool
	}{
		"should indicate unknown mime type error when error contains unknown mime type": {
			err:         errors.New("unknown mime type: application/vnd.docker.distribution.manifest.v1+prettyjws"),
			expectedRes: true,
		},
		"should not indicate unknown mime type error when error does not contain unknown mime type": {
			err: errors.New("some other error"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expectedRes, isUnknownMimeTypeError(c.err))
		})
	}
}
