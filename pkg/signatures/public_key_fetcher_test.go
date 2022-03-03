package signatures

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	stdLog "log"
	"net/http/httptest"
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
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// registryServerWithImage creates a local registry that can be accessed via a httptest.Server during tests with an
// image pushed.
func registryServerWithImage(imgName string) (*httptest.Server, string, error) {
	nopLog := stdLog.New(ioutil.Discard, "", 0)
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

type mockRegistry struct {
	registryTypes.ImageRegistry
	cfg *registryTypes.Config
}

func (m *mockRegistry) Config() *registryTypes.Config {
	return m.cfg
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

	expectedSignatures := &storage.ImageSignature{
		Signatures: []*storage.Signature{
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
		},
	}

	f := &cosignPublicKeyFetcher{}
	mockConfig := &registryTypes.Config{
		Username: "",
		Password: "",
		Insecure: false,
	}
	reg := &mockRegistry{cfg: mockConfig}

	res, exists := f.FetchSignature(context.Background(), img, reg)
	assert.True(t, exists)
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
	f := &cosignPublicKeyFetcher{}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cimg, err := imgUtils.GenerateImageFromString("nginx")
			require.NoError(t, err, "creating test image")
			cimg.Name.FullName = c.img
			img := types.ToImage(cimg)
			res, exists := f.FetchSignature(context.Background(), img, c.registry)
			assert.False(t, exists)
			assert.Nil(t, res)
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

	f := &cosignPublicKeyFetcher{}
	reg := &mockRegistry{cfg: &registryTypes.Config{}}

	result, exists := f.FetchSignature(context.Background(), img, reg)
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, result)
}
