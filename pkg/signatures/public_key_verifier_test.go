package signatures

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPublicKeyVerifier(t *testing.T) {
	cases := map[string]struct {
		pemEncKey string
		fail      bool
		err       error
	}{

		"valid public key": {
			pemEncKey: "-----BEGIN PUBLIC KEY-----\n" +
				"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein\n" +
				"YoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==\n" +
				"-----END PUBLIC KEY-----",
		},
		"non PEM encoded public key": {
			pemEncKey: "anVzdHNvbWV0ZXh0Cg==",
			fail:      true,
			err:       errox.InvariantViolation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := &storage.CosignPublicKeyVerification{
				PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
					{
						Name:            "pemEncKey",
						PublicKeyPemEnc: c.pemEncKey,
					},
				},
			}
			verifier, err := newCosignPublicKeyVerifier(config)
			if c.fail {
				assert.Error(t, err)
				assert.Nil(t, verifier)
				assert.ErrorIs(t, err, c.err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, verifier.parsedPublicKeys, 1)
			}
		})
	}
}

func TestPublicKeyVerifier_VerifySignature_Success(t *testing.T) {
	const pemPublicKey = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein\n" +
		"YoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==\n" +
		"-----END PUBLIC KEY-----"
	const b64Signature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAYZNkn8h" +
		"kAnXI="
	const b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4OTJkLTQ" +
		"4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODliMDA1Mj" +
		"hmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb250YWluZ" +
		"XIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	const imgString = "ttl.sh/d8d3892d-48bd-4671-a546-2e70a900b702@sha256:ee89b00528ff4f02f2405e4ee221743ebc3f8e8dd0" +
		"bfd5c4c20a2fa2aaa7ede3"

	pubKeyVerifier, err := newCosignPublicKeyVerifier(&storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignPublicKeyVerifier",
				PublicKeyPemEnc: pemPublicKey,
			},
		},
	})

	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating image with signature")

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 1)
	assert.Equal(t, img.GetName().GetFullName(), verifiedImageReferences[0],
		"image full name should match verified image reference")
}

func TestPublicKeyVerifier_VerifySignature_Multiple_Names(t *testing.T) {
	const pemPublicKey = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein\n" +
		"YoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==\n" +
		"-----END PUBLIC KEY-----"
	const b64Signature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAYZNkn8h" +
		"kAnXI="
	const b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4OTJkLTQ" +
		"4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODliMDA1Mj" +
		"hmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb250YWluZ" +
		"XIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	const imgString = "ttl.sh/d8d3892d-48bd-4671-a546-2e70a900b702@sha256:ee89b00528ff4f02f2405e4ee221743ebc3f8e8dd0" +
		"bfd5c4c20a2fa2aaa7ede3"

	pubKeyVerifier, err := newCosignPublicKeyVerifier(&storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignPublicKeyVerifier",
				PublicKeyPemEnc: pemPublicKey,
			},
		},
	})

	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating image with signature")

	secondImageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "nginx",
		Tag:      "1.23",
		FullName: "docker.io/nginx:1.23",
	}

	img.Names = append(img.GetNames(), secondImageName)

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 1)
	assert.Contains(t, verifiedImageReferences, img.GetName().GetFullName(),
		"image full name should match verified image reference")
	assert.NotContainsf(t, verifiedImageReferences, secondImageName.GetFullName(),
		"verified image references should not contain image name %s", secondImageName.GetFullName())
}

func TestPublicKeyVerifier_VerifySignature_Failure(t *testing.T) {
	const pemNonMatchingPubKey = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWi3tSxvBH7S/WUmv408nKPxNSJx6\n" +
		"+w7c9FtFSk6coxx2VUbPy/X3US3cXfk/zVA+G7NbXGBYhAGaOsps5ZKjkQ==\n" +
		"-----END PUBLIC KEY-----"
	const b64Signature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAYZNkn8h" +
		"kAnXI="
	const b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4OTJkLTQ" +
		"4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODliMDA1Mj" +
		"hmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb250YWluZ" +
		"XIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	const imgString = "ttl.sh/d8d3892d-48bd-4671-a546-2e70a900b702@sha256:ee89b00528ff4f02f2405e4ee221743ebc3f8e8dd0" +
		"bfd5c4c20a2fa2aaa7ede3"

	pubKeyVerifier, err := newCosignPublicKeyVerifier(&storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "Non matching key",
				PublicKeyPemEnc: pemNonMatchingPubKey,
			},
		},
	})

	require.NoError(t, err, "creating public key verifier")

	emptyPubKeyVerifier, err := newCosignPublicKeyVerifier(&storage.CosignPublicKeyVerification{})
	require.NoError(t, err, "creating empty public key verifier")

	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating image with signature")

	cases := map[string]struct {
		verifier *cosignPublicKeyVerifier
		img      *storage.Image
		err      error
		status   storage.ImageSignatureVerificationResult_Status
	}{
		"fail with non-verifiable public key": {
			img:      img,
			verifier: pubKeyVerifier,
			status:   storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
		},
		"fail with empty public key verifier": {
			img:      img,
			verifier: emptyPubKeyVerifier,
			status:   storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			err:      errNoKeysToVerifyAgainst,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			status, verifiedImageReference, err := c.verifier.VerifySignature(context.Background(), c.img)
			assert.Equal(t, c.status, status, "status should be FAILED verification")
			assert.Error(t, err, "verification should be unsuccessful")
			assert.Empty(t, verifiedImageReference, "verified image reference should be empty")
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			}
		})
	}
}

func TestRetrieveVerificationDataFromImage_Success(t *testing.T) {
	//#nosec G101 -- This is a false positive
	const b64CosignSignature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAY" +
		"ZNkn8hkAnXI="
	const imgHash = "sha256:3a4d57227f02243dfc8a2849ec4a116646bed293b9e93cbf9d4a673a28ef6345"
	const b64CosignSignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4O" +
		"TJkLTQ4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODli" +
		"MDA1MjhmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb25" +
		"0YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="

	img, err := generateImageWithCosignSignature("docker.io/nginx@"+imgHash, b64CosignSignature, b64CosignSignaturePayload)
	require.NoError(t, err, "error creating image")

	sigs, hash, err := retrieveVerificationDataFromImage(img)
	require.NoError(t, err, "should not fail")

	assert.Len(t, sigs, 1, "expected one signature")
	sig := sigs[0]
	b64sig, err := sig.Base64Signature()
	require.NoError(t, err)
	assert.Equal(t, b64CosignSignature, b64sig, "expected the base64 values of the signatures to match")
	payload, err := sig.Payload()
	require.NoError(t, err)
	expectedPayload, err := base64.StdEncoding.DecodeString(b64CosignSignaturePayload)
	require.NoError(t, err)
	assert.Equal(t, expectedPayload, payload, "expected the payloads of the signature to match")
	assert.Equal(t, imgHash, hash.String(), "expected the hash to match the image's hash")
}

func TestRetrieveVerificationDataFromImage_Failure(t *testing.T) {
	cases := map[string]struct {
		imgID string
		err   error
	}{
		"no image SHA": {
			err: errNoImageSHA,
		},
		"hash creation failed": {
			imgID: "invalid-hash",
			err:   errHashCreation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			img, err := generateImageWithCosignSignature("docker.io/nginx:latest", "", "")
			require.NoError(t, err, "error creating image")
			// Since we have no Image Metadata, the Image ID will be returned as SHA. This way we can test for invalid
			// SHA / no SHA.
			img.Id = c.imgID
			_, _, err = retrieveVerificationDataFromImage(img)
			assert.Error(t, err)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestDockerReferenceFromImageName(t *testing.T) {
	cases := map[string]struct {
		name *storage.ImageName
		res  string
	}{
		"shouldn't rewrite registry name for quay.io": {
			name: &storage.ImageName{FullName: "quay.io/some-repo/image:latest"},
			res:  "quay.io/some-repo/image",
		},
		"should rewrite registry name for docker.io": {
			name: &storage.ImageName{FullName: "docker.io/some-repo/image:latest"},
			res:  "index.docker.io/some-repo/image",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := dockerReferenceFromImageName(c.name)
			assert.NoError(t, err)
			assert.Equal(t, c.res, res)
		})
	}
}

func generateImageWithCosignSignature(imgString, b64Sig, b64SigPayload string) (*storage.Image, error) {
	cimg, err := imgUtils.GenerateImageFromString(imgString)
	if err != nil {
		return nil, err
	}

	sigBytes, err := base64.StdEncoding.DecodeString(b64Sig)
	if err != nil {
		return nil, err
	}
	sigPayloadBytes, err := base64.StdEncoding.DecodeString(b64SigPayload)
	if err != nil {
		return nil, err
	}

	img := types.ToImage(cimg)
	img.Signature = &storage.ImageSignature{
		Signatures: []*storage.Signature{
			{
				Signature: &storage.Signature_Cosign{
					Cosign: &storage.CosignSignature{
						RawSignature:     sigBytes,
						SignaturePayload: sigPayloadBytes,
					},
				},
			},
		},
	}
	return img, nil
}
