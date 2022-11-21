package signatures

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// pemMatchingPubKey matches the b64Signature.
	pemMatchingPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein
YoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==
-----END PUBLIC KEY-----
`
	// pemNonMatchingPubKey does not match b64Signature.
	pemNonMatchingPubKey = `-----BEGIN PUBLIC KEY-----
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

func TestVerifyAgainstSignatureIntegration(t *testing.T) {
	testImg, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating test image")

	successfulCosignConfig := &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				PublicKeyPemEnc: pemMatchingPubKey,
			},
		},
	}

	failingCosignConfig := &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				PublicKeyPemEnc: pemNonMatchingPubKey,
			},
		},
	}

	cases := map[string]struct {
		integration        *storage.SignatureIntegration
		results            []storage.ImageSignatureVerificationResult
		verifiedReferences []string
	}{
		"successful verification": {
			integration: &storage.SignatureIntegration{
				Id:     "successful",
				Cosign: successfulCosignConfig,
			},
			results: []storage.ImageSignatureVerificationResult{
				{
					VerifierId:              "successful",
					Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
					VerifiedImageReferences: []string{imgString},
				},
			},
		},
		"failing verification": {
			integration: &storage.SignatureIntegration{
				Id:     "failure",
				Cosign: failingCosignConfig,
			},
			results: []storage.ImageSignatureVerificationResult{
				{
					VerifierId:  "failure",
					Status:      storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
					Description: "1 error occurred:",
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
				assert.Contains(t, results[i].Description, res.Description)
				assert.ElementsMatch(t, results[i].VerifiedImageReferences, res.VerifiedImageReferences)
			}
		})
	}
}

func BenchmarkVerifyAgainstSignatureIntegrations_1Integration(b *testing.B) {
	integrations := createSignatureIntegration(1)
	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(b, err)

	b.ResetTimer()
	benchmarkVerifyAgainstSignatureIntegrations(integrations, img, b)
}

func BenchmarkVerifyAgainstSignatureIntegrations_10Integrations(b *testing.B) {
	integrations := createSignatureIntegration(10)
	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(b, err)

	b.ResetTimer()
	benchmarkVerifyAgainstSignatureIntegrations(integrations, img, b)
}

func BenchmarkVerifyAgainstSignatureIntegrations_100Integrations(b *testing.B) {
	integrations := createSignatureIntegration(100)
	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(b, err)

	b.ResetTimer()
	benchmarkVerifyAgainstSignatureIntegrations(integrations, img, b)
}

func BenchmarkVerifyAgainstSignatureIntegrations_200Integrations(b *testing.B) {
	integrations := createSignatureIntegration(200)
	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(b, err)

	b.ResetTimer()
	benchmarkVerifyAgainstSignatureIntegrations(integrations, img, b)
}

func benchmarkVerifyAgainstSignatureIntegrations(integrations []*storage.SignatureIntegration, img *storage.Image, b *testing.B) {
	for n := 0; n < b.N; n++ {
		VerifyAgainstSignatureIntegrations(context.Background(), integrations, img)
	}
}

func createSignatureIntegration(numberOfIntegrations int) []*storage.SignatureIntegration {
	successfulCosignConfig := &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				PublicKeyPemEnc: pemMatchingPubKey,
			},
		},
	}

	failingCosignConfig := &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				PublicKeyPemEnc: pemNonMatchingPubKey,
			},
		},
	}

	integrations := make([]*storage.SignatureIntegration, 0, numberOfIntegrations)

	for i := 0; i < numberOfIntegrations; i++ {
		var cosignConfig *storage.CosignPublicKeyVerification
		if i%2 == 0 {
			cosignConfig = successfulCosignConfig
		} else {
			cosignConfig = failingCosignConfig
		}

		integrations = append(integrations, &storage.SignatureIntegration{
			Id:     fmt.Sprintf("sig-integration-%d", i),
			Cosign: cosignConfig,
		})
	}

	return integrations
}
