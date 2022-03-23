package signatures

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	stdLog "log"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/sigstore/cosign/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sigstore/cosign/pkg/oci/static"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	sig1 = "MEUCIFcvJdlCG8a36z5FMfxBnT/B9k3iK7T7oc8S2FOxy6B0AiEAjWG0eBCzogfG8gXTwLm9DXWe4RgwYA8dPNZnOBA6LhQ="
	sig2 = "MEYCIQCNA5wSnIrvNBv5Irg8s4ptMzaSJWfEYALLm45iliHzfgIhAOrFxQpJ0FEuR9sanCIbKWE4Y7DSCTvUbTMZPpMpcvnI"

	payload1 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2E1NDIzODZlLTFiMjItNDQ5" +
		"ZC1hYWZkLWIxMjMzZjFhOWEzYyJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmJiMTI5YTcxMmM" +
		"yNDMxZWNjZTRhZjhkZGU4MzFlOTgwMzczYjI2MzY4MjMzZWYwZjNiMmJhZTllOWVjNTE1ZWUifSwidHlwZSI6ImNvc2lnbiBjb2" +
		"50YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	payload2 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2M4M2MyMTIzLTQ5ZGItNGM2" +
		"ZC1hM2Q2LWI5Y2JlNGQ3YjE3NCJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmJiMTI5YTcxMmM" +
		"yNDMxZWNjZTRhZjhkZGU4MzFlOTgwMzczYjI2MzY4MjMzZWYwZjNiMmJhZTllOWVjNTE1ZWUifSwidHlwZSI6ImNvc2lnbiBjb2" +
		"50YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
)

type mockRegistry struct {
	registryTypes.ImageRegistry
	cfg *registryTypes.Config
}

func (m *mockRegistry) Config() *registryTypes.Config {
	return m.cfg
}

type mockLogSink struct {
	io.Writer
}

func (cw mockLogSink) Close() error {
	return nil
}

func (cw mockLogSink) Sync() error {
	return nil
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
func uploadSignatureForImage(imgRef string, b64Sig string, sigPayload []byte) error {
	sig, err := static.NewSignature(sigPayload, b64Sig)
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

func TestPublicKey_FetchSignature_Success(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	cimg, err := imgUtils.GenerateImageFromString(imgRef)
	require.NoError(t, err, "creating test image")
	img := types.ToImage(cimg)

	rawSig1, err := base64.StdEncoding.DecodeString(sig1)
	require.NoError(t, err, "decoding signature")
	rawSig2, err := base64.StdEncoding.DecodeString(sig2)
	require.NoError(t, err, "decoding signature")
	sigPayload1, err := base64.StdEncoding.DecodeString(payload1)
	require.NoError(t, err, "decoding signature")
	sigPayload2, err := base64.StdEncoding.DecodeString(payload2)
	require.NoError(t, err, "decoding signature")

	require.NoError(t, uploadSignatureForImage(imgRef, sig1, sigPayload1), "uploading signature")
	require.NoError(t, uploadSignatureForImage(imgRef, sig2, sigPayload2), "uploading signature")

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
	}

	f := &cosignPublicKeySignatureFetcher{}
	mockConfig := &registryTypes.Config{
		Username: "",
		Password: "",
		Insecure: false,
	}
	reg := &mockRegistry{cfg: mockConfig}

	res, err := f.FetchSignatures(context.Background(), img, reg)
	assert.NoError(t, err)
	assert.Equal(t, expectedSignatures, res)
}

func TestPublicKey_FetchSignature_Failure(t *testing.T) {
	registryServer, _, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	cases := map[string]struct {
		registry registryTypes.ImageRegistry
		img      string
	}{
		"non-existing repository": {
			registry: &mockRegistry{cfg: &registryTypes.Config{}},
			img:      fmt.Sprintf("%s/%s", registryServer.Listener.Addr().String(), "some/private/repo"),
		},
		"failed parse reference": {
			img: "fa@wrongreference",
		},
	}
	f := &cosignPublicKeySignatureFetcher{}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cimg, err := imgUtils.GenerateImageFromString("nginx")
			require.NoError(t, err, "creating test image")
			cimg.Name.FullName = c.img
			img := types.ToImage(cimg)
			res, err := f.FetchSignatures(context.Background(), img, c.registry)
			assert.Nil(t, res)
			require.Error(t, err)
			assert.False(t, retry.IsRetryable(err))
		})
	}
}

func TestPublicKey_FetchSignature_NoSignature(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	cimg, err := imgUtils.GenerateImageFromString(imgRef)
	require.NoError(t, err, "creating test image")
	img := types.ToImage(cimg)

	f := &cosignPublicKeySignatureFetcher{}
	reg := &mockRegistry{cfg: &registryTypes.Config{}}

	// Create a mock logger to check that no error is written to the log.
	var output bytes.Buffer
	bWriter := bufio.NewWriter(&output)
	cfg := zap.NewProductionConfig()
	err = zap.RegisterSink("customwriter", func(url *url.URL) (zap.Sink, error) {
		return mockLogSink{bWriter}, nil
	})
	require.NoError(t, err)
	customPath := fmt.Sprintf("%s:whatever", "customwriter")
	cfg.OutputPaths = []string{customPath}
	l, err := cfg.Build()
	require.NoError(t, err)
	logger := &logging.Logger{
		SugaredLogger: l.Sugar(),
	}

	stdLogger := log
	log = logger
	defer func() { log = stdLogger }()

	result, err := f.FetchSignatures(context.Background(), img, reg)
	assert.NoError(t, err)
	assert.Nil(t, result)
	require.NoError(t, bWriter.Flush(), "writing log output")
	assert.Empty(t, output.String())
}
