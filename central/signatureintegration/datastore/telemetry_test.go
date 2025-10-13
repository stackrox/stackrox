package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGather(t *testing.T) {
	assert.Empty(t, computeTelemetryProperties(context.Background(), []*storage.SignatureIntegration{}))

	props := computeTelemetryProperties(context.Background(), []*storage.SignatureIntegration{
		{},
		{
			Cosign:          goodCosignConfig,
			TransparencyLog: &storage.TransparencyLogVerification{Enabled: true, ValidateOffline: true},
		},
		{
			CosignCertificates: goodCosignCertificateVerificationConfig,
			TransparencyLog:    &storage.TransparencyLogVerification{Enabled: true, Url: "https://custom.rekor"},
		},
		{
			Cosign:             goodCosignConfig,
			CosignCertificates: goodCosignCertificateVerificationConfig,
			TransparencyLog:    &storage.TransparencyLogVerification{Enabled: true, Url: "https://rekor.sigstore.dev"},
		},
		{
			Cosign: goodCosignConfig,
			CosignCertificates: []*storage.CosignCertificateVerification{
				{
					CertificatePemEnc: "===",
				},
			},
		},
	})

	expectedProps := map[string]any{
		"Total Signature Integration Certificates":                                 3,
		"Total Signature Integration Cosign Public Keys":                           3,
		"Total Signature Integration With Certificate Transparency Log Validation": 2,
		"Total Signature Integration With Custom Certificate":                      1,
		"Total Signature Integration With Custom Chain":                            2,
		"Total Signature Integration With Transparency Log Custom Rekor URL":       1,
		"Total Signature Integration With Transparency Log Offline Validation":     1,
		"Total Signature Integration With Transparency Log Validation":             3,
		"Total Signature Integrations":                                             5,
	}
	assert.Equal(t, expectedProps, props)
}
