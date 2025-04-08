package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var (
	goodName                   = "Sheer Heart Attack"
	goodID                     = "io.stackrox.signatureintegration.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13"
	badID                      = "Killer Queen"
	badEmptyCosignConfig       = &storage.CosignPublicKeyVerification{}
	badPEMEncodingCosignConfig = &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				PublicKeyPemEnc: "this is not PEM encoded",
			},
		},
	}
	goodCosignConfig = &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "key name",
				PublicKeyPemEnc: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----",
			},
		},
	}

	badEmptyCosignCertificateVerificationConfig []*storage.CosignCertificateVerification
	badCosignCertificateVerificationConfig      = []*storage.CosignCertificateVerification{
		{
			CertificateChainPemEnc: "@@@",
		},
	}

	goodCosignCertificateVerificationConfig = []*storage.CosignCertificateVerification{
		{
			CertificateChainPemEnc: "-----BEGIN CERTIFICATE-----\nMIIEGDCCAgCgAwIBAgIUJk1lM6fPU8kjiZa5IvXhzK3V/+MwDQYJKoZIhvcNAQEL\nBQAwJDEQMA4GA1UECgwHdGVzdGluZzEQMA4GA1UEAwwHdGVzdGluZzAeFw0yNDA0\nMjEyMjQwNDBaFw0yNTA5MDMyMjQwNDBaMCQxEDAOBgNVBAoMB3Rlc3RpbmcxEDAO\nBgNVBAMMB3Rlc3RpbmcwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDJ\nAch0Q/wXrkivOJ6O9c43EGltWgbd7EBATOQ1SP3J4IUGpCHfSWp7Mnnp1Kfo7fiS\nLrSvBAQgf1+IIOlRR9qb4Dz1zkv5JIi2Bsx3CY5Z7LR49o1h3GwKdNzg/57wOrm6\n8xkWv9Eh/2rV4XU4h2uLR0F3iZvJ+roT4JtMAJRIU/9cUr8zHwfBfiJEbFK+KxZW\nQJMKRg7NYup4vbsDrJplLKUFXdwd5kq7RM059W7VdbWSbsPXONmKGOimLXKew7Vh\n6ZAxxj2dZC88oocvGJmtn4Bg46/LTqU+81ot7fUDeRluoXUIGjUmUDsYWTj9i81k\nk9/0pVKNrRsVd69yxHWPAgMBAAGjQjBAMB0GA1UdDgQWBBT8QfcgemDIYq5sD6tn\n0bFYYq+vlTAfBgNVHSMEGDAWgBSC6V+J82/YA/XnisCVYhB03nh2LDANBgkqhkiG\n9w0BAQsFAAOCAgEAWOr1xyZ+YKaZUAPSCmfA9BwIFACrNnkmm5HiY1lU7Yhs0Xgr\nq9ed115I5ixOk5QR6YlHy3xnC4aNHyPUlxXefIWTELm1s3Ii0Dm7SrAXfM5iyHrG\nYKBpyV320P4udnfBhEVL3kL3xxk23jQJzfAHJCMNLtms1V4XqXun7tMv5tMukCgk\nRC9Y/grAK/1m13KKQNyMoRPqp+qBZmuMSwSliNNpZgb6BhljiyUJ4UZnZr6irRTe\nWu4nnqZtX1qqxrgKuF68f5jBKwOxRIZ0BJaCSaGlLGL4en8CNYd6TAE/OC4P4Zpz\n18EDZAejZ4rS5tDEGtpDpHD2XeCeQACt/joJMTCwmePJEH73VF+ZFywbMNIAMsPR\nmFESys6J7jqoSC9lQPjNoC2KbRnk+PwQ0U7NTJLJETGsWYhFHDwTi7g0Ogmwr09A\nf4vOwI6+qmsckq2K3lob7VmdLhfzVy+u+q6Cg5eBHHyePoO5qtI/Zk9x4paUgMmE\n74MTUknrnOm0GMGHMAyJKqsWZcfGWpLZf2TVFKu1MLcj3wx2Q7TFsqomcZW4Jlez\ntZFwJgN+v0YobsrJlWjcS8vK6hWcMSyHoX+wVvMckaab0ycjTuYpybSuQG5G+002\nI8+UmQtwa0MOOcoUeXXJXjGagodO6A22hzjwQyf5e87eeLA1FfwtGYNLjoA=\n-----END CERTIFICATE-----",
			CertificateOidcIssuer:  ".*",
			CertificateIdentity:    ".*",
		},
	}

	invalidCosignCertificateVerificationConfig = []*storage.CosignCertificateVerification{
		{
			CertificateOidcIssuer: "a(b",
			CertificateIdentity:   "a(b",
		},
	}

	invalidCTLogPublicKeyConfig = []*storage.CosignCertificateVerification{
		{
			CertificateTransparencyLog: &storage.CertificateTransparencyLogVerification{
				Enabled:         true,
				PublicKeyPemEnc: "@@@",
			},
		},
	}

	invalidRekorPublicKeyConfig = &storage.TransparencyLogVerification{
		Enabled:         true,
		PublicKeyPemEnc: "@@@",
	}

	invalidRekorURLConfig = &storage.TransparencyLogVerification{
		Enabled: true,
		Url:     "invalid-url",
	}
)

func TestValidateSignatureIntegration_Failure(t *testing.T) {
	testCasesBad := map[string]*storage.SignatureIntegration{
		"name field must be set": {
			Id:     goodID,
			Cosign: goodCosignConfig,
		},
		"id must follow format": {
			Id:     badID,
			Name:   goodName,
			Cosign: goodCosignConfig,
		},
		"id field must be set": {
			Name:   goodName,
			Cosign: goodCosignConfig,
		},
		"at least one signature verification config should be present": {
			Id:   goodID,
			Name: goodName,
		},
		"at least one public key in cosign config should be present": {
			Id:     goodID,
			Name:   goodName,
			Cosign: badEmptyCosignConfig,
		},
		"public keys in cosign config should be PEM-encoded": {
			Id:     goodID,
			Name:   goodName,
			Cosign: badPEMEncodingCosignConfig,
		},
		"at least one certificate verification config should be present": {
			Id:                 goodID,
			Name:               goodName,
			CosignCertificates: badEmptyCosignCertificateVerificationConfig,
		},
		"certificates in the config should be PEM encoded and required fields filled": {
			Id:                 goodID,
			Name:               goodName,
			CosignCertificates: badCosignCertificateVerificationConfig,
		},
		"invalid regexp for identity and issuer": {
			Id:                 goodID,
			Name:               goodName,
			CosignCertificates: invalidCosignCertificateVerificationConfig,
		},
		"invalid ctlog public key": {
			Id:                 goodID,
			Name:               goodName,
			CosignCertificates: invalidCTLogPublicKeyConfig,
		},
		"invalid rekor public key": {
			Id:              goodID,
			Name:            goodName,
			TransparencyLog: invalidRekorPublicKeyConfig,
		},
		"invalid rekor url": {
			Id:              goodID,
			Name:            goodName,
			TransparencyLog: invalidRekorURLConfig,
		},
	}

	for desc, signatureIntegration := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSignatureIntegration(signatureIntegration)
			assert.Errorf(t, err, "signature integration: '%+v'", signatureIntegration)
		})
	}
}

func TestValidateSignatureIntegration_Success(t *testing.T) {
	testCasesGood := map[string]*storage.SignatureIntegration{
		"valid name, id, and cosign config": {
			Id:     goodID,
			Name:   goodName,
			Cosign: goodCosignConfig,
		},
		"valid name, id, and cosign certificate config": {
			Id:                 goodID,
			Name:               goodName,
			CosignCertificates: goodCosignCertificateVerificationConfig,
		},
	}

	for desc, signatureIntegration := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSignatureIntegration(signatureIntegration)
			assert.NoErrorf(t, err, "signature integration: '%+v'", signatureIntegration)
		})
	}
}
