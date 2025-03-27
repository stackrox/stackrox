package signatures

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// imgName_1a and imgName_1b are both signed by pemPublicKey_1.
	pemPublicKey_1 = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEvVPaOS6sfpySbaceiwg0e6p6Bzo1\n" +
		"A2qyHC5rfVNn2uBRz2Xy0nYBiJ1mFLXaiWGQ2AyCfVh5DzDqJx8eY7YwHQ==\n" +
		"-----END PUBLIC KEY-----"
	imgName_1a = "ttl.sh/2fd430e3-6589-4a5c-865b-4c0aa5b8b041@sha256:298d1eed2893f61ed254c68aff2417deaf342632ab" +
		"26dbbdb3991714e501e1c9"
	b64Signature_1a = "MEUCIDuWppiw7vQzqSRaNA7jx0DtLtWtQvDd/Sm6JfL3nb9uAiEA7bm6pY+GozjTE4hZLPf6ypJqusBNSc9OdC0X" +
		"Mx7sXjk="
	b64SignaturePayload_1a = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoLzJmZDQzMGUzL" +
		"TY1ODktNGE1Yy04NjViLTRjMGFhNWI4YjA0MSJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OjI5OGQxZWVk" +
		"Mjg5M2Y2MWVkMjU0YzY4YWZmMjQxN2RlYWYzNDI2MzJhYjI2ZGJiZGIzOTkxNzE0ZTUwMWUxYzkifSwidHlwZSI6ImNvc2lnbiBjb250YWl" +
		"uZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	b64Signature_1b = "MEQCIHJQ8ijck0xwcFpIru/VW8zhdOadKLuilZwyJGgbAEsaAiB4LdLNIvRJbNaRFhyYTXGFo8cK/VZ7P+Z4Ky+M" +
		"Hk11IA=="
	b64SignaturePayload_1b = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2FjNzExYTA1L" +
		"WZhNjItNDBmYy04ZDNmLWVhM2JjMTk5N2E2NCJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OjI5OGQxZWVk" + "Mjg5M2Y2MWVkMjU0YzY4YWZmMjQxN2RlYWYzNDI2MzJhYjI2ZGJiZGIzOTkxNzE0ZTUwMWUxYzkifSwidHlwZSI6ImNvc2lnbiBjb250YWl" +
		"uZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	imgName_1b = "ttl.sh/ac711a05-fa62-40fc-8d3f-ea3bc1997a64@sha256:298d1eed2893f61ed254c68aff2417deaf342632ab" +
		"26dbbdb3991714e501e1c9"

	// imgName_2 is signed by pemPublicKey_2.
	pemPublicKey_2 = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEKt34iWzgZUoGERCjOhjlxTviXQYk\n" +
		"CxNmhRr5rMeatATTskxx0i6eB04jum2FFPmfIvThIuzyzQxlBDj1gNKXLw==\n" +
		"-----END PUBLIC KEY-----"
	b64Signature_2 = "MEUCIQCoqJSXaMtrHla2iARwMkSiQCadPUgL4y3QQYjW8MLgnAIgC+6233YksZtzLlyMfvHcGUzHg8QoEeKef027" +
		"2n3JN2I="
	b64SignaturePayload_2 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2MwMGRhZjA1L" +
		"TljNWItNDA0MC05ZGJlLTQ0NDgzNjE3MGJlYSJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OjI5OGQxZWVk" +
		"Mjg5M2Y2MWVkMjU0YzY4YWZmMjQxN2RlYWYzNDI2MzJhYjI2ZGJiZGIzOTkxNzE0ZTUwMWUxYzkifSwidHlwZSI6ImNvc2lnbiBjb250YWl" +
		"uZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	imgName_2 = "ttl.sh/c00daf05-9c5b-4040-9dbe-444836170bea@sha256:298d1eed2893f61ed254c68aff2417deaf342632ab" +
		"26dbbdb3991714e501e1c9"

	// imgName_3 is not signed by pemPublicKey_1 or pemPublicKey_2.
	imgName_3 = "ttl.sh/c00daf05-9c5b-4040-9dbe-444836170bea@sha256:298d1eed2893f61ed254c68aff2417deaf342632ab" +
		"26dbbdb3991714e501e1c9"
	b64Signature_3 = "MEUCIQCoqJSXaMtrHla2iARwMkSiQCadPUgL4y3QQYjW8MLgnAIgC+6233YksZtzLlyMfvHcGUzHg8QoEeKef027" +
		"2n3JN2I="
	b64SignaturePayload_3 = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2MwMGRhZjA1L" +
		"TljNWItNDA0MC05ZGJlLTQ0NDgzNjE3MGJlYSJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OjI5OGQxZWVk" +
		"Mjg5M2Y2MWVkMjU0YzY4YWZmMjQxN2RlYWYzNDI2MzJhYjI2ZGJiZGIzOTkxNzE0ZTUwMWUxYzkifSwidHlwZSI6ImNvc2lnbiBjb250YWl" +
		"uZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
)

func TestNewCosignSignatureVerifier(t *testing.T) {
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
			config := &storage.SignatureIntegration{
				Cosign: &storage.CosignPublicKeyVerification{
					PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
						{
							Name:            "pemEncKey",
							PublicKeyPemEnc: c.pemEncKey,
						},
					},
				},
			}
			verifier, err := newCosignSignatureVerifier(config)
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

func TestCosignSignatureVerifier_VerifySignature_Success(t *testing.T) {
	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignSignatureVerifier",
				PublicKeyPemEnc: pemPublicKey_1,
			},
		},
	}})

	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
	require.NoError(t, err, "creating image with signature")

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 1)
	assert.Equal(t, img.GetName().GetFullName(), verifiedImageReferences[0],
		"image full name should match verified image reference")
}

func TestCosignSignatureVerifier_VerifySignature_Multiple_Names_One_Sig(t *testing.T) {
	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignSignatureVerifier",
				PublicKeyPemEnc: pemPublicKey_1,
			},
		},
	}})

	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
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

func TestCosignSignatureVerifier_VerifySignature_Multiple_Names_Multiple_Sigs(t *testing.T) {
	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignSignatureVerifier",
				PublicKeyPemEnc: pemPublicKey_1,
			},
		},
	}})
	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// Use the helper function to create a new image, but we only want the signature here.
	img_2, err := generateImageWithCosignSignature(imgName_1b, b64Signature_1b, b64SignaturePayload_1b, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// `img` must have both names and signatures for the test.
	img.Names = append(img.GetNames(), img_2.GetNames()...)
	img.Signature.Signatures = append(img.GetSignature().GetSignatures(), img_2.GetSignature().GetSignatures()...)

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 2)
	assert.Contains(t, verifiedImageReferences, img.GetName().GetFullName(),
		"image full name should match verified image reference")
}

func TestCosignSignatureVerifier_VerifySignature_Duplicate_Names_Multiple_Sigs(t *testing.T) {
	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignSignatureVerifier",
				PublicKeyPemEnc: pemPublicKey_1,
			},
		},
	}})
	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// Duplicate the names and signature for testing purposes.
	img.Names = append(img.GetNames(), img.GetNames()...)
	img.Signature.Signatures = append(img.GetSignature().GetSignatures(), img.GetSignature().GetSignatures()...)

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 1)
	assert.Contains(t, verifiedImageReferences, img.GetName().GetFullName(),
		"image full name should match verified image reference")
}

func TestCosignSignatureVerifier_VerifySignature_One_Key_Partial_Match(t *testing.T) {
	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignSignatureVerifier",
				PublicKeyPemEnc: pemPublicKey_1,
			},
		},
	}})
	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// Use the helper function to create a new image, but we only want the signature here.
	img_3, err := generateImageWithCosignSignature(imgName_3, b64Signature_3, b64SignaturePayload_3, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// `img` must have both names and signatures for the test.
	img.Names = append(img.GetNames(), img_3.GetNames()...)
	img.Signature.Signatures = append(img.GetSignature().GetSignatures(), img_3.GetSignature().GetSignatures()...)

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 1)
	assert.Contains(t, verifiedImageReferences, img.GetName().GetFullName(),
		"image full name should match verified image reference")
	assert.NotContains(t, verifiedImageReferences, img_3.GetName().GetFullName(),
		"image full name must not match verified image reference")
}

func TestCosignSignatureVerifier_VerifySignature_Multiple_Keys_Full_Match(t *testing.T) {
	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "cosignSignatureVerifier_1",
				PublicKeyPemEnc: pemPublicKey_1,
			},
			{
				Name:            "cosignSignatureVerifier_2",
				PublicKeyPemEnc: pemPublicKey_2,
			},
		},
	}})
	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// Use the helper function to create a new image, but we only want the signature here.
	img_2, err := generateImageWithCosignSignature(imgName_2, b64Signature_2, b64SignaturePayload_2, nil, nil)
	require.NoError(t, err, "creating image with signature")
	// `img` must have both names and signatures for the test.
	img.Names = append(img.GetNames(), img_2.GetNames()...)
	img.Signature.Signatures = append(img.GetSignature().GetSignatures(), img_2.GetSignature().GetSignatures()...)

	status, verifiedImageReferences, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
	require.Len(t, verifiedImageReferences, 2)
	assert.Contains(t, verifiedImageReferences, img.GetName().GetFullName(),
		"image full name should match verified image reference")
	assert.Contains(t, verifiedImageReferences, img_2.GetName().GetFullName(),
		"image full name should match verified image reference")
}

func TestCosignSignatureVerifier_VerifySignature_Failure(t *testing.T) {
	const pemNonMatchingPubKey = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWi3tSxvBH7S/WUmv408nKPxNSJx6\n" +
		"+w7c9FtFSk6coxx2VUbPy/X3US3cXfk/zVA+G7NbXGBYhAGaOsps5ZKjkQ==\n" +
		"-----END PUBLIC KEY-----"

	pubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "Non matching key",
				PublicKeyPemEnc: pemNonMatchingPubKey,
			},
		},
	}})

	require.NoError(t, err, "creating public key verifier")

	emptyPubKeyVerifier, err := newCosignSignatureVerifier(&storage.SignatureIntegration{Cosign: &storage.CosignPublicKeyVerification{}})
	require.NoError(t, err, "creating empty public key verifier")

	img, err := generateImageWithCosignSignature(imgName_1a, b64Signature_1a, b64SignaturePayload_1a, nil, nil)
	require.NoError(t, err, "creating image with signature")

	cases := map[string]struct {
		verifier *cosignSignatureVerifier
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
			err:      errNoVerificationData,
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

func TestCosignVerifier_VerifySignature_Certificate(t *testing.T) {
	const b64Signature = "t18zuH/3IWewBf4EcwjusIvHv5b7jkdtFglPRfdW/oCXweVSDOyX0uVIjolHl2aSRJkJyE182e/" +
		"7ib0V7KtJPm8jvJjUWbB7mgANcoVEEEzNvjYeipOPFT7+fMf1F62torp3fLvK08eU/7i2uuHC+ZDUFSkhK6ZHG8XwI/" +
		"hguWme6fTcJvsO/7F9TlgGni9kJrAnNiFpMxyiP8XQYfqRy2yjuXdmRRmdsVEdiXF4BNfY5tdyaU4LXePYq5KxKRWsQ" +
		"fgqtHATDNqOXV4c3rxq9LxXn/Sl6g1XPT5iKqf8TBwxUl7H/gIV+LFZKRCVhunz1N9cA/4I8ASxe9SOmsH8kQ=="
	const b64Payload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnN" +
		"oLzQ4NTZkNDg1LTg1YjEtNDBkYy1iYTNlLTIzMmU5MzA0OWM1MiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3Q" +
		"tZGlnZXN0Ijoic2hhMjU2OmE5N2ExNTMxNTJmY2Q2NDEwYmRmNGZiNjRmNTYyMmVjZjk3YTc1M2YwN2RjYzg5ZGF" +
		"iMTQ1MDlkMDU5NzM2Y2YifSwidHlwZSI6ImNvc2lnbiBjb250YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGl" +
		"vbmFsIjpudWxsfQ=="
	const imgString = "ttl.sh/4856d485-85b1-40dc-ba3e-232e93049c52@sha256:a97a153152fcd6410bdf4fb64f5622ecf97a753f07dcc89dab14509d059736cf"
	certPEM, err := os.ReadFile("testdata/cert.pem")
	require.NoError(t, err)
	chainPEM, err := os.ReadFile("testdata/chain.pem")
	require.NoError(t, err)

	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64Payload, certPEM, chainPEM)
	require.NoError(t, err, "creating image with signature")

	cases := map[string]struct {
		fail   bool
		status storage.ImageSignatureVerificationResult_Status
		v      func() (*cosignSignatureVerifier, error)
	}{
		"verifying with both cert and chain should work": {
			status: storage.ImageSignatureVerificationResult_VERIFIED,
			v: func() (*cosignSignatureVerifier, error) {
				return newCosignSignatureVerifier(&storage.SignatureIntegration{
					CosignCertificates: []*storage.CosignCertificateVerification{
						{
							CertificatePemEnc:      string(certPEM),
							CertificateChainPemEnc: string(chainPEM),
							CertificateIdentity:    ".*",
							CertificateOidcIssuer:  ".*",
						},
					},
				})
			},
		},
		"verifying with only the chain should work": {
			status: storage.ImageSignatureVerificationResult_VERIFIED,
			v: func() (*cosignSignatureVerifier, error) {
				return newCosignSignatureVerifier(&storage.SignatureIntegration{
					CosignCertificates: []*storage.CosignCertificateVerification{
						{
							CertificateChainPemEnc: string(chainPEM),
							CertificateIdentity:    ".*",
							CertificateOidcIssuer:  ".*",
						},
					},
				})
			},
		},
		"verifying with only the cert should not work due to the wrong chain": {
			status: storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			fail:   true,
			v: func() (*cosignSignatureVerifier, error) {
				return newCosignSignatureVerifier(&storage.SignatureIntegration{
					CosignCertificates: []*storage.CosignCertificateVerification{
						{
							CertificatePemEnc:     string(certPEM),
							CertificateIdentity:   ".*",
							CertificateOidcIssuer: ".*",
						},
					},
				})
			},
		},
		"verifying with only the chain but a mismatch in issuer should fail": {
			status: storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			fail:   true,
			v: func() (*cosignSignatureVerifier, error) {
				return newCosignSignatureVerifier(&storage.SignatureIntegration{
					CosignCertificates: []*storage.CosignCertificateVerification{
						{
							CertificateChainPemEnc: string(chainPEM),
							CertificateOidcIssuer:  "invalid-issuer",
							CertificateIdentity:    "invalid-identity",
						},
					},
				})
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			verifier, err := tc.v()
			require.NoError(t, err)
			status, _, err := verifier.VerifySignature(context.Background(), img)
			if tc.fail {
				assert.Error(t, err)
			}
			assert.Equal(t, tc.status, status)
		})
	}
}

func TestRetrieveVerificationDataFromImage_Success(t *testing.T) {
	//#nosec G101 -- This is a false positive
	const (
		b64CosignSignature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAY" +
			"ZNkn8hkAnXI="
		imgHash                   = "sha256:3a4d57227f02243dfc8a2849ec4a116646bed293b9e93cbf9d4a673a28ef6345"
		b64CosignSignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4O" +
			"TJkLTQ4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODli" +
			"MDA1MjhmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb25" +
			"0YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	)

	img, err := generateImageWithCosignSignature("docker.io/nginx@"+imgHash,
		b64CosignSignature, b64CosignSignaturePayload, nil, nil)
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
			img, err := generateImageWithCosignSignature(
				"docker.io/nginx:latest", "", "", nil, nil)
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

func generateImageWithCosignSignature(imgString, b64Sig, b64SigPayload string,
	certPEM, chainPEM []byte) (*storage.Image, error) {
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
						CertPem:          certPEM,
						CertChainPem:     chainPEM,
					},
				},
			},
		},
	}
	return img, nil
}
