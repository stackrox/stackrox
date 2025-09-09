package signatures

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Signatures have been generated via cosign:
// $ cosign generate-key-pair
// $ cosign sign -y ttl.sh/88795dd4-270c-4eb7-b3d4-50241d5bc04c@sha256:f2e98ad37e4970f48e85946972ac4acb5574c39f27c624efbd9b17a3a402bfe4 --key=cosign.key
const (
	// pemMatchingPubKey matches the b64Signature.
	pemMatchingPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwz6a8oxByJq9s8kCxvk7RStygmDV
0uXX5qYHbN5sxY8lblhLk9uOr1nFOhNAJua95zL6EwCI2wFykRwqgF1BLg==
-----END PUBLIC KEY-----
`
	// pemNonMatchingPubKey does not match b64Signature.
	pemNonMatchingPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWi3tSxvBH7S/WUmv408nKPxNSJx6
+w7c9FtFSk6coxx2VUbPy/X3US3cXfk/zVA+G7NbXGBYhAGaOsps5ZKjkQ==
-----END PUBLIC KEY-----
`
	// b64Signature is a cosign signature b64 encoded.
	b64Signature = "MEQCIDsFckfIg/uxqSGvf4UC4c9MzAVhuHwzq3NnYovbobYfAiAw31/xz56hS9xRl/xhRV7+RqOl3hXYi7UKO+q7q+" +
		"kOxQ=="
	// b64SignaturePayload is the payload associated with the cosign signature, it references the imgString.
	b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoLzg4Nzk1ZGQ0LTI" +
		"3MGMtNGViNy1iM2Q0LTUwMjQxZDViYzA0YyJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmYyZTk4" +
		"YWQzN2U0OTcwZjQ4ZTg1OTQ2OTcyYWM0YWNiNTU3NGMzOWYyN2M2MjRlZmJkOWIxN2EzYTQwMmJmZTQifSwidHlwZSI6ImNvc2lnb" +
		"iBjb250YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	// imgString points to a temporary available docker image reference, which was used to create the b64signature.
	imgString = "ttl.sh/88795dd4-270c-4eb7-b3d4-50241d5bc04c@sha256:f2e98ad37e4970f48e85946972ac4acb5574c39f27" +
		"c624efbd9b17a3a402bfe4"
)

func TestVerifyAgainstSignatureIntegration(t *testing.T) {
	bundle, err := os.ReadFile("testdata/bundle_bench_test.json")
	require.NoError(t, err)
	testImg, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload, nil, nil, bundle)
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
		result             *storage.ImageSignatureVerificationResult
		verifiedReferences []string
	}{
		"successful verification": {
			integration: &storage.SignatureIntegration{
				Id:     "successful",
				Cosign: successfulCosignConfig,
				TransparencyLog: &storage.TransparencyLogVerification{
					Enabled:         true,
					ValidateOffline: true,
				},
			},
			result: &storage.ImageSignatureVerificationResult{
				VerifierId:              "successful",
				Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
				VerifiedImageReferences: []string{imgString},
			},
		},
		"failing verification": {
			integration: &storage.SignatureIntegration{
				Id:     "failure",
				Cosign: failingCosignConfig,
			},
			result: &storage.ImageSignatureVerificationResult{
				VerifierId:  "failure",
				Status:      storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
				Description: "1 error occurred:",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			result := VerifyAgainstSignatureIntegration(context.Background(), c.integration, testImg)
			assert.Equal(t, c.result.VerifierId, result.VerifierId)
			assert.Equal(t, c.result.Status, result.Status)
			assert.Contains(t, result.Description, c.result.Description)
			assert.ElementsMatch(t, c.result.VerifiedImageReferences, result.VerifiedImageReferences)
		})
	}
}

func BenchmarkVerifyAgainstSignatureIntegrations(b *testing.B) {
	numIntegrations := []int{1, 5, 10, 100}
	numSignatures := []int{1, 5, 10, 100}
	bundle, err := os.ReadFile("testdata/bundle_bench_test.json")
	require.NoError(b, err)
	withBundle := [][]byte{nil, bundle}

	for _, numInt := range numIntegrations {
		for _, numSig := range numSignatures {
			for _, bundle := range withBundle {
				verifyBundle := len(bundle) > 0
				b.Run(fmt.Sprintf("numInt=%d, numSig=%d, verifyBundle=%v", numInt, numSig, verifyBundle), func(b *testing.B) {
					integrations := createSignatureIntegration(numInt, verifyBundle)
					img, err := generateImageWithManySignatures(numSig, imgString, bundle)
					require.NoError(b, err)

					b.ResetTimer()
					benchmarkVerifyAgainstSignatureIntegrations(integrations, img, b)
				})
			}
		}
	}
}

func benchmarkVerifyAgainstSignatureIntegrations(integrations []*storage.SignatureIntegration, img *storage.Image, b *testing.B) {
	for n := 0; n < b.N; n++ {
		VerifyAgainstSignatureIntegrations(context.Background(), integrations, img)
	}
}

func generateImageWithManySignatures(numberOfSignatures int, imgString string, byteBundle []byte) (*storage.Image, error) {
	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload, nil, nil, byteBundle)
	for range numberOfSignatures - 1 {
		img.GetSignature().Signatures = append(img.GetSignature().Signatures, img.GetSignature().Signatures[0])
	}
	return img, err
}

func createSignatureIntegration(numberOfIntegrations int, verifyBundle bool) []*storage.SignatureIntegration {
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
			TransparencyLog: &storage.TransparencyLogVerification{
				Enabled:         verifyBundle,
				ValidateOffline: true,
			},
		})
	}

	return integrations
}
