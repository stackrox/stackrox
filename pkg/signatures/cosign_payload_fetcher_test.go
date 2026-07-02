package signatures

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/v3/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v3/pkg/oci/remote"
	"github.com/sigstore/cosign/v3/pkg/oci/static"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadSignatureAsReferrer stores a cosign signature as an OCI 1.1 referrer for the given image.
func uploadSignatureAsReferrer(imgRef string, b64Sig string, sigPayload, certPEM, chainPEM []byte) error {
	sigOpts := []static.Option{static.WithCertChain(certPEM, chainPEM)}

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

	return ociremote.WriteSignaturesExperimentalOCI(d, newSE)
}

func TestFetchTagPayloads_WithSignatures(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	sigPayload, err := base64.StdEncoding.DecodeString(payload1)
	require.NoError(t, err)

	require.NoError(t, uploadSignatureForImage(imgRef, sig1, sigPayload, nil, nil, nil),
		"uploading tag-based signature")

	ref, err := name.ParseReference(imgRef)
	require.NoError(t, err)
	img, err := imgUtils.GenerateImageFromString(imgRef)
	require.NoError(t, err)

	payloads, err := fetchTagPayloads(types.ToImage(img), ref, nil)
	require.NoError(t, err)
	assert.Len(t, payloads, 1)
	assert.Equal(t, sigPayload, payloads[0].Payload)
	assert.Equal(t, sig1, payloads[0].Base64Signature)
}

func TestFetchTagPayloads_NoSignatures(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	ref, err := name.ParseReference(imgRef)
	require.NoError(t, err)
	img, err := imgUtils.GenerateImageFromString(imgRef)
	require.NoError(t, err)

	payloads, err := fetchTagPayloads(types.ToImage(img), ref, nil)
	require.NoError(t, err)
	assert.Empty(t, payloads)
}

func TestFetchReferrerPayloads_WithSignatures(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	sigPayload, err := base64.StdEncoding.DecodeString(payload1)
	require.NoError(t, err)

	require.NoError(t, uploadSignatureAsReferrer(imgRef, sig1, sigPayload, nil, nil),
		"uploading referrer-based signature")

	ref, err := name.ParseReference(imgRef)
	require.NoError(t, err)
	d, ok := ref.(name.Digest)
	require.True(t, ok)

	payloads, err := fetchReferrerPayloads(context.Background(), d, ref.Context(), nil)
	require.NoError(t, err)
	assert.Len(t, payloads, 1)
	assert.Equal(t, sigPayload, payloads[0].Payload)
	assert.Equal(t, sig1, payloads[0].Base64Signature)
}

func TestFetchReferrerPayloads_NoSignatures(t *testing.T) {
	registryServer, imgRef, err := registryServerWithImage("nginx")
	require.NoError(t, err, "setting up registry")
	defer registryServer.Close()

	ref, err := name.ParseReference(imgRef)
	require.NoError(t, err)
	d, ok := ref.(name.Digest)
	require.True(t, ok)

	payloads, err := fetchReferrerPayloads(context.Background(), d, ref.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, payloads)
}
