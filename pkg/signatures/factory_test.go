package signatures

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyAgainstSignatureIntegration(t *testing.T) {
	const (
		// b64MatchingPubKey matches the b64Signature.
		b64MatchingPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein
YoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==
-----END PUBLIC KEY-----
`
		// b64NonMatchingPubKey does not match b64Signature.
		b64NonMatchingPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWi3tSxvBH7S/WUmv408nKPxNSJx6
+w7c9FtFSk6coxx2VUbPy/X3US3cXfk/zVA+G7NbXGBYhAGaOsps5ZKjkQ==
-----END PUBLIC KEY-----
`
		// b64Signature is a cosign signature b64 encoded.
		b64Signature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAYZNkn8h" +
			"kAnXI="
		// b64SignaturePayload is the payload associated with the cosign signature, it references the imgString.
		b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4OTJkLTQ" +
			"4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODli" +
			"MDA1MjhmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnb" +
			"iBjb250YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
		// imgString points to a temporary available docker image reference, which was used to create the b64signature.
		imgString = "ttl.sh/d8d3892d-48bd-4671-a546-2e70a900b702@sha256:ee89b00528ff4f02f2405e4ee221743ebc3f8e8dd0" +
			"bfd5c4c20a2fa2aaa7ede3"
	)

	testImg, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating test image")

	successfulCosignConfig := &storage.SignatureVerificationConfig{
		Config: &storage.SignatureVerificationConfig_CosignVerification{
			CosignVerification: &storage.CosignPublicKeyVerification{
				PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
					{
						PublicKeyPemEnc: b64MatchingPubKey,
					},
				},
			},
		},
	}

	failingCosignConfig := &storage.SignatureVerificationConfig{
		Config: &storage.SignatureVerificationConfig_CosignVerification{
			CosignVerification: &storage.CosignPublicKeyVerification{
				PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
					{
						PublicKeyPemEnc: b64NonMatchingPubKey,
					},
				},
			},
		},
	}

	cases := map[string]struct {
		integration *storage.SignatureIntegration
		results     []storage.ImageSignatureVerificationResult
	}{
		"successful verification": {
			integration: &storage.SignatureIntegration{
				Id: "successful",
				SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
					successfulCosignConfig,
				},
			},
			results: []storage.ImageSignatureVerificationResult{
				{
					VerifierId: "successful",
					Status:     storage.ImageSignatureVerificationResult_VERIFIED,
				},
			},
		},
		"failing verification": {
			integration: &storage.SignatureIntegration{
				Id: "failure",
				SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
					failingCosignConfig,
				},
			},
			results: []storage.ImageSignatureVerificationResult{
				{
					VerifierId:  "failure",
					Status:      storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
					Description: "1 error occurred:\n\t* failed to verify signature\n\n",
				},
			},
		},
		"mix of failing and successful verification": {
			integration: &storage.SignatureIntegration{
				Id: "success-failure",
				SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
					failingCosignConfig, successfulCosignConfig,
				},
			},
			results: []storage.ImageSignatureVerificationResult{
				{
					VerifierId: "success-failure",
					Status:     storage.ImageSignatureVerificationResult_VERIFIED,
				},
			},
		},
		"multiple failing verification results": {
			integration: &storage.SignatureIntegration{
				Id: "multiple-failures",
				SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
					failingCosignConfig, failingCosignConfig,
				},
			},
			results: []storage.ImageSignatureVerificationResult{
				{
					VerifierId:  "multiple-failures",
					Status:      storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
					Description: "1 error occurred:\n\t* failed to verify signature\n\n",
				},
				{
					VerifierId:  "multiple-failures",
					Status:      storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
					Description: "1 error occurred:\n\t* failed to verify signature\n\n",
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			results := VerifyAgainstSignatureIntegration(context.Background(), c.integration, testImg)
			require.Len(t, results, len(c.results))
			for i, res := range c.results {
				assert.Equal(t, res.VerifierId, results[i].VerifierId)
				assert.Equal(t, res.Status, results[i].Status)
				assert.Equal(t, res.Description, results[i].Description)
			}
		})
	}
}
